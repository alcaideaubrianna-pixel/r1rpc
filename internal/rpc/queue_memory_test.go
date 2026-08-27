package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type queueTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func newQueueTestClock(now time.Time) *queueTestClock {
	return &queueTestClock{now: now}
}

func (c *queueTestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *queueTestClock) Set(now time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

func TestMemoryJobQueueLifecycleAndTrace(t *testing.T) {
	queue := newMemoryJobQueue(2)
	job := &Job{RequestID: "request-1", Group: "xhs", Action: "content.search_notes", ClientID: "device-1"}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if job.JobID != JobID(job.RequestID) || job.QueueID != queue.ID() {
		t.Fatalf("identity not assigned: job=%q queue=%q", job.JobID, job.QueueID)
	}
	dequeued, err := queue.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if dequeued == job || dequeued.RequestID != job.RequestID {
		t.Fatalf("dequeue must return an equal copy: got=%+v want=%+v", dequeued, job)
	}
	if err := queue.Requeue(job, "send_failed"); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	events := queue.Events()
	want := []QueueEventType{QueueEventQueued, QueueEventDequeued, QueueEventRequeued}
	if len(events) != len(want) {
		t.Fatalf("events=%d want=%d", len(events), len(want))
	}
	for index, eventType := range want {
		if events[index].Type != eventType {
			t.Fatalf("event[%d]=%q want=%q", index, events[index].Type, eventType)
		}
		if events[index].RequestID != job.RequestID || events[index].Action != job.Action {
			t.Fatalf("event metadata mismatch: %+v", events[index])
		}
	}
}

func TestMemoryJobQueueCapacityAndExpiryAreExplicit(t *testing.T) {
	queue := newMemoryJobQueue(1)
	first := &Job{RequestID: "first"}
	prepareJobForQueue(queue.ID(), first)
	if err := queue.Enqueue(first); err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	second := &Job{RequestID: "second"}
	prepareJobForQueue(queue.ID(), second)
	if err := queue.Enqueue(second); !errors.Is(err, ErrClientQueueFull) {
		t.Fatalf("capacity error=%v", err)
	}
	expired := &Job{RequestID: "expired", DeadlineAt: time.Now().Add(-time.Second)}
	prepareJobForQueue(queue.ID(), expired)
	if err := queue.Enqueue(expired); !errors.Is(err, ErrJobExpired) {
		t.Fatalf("expiry error=%v", err)
	}
}

func TestMemoryJobQueueDequeueCancellation(t *testing.T) {
	queue := newMemoryJobQueue(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := queue.Dequeue(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("dequeue error=%v", err)
	}
}

func TestMemoryJobQueueRenewExtendsSameReservationAtomically(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	queue := newMemoryJobQueueWithNow(1, clock.Now)
	job := &Job{RequestID: "renew", DeadlineAt: start.Add(5 * time.Second)}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), 2*time.Second)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	clock.Set(start.Add(time.Second))
	renewed, err := queue.Renew(delivery, 10*time.Second)
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if renewed.LeaseID != delivery.LeaseID || !renewed.LeaseUntil.Equal(start.Add(11*time.Second)) {
		t.Fatalf("renewed delivery=%+v", renewed)
	}
	clock.Set(start.Add(3 * time.Second))
	if err := queue.Ack(renewed); err != nil {
		t.Fatalf("ack renewed delivery: %v", err)
	}
	if queue.Len() != 0 {
		t.Fatalf("queue depth=%d", queue.Len())
	}
}

func TestMemoryJobQueueRenewExpiredReservationRequeuesOnce(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	queue := newMemoryJobQueueWithNow(1, clock.Now)
	job := &Job{RequestID: "expired-renew", DeadlineAt: start.Add(time.Minute)}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	clock.Set(delivery.LeaseUntil)
	if _, err := queue.Renew(delivery, time.Second); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("renew expired error=%v", err)
	}
	retried, err := queue.Reserve(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("reserve retry: %v", err)
	}
	if retried.Attempt != delivery.Attempt+1 || retried.JobID != delivery.JobID {
		t.Fatalf("retried delivery=%+v original=%+v", retried, delivery)
	}
}

func TestMemoryJobQueueTraceIsBounded(t *testing.T) {
	queue := newMemoryJobQueue(1)
	queue.traceLimit = 3
	for index := 0; index < 4; index++ {
		job := &Job{RequestID: newOpaqueID("request_")}
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		if _, err := queue.Dequeue(context.Background()); err != nil {
			t.Fatalf("dequeue: %v", err)
		}
	}
	if events := queue.Events(); len(events) != queue.traceLimit {
		t.Fatalf("events=%d limit=%d", len(events), queue.traceLimit)
	}
}

func TestMemoryJobQueueRejectsCrossQueueRequeue(t *testing.T) {
	first := newMemoryJobQueue(1)
	second := newMemoryJobQueue(1)
	job := &Job{RequestID: "request-1"}
	prepareJobForQueue(first.ID(), job)
	if err := first.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := second.Requeue(job, "wrong_owner"); !errors.Is(err, ErrQueueIdentityMismatch) {
		t.Fatalf("cross queue error=%v", err)
	}
}

func TestMemoryJobQueueConcurrentCapacity(t *testing.T) {
	const capacity = 16
	queue := newMemoryJobQueue(capacity)
	var wait sync.WaitGroup
	var accepted int
	var acceptedMu sync.Mutex
	for index := 0; index < capacity*4; index++ {
		job := &Job{RequestID: newOpaqueID("request_")}
		prepareJobForQueue(queue.ID(), job)
		wait.Add(1)
		go func(job *Job) {
			defer wait.Done()
			if err := queue.Enqueue(job); err == nil {
				acceptedMu.Lock()
				accepted++
				acceptedMu.Unlock()
			} else if !errors.Is(err, ErrClientQueueFull) {
				t.Errorf("enqueue error=%v", err)
			}
		}(job)
	}
	wait.Wait()
	if accepted != capacity || queue.Len() != capacity {
		t.Fatalf("accepted=%d len=%d capacity=%d", accepted, queue.Len(), capacity)
	}
}

func TestMemoryJobQueueConcurrentSameJobDoesNotMutateIdentity(t *testing.T) {
	queue := newMemoryJobQueue(8)
	job := &Job{RequestID: "shared"}
	prepareJobForQueue(queue.ID(), job)
	wantJobID, wantQueueID := job.JobID, job.QueueID
	var wait sync.WaitGroup
	for index := 0; index < 8; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_ = queue.Enqueue(job)
		}()
	}
	wait.Wait()
	if job.JobID != wantJobID || job.QueueID != wantQueueID {
		t.Fatalf("identity changed: job=%q queue=%q", job.JobID, job.QueueID)
	}
}

func TestMemoryJobQueueRemoveUnsupported(t *testing.T) {
	queue := newMemoryJobQueue(3)
	for _, action := range []string{"content.search_notes", "content.search_by_image"} {
		job := &Job{RequestID: action, Action: action}
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue %s: %v", action, err)
		}
	}
	removed := queue.RemoveUnsupported(map[string]struct{}{"content.search_notes": {}}, "capability_removed")
	if len(removed) != 1 || removed[0].Action != "content.search_by_image" {
		t.Fatalf("removed=%+v", removed)
	}
	if queue.Len() != 1 {
		t.Fatalf("remaining=%d", queue.Len())
	}
}

func TestMemoryJobQueueReserveFIFO(t *testing.T) {
	queue := newMemoryJobQueue(3)
	jobs := []*Job{
		{RequestID: "first"},
		{RequestID: "second"},
		{RequestID: "third"},
	}
	for _, job := range jobs {
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue %s: %v", job.RequestID, err)
		}
	}

	seenLeases := make(map[LeaseID]struct{}, len(jobs))
	for index, want := range jobs {
		delivery, err := queue.Reserve(context.Background(), time.Minute)
		if err != nil {
			t.Fatalf("reserve %d: %v", index, err)
		}
		if delivery.Job == want || delivery.Job.RequestID != want.RequestID || delivery.QueueID != queue.ID() || delivery.JobID != want.JobID {
			t.Fatalf("delivery[%d] identity mismatch: %+v", index, delivery)
		}
		if delivery.LeaseID == "" || delivery.Attempt != 1 || delivery.LeaseUntil.IsZero() {
			t.Fatalf("delivery[%d] lease metadata: %+v", index, delivery)
		}
		if _, duplicate := seenLeases[delivery.LeaseID]; duplicate {
			t.Fatalf("duplicate lease ID %q", delivery.LeaseID)
		}
		seenLeases[delivery.LeaseID] = struct{}{}
	}
}

func TestMemoryJobQueueRejectsInvalidDeliveryIdentity(t *testing.T) {
	queue := newMemoryJobQueue(1)
	job := &Job{RequestID: "request-1"}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	wrongQueue := delivery
	wrongQueue.QueueID = QueueID("queue_wrong")
	if err := queue.Ack(wrongQueue); !errors.Is(err, ErrQueueIdentityMismatch) {
		t.Fatalf("wrong queue error=%v", err)
	}

	wrongJob := delivery
	wrongJob.JobID = JobID("job_wrong")
	if err := queue.Ack(wrongJob); !errors.Is(err, ErrLeaseIdentityMismatch) {
		t.Fatalf("wrong job error=%v", err)
	}

	wrongLease := delivery
	wrongLease.LeaseID = LeaseID("lease_wrong")
	if err := queue.Ack(wrongLease); !errors.Is(err, ErrLeaseNotFound) {
		t.Fatalf("wrong lease ack error=%v", err)
	}
	if err := queue.RequeueDelivery(wrongLease, "retry"); !errors.Is(err, ErrLeaseNotFound) {
		t.Fatalf("wrong lease requeue error=%v", err)
	}

	if err := queue.Ack(delivery); err != nil {
		t.Fatalf("valid ack after rejected identities: %v", err)
	}
}

func TestMemoryJobQueueAckIsIdempotent(t *testing.T) {
	queue := newMemoryJobQueue(1)
	job := &Job{RequestID: "request-1"}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := queue.Ack(delivery); err != nil {
		t.Fatalf("first ack: %v", err)
	}
	if err := queue.Ack(delivery); err != nil {
		t.Fatalf("second ack: %v", err)
	}
	if queue.Len() != 0 {
		t.Fatalf("len after ack=%d", queue.Len())
	}

	ackedEvents := 0
	for _, event := range queue.Events() {
		if event.Type == QueueEventAcked {
			ackedEvents++
		}
	}
	if ackedEvents != 1 {
		t.Fatalf("acked events=%d want=1", ackedEvents)
	}
}

func TestMemoryJobQueueRequeueDeliveryMovesToTailAndIncrementsAttempt(t *testing.T) {
	queue := newMemoryJobQueue(2)
	first := &Job{RequestID: "first"}
	second := &Job{RequestID: "second"}
	for _, job := range []*Job{first, second} {
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue %s: %v", job.RequestID, err)
		}
	}

	firstDelivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve first: %v", err)
	}
	if err := queue.RequeueDelivery(firstDelivery, "send_failed"); err != nil {
		t.Fatalf("requeue first: %v", err)
	}

	secondDelivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve second: %v", err)
	}
	if secondDelivery.Job == second || secondDelivery.Job.RequestID != second.RequestID {
		t.Fatalf("tail order got=%s want=%s", secondDelivery.Job.RequestID, second.RequestID)
	}
	if err := queue.Ack(secondDelivery); err != nil {
		t.Fatalf("ack second: %v", err)
	}

	retried, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve retried first: %v", err)
	}
	if retried.Job == first || retried.Job.RequestID != first.RequestID || retried.Attempt != firstDelivery.Attempt+1 {
		t.Fatalf("retried delivery=%+v", retried)
	}
	if retried.LeaseID == firstDelivery.LeaseID {
		t.Fatalf("reused lease ID %q", retried.LeaseID)
	}
}

func TestMemoryJobQueueSweepRequeuesOnlyExpiredLeases(t *testing.T) {
	queue := newMemoryJobQueue(2)
	first := &Job{RequestID: "first"}
	second := &Job{RequestID: "second"}
	for _, job := range []*Job{first, second} {
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue %s: %v", job.RequestID, err)
		}
	}
	firstDelivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve first: %v", err)
	}
	secondDelivery, err := queue.Reserve(context.Background(), 2*time.Minute)
	if err != nil {
		t.Fatalf("reserve second: %v", err)
	}

	if swept := queue.Sweep(firstDelivery.LeaseUntil.Add(-time.Nanosecond)); swept != 0 {
		t.Fatalf("early sweep=%d", swept)
	}
	if swept := queue.Sweep(firstDelivery.LeaseUntil); swept != 1 {
		t.Fatalf("expired sweep=%d want=1", swept)
	}
	retried, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve swept delivery: %v", err)
	}
	if retried.Job == first || retried.Job.RequestID != first.RequestID || retried.Attempt != 2 {
		t.Fatalf("swept delivery=%+v", retried)
	}
	if err := queue.Ack(secondDelivery); err != nil {
		t.Fatalf("second lease should remain valid: %v", err)
	}

	leaseExpiredEvents := 0
	for _, event := range queue.Events() {
		if event.Type == QueueEventLeaseExpired {
			leaseExpiredEvents++
		}
	}
	if leaseExpiredEvents != 1 {
		t.Fatalf("lease expired events=%d want=1", leaseExpiredEvents)
	}
}

func TestMemoryJobQueueSweepCutoffDoesNotExpireAckTombstones(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	queue := newMemoryJobQueueWithNow(2, clock.Now)
	for _, requestID := range []string{"acked", "leased"} {
		job := &Job{RequestID: requestID}
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue %s: %v", requestID, err)
		}
	}

	acked, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve acked delivery: %v", err)
	}
	if err := queue.Ack(acked); err != nil {
		t.Fatalf("first ack: %v", err)
	}
	leased, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve leased delivery: %v", err)
	}

	if swept := queue.Sweep(leased.LeaseUntil.Add(time.Hour)); swept != 1 {
		t.Fatalf("future sweep=%d want=1", swept)
	}
	if err := queue.Ack(acked); err != nil {
		t.Fatalf("repeat ack while authoritative clock is within max age: %v", err)
	}
}

func TestMemoryJobQueueSweepTerminatesAtDeadline(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	queue := newMemoryJobQueueWithNow(1, clock.Now)
	deadline := start.Add(time.Hour)
	job := &Job{RequestID: "deadline", DeadlineAt: deadline}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), 2*time.Hour)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if !delivery.LeaseUntil.Equal(deadline) {
		t.Fatalf("lease until=%s want deadline=%s", delivery.LeaseUntil, deadline)
	}
	clock.Set(deadline)
	if swept := queue.Sweep(deadline); swept != 1 {
		t.Fatalf("swept=%d want=1", swept)
	}
	if queue.Len() != 0 {
		t.Fatalf("deadline job returned to ready queue: len=%d", queue.Len())
	}
	if err := queue.Ack(delivery); !errors.Is(err, ErrLeaseNotFound) {
		t.Fatalf("ack after terminal sweep error=%v", err)
	}
}

func TestMemoryJobQueueCapacityIncludesLeasedJobs(t *testing.T) {
	clock := newQueueTestClock(time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC))
	queue := newMemoryJobQueueWithNow(1, clock.Now)
	first := &Job{RequestID: "first"}
	prepareJobForQueue(queue.ID(), first)
	if err := queue.Enqueue(first); err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve first: %v", err)
	}
	if queue.Len() != queue.Cap() {
		t.Fatalf("leased depth=%d capacity=%d", queue.Len(), queue.Cap())
	}

	second := &Job{RequestID: "second"}
	prepareJobForQueue(queue.ID(), second)
	if err := queue.Enqueue(second); !errors.Is(err, ErrClientQueueFull) {
		t.Fatalf("enqueue while leased error=%v", err)
	}
	if err := queue.Ack(delivery); err != nil {
		t.Fatalf("ack: %v", err)
	}
	if err := queue.Enqueue(second); err != nil {
		t.Fatalf("enqueue after ack: %v", err)
	}
}

func TestMemoryJobQueueLeaseTraceExcludesPayload(t *testing.T) {
	queue := newMemoryJobQueue(1)
	secret := []byte(`{"accessToken":"must-not-appear"}`)
	job := &Job{RequestID: "request-1", Payload: secret}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := queue.Ack(delivery); err != nil {
		t.Fatalf("ack: %v", err)
	}

	encoded, err := json.Marshal(queue.Events())
	if err != nil {
		t.Fatalf("marshal events: %v", err)
	}
	if bytes.Contains(encoded, secret) || bytes.Contains(encoded, []byte("must-not-appear")) {
		t.Fatalf("trace contains payload: %s", encoded)
	}
	events := queue.Events()
	if len(events) != 3 || events[1].Type != QueueEventReserved || events[2].Type != QueueEventAcked {
		t.Fatalf("lease trace=%+v", events)
	}
	if events[1].LeaseRef != leaseReference(delivery.LeaseID) || events[1].Attempt != delivery.Attempt {
		t.Fatalf("reserved trace metadata=%+v", events[1])
	}
	if bytes.Contains(encoded, []byte(delivery.LeaseID)) || bytes.Contains(encoded, []byte(`"leaseId"`)) {
		t.Fatalf("trace exposes actionable lease ID: %s", encoded)
	}
}

func TestMemoryJobQueueConcurrentAckIsIdempotent(t *testing.T) {
	const workers = 32
	queue := newMemoryJobQueue(1)
	job := &Job{RequestID: "request-1"}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	start := make(chan struct{})
	var wait sync.WaitGroup
	var failures atomic.Int32
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			if err := queue.Ack(delivery); err != nil {
				failures.Add(1)
			}
		}()
	}
	close(start)
	wait.Wait()
	if failures.Load() != 0 {
		t.Fatalf("ack failures=%d", failures.Load())
	}
}

func TestMemoryJobQueueConcurrentAckAndRequeueAreAtomic(t *testing.T) {
	queue := newMemoryJobQueue(1)
	job := &Job{RequestID: "request-1"}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		results <- queue.Ack(delivery)
	}()
	go func() {
		<-start
		results <- queue.RequeueDelivery(delivery, "concurrent_retry")
	}()
	close(start)

	succeeded := 0
	for index := 0; index < 2; index++ {
		if err := <-results; err == nil {
			succeeded++
		} else if !errors.Is(err, ErrLeaseNotFound) {
			t.Fatalf("race error=%v", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("successful transitions=%d want=1", succeeded)
	}
	if depth := queue.Len(); depth != 0 && depth != 1 {
		t.Fatalf("depth after race=%d", depth)
	}
}

func TestMemoryJobQueueLeaseBoundaryTransitions(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	operations := []struct {
		name string
		run  func(*memoryJobQueue, Delivery) error
	}{
		{name: "ack", run: func(queue *memoryJobQueue, delivery Delivery) error {
			return queue.Ack(delivery)
		}},
		{name: "requeue", run: func(queue *memoryJobQueue, delivery Delivery) error {
			return queue.RequeueDelivery(delivery, "boundary")
		}},
	}

	for _, operation := range operations {
		operation := operation
		t.Run(operation.name+" before boundary", func(t *testing.T) {
			clock := newQueueTestClock(start)
			queue := newMemoryJobQueueWithNow(1, clock.Now)
			job := &Job{RequestID: operation.name + "-before"}
			prepareJobForQueue(queue.ID(), job)
			if err := queue.Enqueue(job); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
			delivery, err := queue.Reserve(context.Background(), time.Minute)
			if err != nil {
				t.Fatalf("reserve: %v", err)
			}
			clock.Set(delivery.LeaseUntil.Add(-time.Nanosecond))
			if err := operation.run(queue, delivery); err != nil {
				t.Fatalf("transition before boundary: %v", err)
			}
		})

		t.Run(operation.name+" at boundary", func(t *testing.T) {
			clock := newQueueTestClock(start)
			queue := newMemoryJobQueueWithNow(1, clock.Now)
			job := &Job{RequestID: operation.name + "-at"}
			prepareJobForQueue(queue.ID(), job)
			if err := queue.Enqueue(job); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
			delivery, err := queue.Reserve(context.Background(), time.Minute)
			if err != nil {
				t.Fatalf("reserve: %v", err)
			}
			clock.Set(delivery.LeaseUntil)
			if err := operation.run(queue, delivery); !errors.Is(err, ErrLeaseExpired) {
				t.Fatalf("transition at boundary error=%v", err)
			}
			retried, err := queue.Reserve(context.Background(), time.Minute)
			if err != nil {
				t.Fatalf("reserve expired lease: %v", err)
			}
			if retried.Job.RequestID != job.RequestID || retried.Attempt != delivery.Attempt+1 {
				t.Fatalf("expired lease was not requeued: %+v", retried)
			}
		})
	}
}

func TestMemoryJobQueueExpiredTransitionsWakeWaiter(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	operations := []struct {
		name string
		run  func(*memoryJobQueue, Delivery) error
	}{
		{name: "ack", run: func(queue *memoryJobQueue, delivery Delivery) error {
			return queue.Ack(delivery)
		}},
		{name: "requeue", run: func(queue *memoryJobQueue, delivery Delivery) error {
			return queue.RequeueDelivery(delivery, "expired")
		}},
		{name: "sweep", run: func(queue *memoryJobQueue, delivery Delivery) error {
			if swept := queue.Sweep(delivery.LeaseUntil); swept != 1 {
				return errors.New("sweep did not expire lease")
			}
			return nil
		}},
	}

	for _, operation := range operations {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			clock := newQueueTestClock(start)
			queue := newMemoryJobQueueWithNow(1, clock.Now)
			job := &Job{RequestID: operation.name}
			prepareJobForQueue(queue.ID(), job)
			if err := queue.Enqueue(job); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
			delivery, err := queue.Reserve(context.Background(), time.Minute)
			if err != nil {
				t.Fatalf("reserve: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			started := make(chan struct{})
			result := make(chan Delivery, 1)
			errorsCh := make(chan error, 1)
			go func() {
				close(started)
				retried, reserveErr := queue.Reserve(ctx, time.Minute)
				if reserveErr != nil {
					errorsCh <- reserveErr
					return
				}
				result <- retried
			}()
			<-started

			clock.Set(delivery.LeaseUntil)
			transitionErr := operation.run(queue, delivery)
			if operation.name == "sweep" {
				if transitionErr != nil {
					t.Fatalf("sweep: %v", transitionErr)
				}
			} else if !errors.Is(transitionErr, ErrLeaseExpired) {
				t.Fatalf("expired transition error=%v", transitionErr)
			}

			select {
			case retried := <-result:
				if retried.Job.RequestID != job.RequestID || retried.Attempt != 2 {
					t.Fatalf("waiter delivery=%+v", retried)
				}
			case reserveErr := <-errorsCh:
				t.Fatalf("waiting reserve: %v", reserveErr)
			case <-ctx.Done():
				t.Fatal("expired transition did not wake waiting reserve")
			}
		})
	}
}

func TestMemoryJobQueueExpiredLeaseReclaimsCapacityAtDeadline(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	for _, operation := range []struct {
		name string
		run  func(*memoryJobQueue, Delivery) error
	}{
		{name: "ack", run: func(queue *memoryJobQueue, delivery Delivery) error {
			return queue.Ack(delivery)
		}},
		{name: "requeue", run: func(queue *memoryJobQueue, delivery Delivery) error {
			return queue.RequeueDelivery(delivery, "deadline")
		}},
	} {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			clock := newQueueTestClock(start)
			queue := newMemoryJobQueueWithNow(1, clock.Now)
			job := &Job{RequestID: "deadline", DeadlineAt: start.Add(time.Minute)}
			prepareJobForQueue(queue.ID(), job)
			if err := queue.Enqueue(job); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
			delivery, err := queue.Reserve(context.Background(), 2*time.Minute)
			if err != nil {
				t.Fatalf("reserve: %v", err)
			}
			clock.Set(delivery.LeaseUntil)
			if err := operation.run(queue, delivery); !errors.Is(err, ErrLeaseExpired) {
				t.Fatalf("expired transition error=%v", err)
			}
			if depth := queue.Len(); depth != 0 {
				t.Fatalf("terminal lease depth=%d", depth)
			}
			next := &Job{RequestID: "next"}
			prepareJobForQueue(queue.ID(), next)
			if err := queue.Enqueue(next); err != nil {
				t.Fatalf("capacity was not reclaimed: %v", err)
			}
		})
	}
}

func TestMemoryJobQueueAckTombstonesAreBoundedByMaxAgeAndMaxEntries(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)

	t.Run("max age", func(t *testing.T) {
		clock := newQueueTestClock(start)
		queue := newMemoryJobQueueWithNow(1, clock.Now)
		if queue.ackTombstoneMaxAge != 5*time.Minute {
			t.Fatalf("max age=%s want=5m", queue.ackTombstoneMaxAge)
		}
		job := &Job{RequestID: "max-age"}
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		delivery, err := queue.Reserve(context.Background(), time.Minute)
		if err != nil {
			t.Fatalf("reserve: %v", err)
		}
		if err := queue.Ack(delivery); err != nil {
			t.Fatalf("first ack: %v", err)
		}
		clock.Set(start.Add(memoryAckTombstoneMaxAge - time.Nanosecond))
		if err := queue.Ack(delivery); err != nil {
			t.Fatalf("ack before max age: %v", err)
		}
		clock.Set(start.Add(memoryAckTombstoneMaxAge))
		if err := queue.Ack(delivery); !errors.Is(err, ErrLeaseNotFound) {
			t.Fatalf("ack at expired tombstone boundary error=%v", err)
		}
		if len(queue.acked) != 0 {
			t.Fatalf("expired tombstones=%d", len(queue.acked))
		}
	})

	t.Run("max entries evict oldest", func(t *testing.T) {
		clock := newQueueTestClock(start)
		queue := newMemoryJobQueueWithNow(1, clock.Now)
		if queue.ackTombstoneMaxEntries != 1024 {
			t.Fatalf("max entries=%d want=1024", queue.ackTombstoneMaxEntries)
		}
		queue.ackTombstoneMaxEntries = 3
		deliveries := make([]Delivery, 0, 5)
		for index := 0; index < 5; index++ {
			job := &Job{RequestID: newOpaqueID("tombstone_")}
			prepareJobForQueue(queue.ID(), job)
			if err := queue.Enqueue(job); err != nil {
				t.Fatalf("enqueue %d: %v", index, err)
			}
			delivery, err := queue.Reserve(context.Background(), time.Minute)
			if err != nil {
				t.Fatalf("reserve %d: %v", index, err)
			}
			if err := queue.Ack(delivery); err != nil {
				t.Fatalf("ack %d: %v", index, err)
			}
			deliveries = append(deliveries, delivery)
		}
		if len(queue.acked) != queue.ackTombstoneMaxEntries {
			t.Fatalf("tombstones=%d max entries=%d", len(queue.acked), queue.ackTombstoneMaxEntries)
		}
		if err := queue.Ack(deliveries[0]); !errors.Is(err, ErrLeaseNotFound) {
			t.Fatalf("evicted tombstone ack error=%v", err)
		}
		if err := queue.Ack(deliveries[len(deliveries)-1]); err != nil {
			t.Fatalf("newest tombstone ack: %v", err)
		}

		clock.Set(start.Add(memoryAckTombstoneMaxAge))
		queue.Sweep(clock.Now())
		if len(queue.acked) != 0 {
			t.Fatalf("sweep retained expired tombstones=%d", len(queue.acked))
		}
	})
}

func TestMemoryJobQueueOwnsJobCopies(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	queue := newMemoryJobQueueWithNow(2, clock.Now)
	deadline := start.Add(time.Hour)
	originalPayload := []byte(`{"stable":true}`)
	job := &Job{
		JobID:      "job-original",
		RequestID:  "request-original",
		Action:     "action.original",
		Payload:    append([]byte(nil), originalPayload...),
		DeadlineAt: deadline,
	}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	job.JobID = "job-mutated"
	job.Action = "action.mutated"
	job.Payload[0] = 'x'
	job.DeadlineAt = start.Add(-time.Hour)

	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if delivery.Job == job || delivery.Job.JobID != "job-original" || delivery.Job.Action != "action.original" ||
		!bytes.Equal(delivery.Job.Payload, originalPayload) || !delivery.Job.DeadlineAt.Equal(deadline) {
		t.Fatalf("reserve copy was polluted: %+v", delivery.Job)
	}

	delivery.Job.JobID = "delivery-mutated"
	delivery.Job.Action = "delivery.mutated"
	delivery.Job.Payload[0] = 'y'
	delivery.Job.DeadlineAt = start.Add(-time.Hour)
	if err := queue.RequeueDelivery(delivery, "retry"); err != nil {
		t.Fatalf("requeue delivery: %v", err)
	}
	retried, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve retry: %v", err)
	}
	if retried.Job == delivery.Job || retried.Job.JobID != "job-original" || retried.Job.Action != "action.original" ||
		!bytes.Equal(retried.Job.Payload, originalPayload) || !retried.Job.DeadlineAt.Equal(deadline) {
		t.Fatalf("delivery copy polluted queue state: %+v", retried.Job)
	}

	dequeueJob := &Job{JobID: "dequeue-original", RequestID: "dequeue", Action: "dequeue.original", Payload: []byte("payload")}
	prepareJobForQueue(queue.ID(), dequeueJob)
	if err := queue.Enqueue(dequeueJob); err != nil {
		t.Fatalf("enqueue dequeue job: %v", err)
	}
	dequeueJob.Action = "dequeue.mutated"
	dequeueJob.Payload[0] = 'X'
	dequeued, err := queue.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if dequeued == dequeueJob || dequeued.Action != "dequeue.original" || string(dequeued.Payload) != "payload" {
		t.Fatalf("dequeue copy was polluted: %+v", dequeued)
	}
}

func TestMemoryJobQueueJobCopyIsolationUnderRace(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	queue := newMemoryJobQueueWithNow(1, clock.Now)
	job := &Job{JobID: "private", RequestID: "race", Action: "stable", Payload: []byte("stable")}
	prepareJobForQueue(queue.ID(), job)
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivery, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	startMutating := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-startMutating
		for index := 0; index < 1000; index++ {
			job.Action = newOpaqueID("caller_")
			job.Payload[0] = byte(index)
			job.JobID = JobID(newOpaqueID("caller_job_"))
		}
	}()
	go func() {
		defer wait.Done()
		<-startMutating
		for index := 0; index < 1000; index++ {
			delivery.Job.Action = newOpaqueID("delivery_")
			delivery.Job.Payload[0] = byte(index)
			delivery.Job.JobID = JobID(newOpaqueID("delivery_job_"))
		}
	}()
	close(startMutating)
	if err := queue.RequeueDelivery(delivery, "race"); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	for index := 0; index < 100; index++ {
		_ = queue.Events()
	}
	wait.Wait()

	retried, err := queue.Reserve(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("reserve retry: %v", err)
	}
	if retried.Job.JobID != "private" || retried.Job.Action != "stable" || string(retried.Job.Payload) != "stable" {
		t.Fatalf("private job was polluted: %+v", retried.Job)
	}
}

func TestMemoryJobQueueMixedConcurrentDepthNeverExceedsCapacity(t *testing.T) {
	const capacity = 12
	startTime := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(startTime)
	queue := newMemoryJobQueueWithNow(capacity, clock.Now)
	deliveries := make([]Delivery, 0, capacity)
	for index := 0; index < capacity; index++ {
		job := &Job{RequestID: newOpaqueID("mixed_")}
		prepareJobForQueue(queue.ID(), job)
		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue initial %d: %v", index, err)
		}
		leaseDuration := time.Minute
		if index%2 == 1 {
			leaseDuration = 2 * time.Minute
		}
		delivery, err := queue.Reserve(context.Background(), leaseDuration)
		if err != nil {
			t.Fatalf("reserve initial %d: %v", index, err)
		}
		deliveries = append(deliveries, delivery)
	}
	clock.Set(startTime.Add(time.Minute))

	start := make(chan struct{})
	var wait sync.WaitGroup
	var maximum atomic.Int32
	observeDepth := func() {
		depth := int32(queue.Len())
		for current := maximum.Load(); depth > current && !maximum.CompareAndSwap(current, depth); current = maximum.Load() {
		}
		if depth > capacity {
			t.Errorf("depth=%d capacity=%d", depth, capacity)
		}
	}

	for index, delivery := range deliveries {
		index, delivery := index, delivery
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			var err error
			switch index % 3 {
			case 0:
				err = queue.Ack(delivery)
			case 1:
				err = queue.RequeueDelivery(delivery, "mixed")
			case 2:
				queue.Sweep(clock.Now())
			}
			if err != nil && !errors.Is(err, ErrLeaseExpired) && !errors.Is(err, ErrLeaseNotFound) {
				t.Errorf("transition %d: %v", index, err)
			}
			observeDepth()
		}()
	}
	for index := 0; index < capacity; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			job := &Job{RequestID: newOpaqueID("concurrent_enqueue_")}
			prepareJobForQueue(queue.ID(), job)
			if err := queue.Enqueue(job); err != nil && !errors.Is(err, ErrClientQueueFull) {
				t.Errorf("enqueue %d: %v", index, err)
			}
			observeDepth()
		}()
	}
	close(start)
	wait.Wait()
	observeDepth()
	if maximum.Load() > capacity {
		t.Fatalf("maximum depth=%d capacity=%d", maximum.Load(), capacity)
	}
}
