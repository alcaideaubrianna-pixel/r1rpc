package rpc

import (
	"context"
	"sort"
	"sync"
	"time"
)

const defaultQueueTraceLimit = 512

const (
	memoryAckTombstoneMaxAge     = 5 * time.Minute
	memoryAckTombstoneMaxEntries = 1024
)

type readyJob struct {
	job     *Job
	attempt int
}

type leasedJob struct {
	delivery Delivery
	sequence uint64
}

type ackedDelivery struct {
	queueID   QueueID
	jobID     JobID
	expiresAt time.Time
	sequence  uint64
}

type memoryJobQueue struct {
	mu                     sync.Mutex
	id                     QueueID
	ready                  []readyJob
	leased                 map[LeaseID]leasedJob
	acked                  map[LeaseID]ackedDelivery
	nextSequence           uint64
	limit                  int
	notify                 chan struct{}
	events                 []QueueEvent
	traceLimit             int
	now                    func() time.Time
	ackTombstoneMaxAge     time.Duration
	ackTombstoneMaxEntries int
}

func newMemoryJobQueue(limit int) *memoryJobQueue {
	return newMemoryJobQueueWithNow(limit, time.Now)
}

func newMemoryJobQueueWithNow(limit int, now func() time.Time) *memoryJobQueue {
	if limit <= 0 {
		limit = 1
	}
	if now == nil {
		now = time.Now
	}
	return &memoryJobQueue{
		id:                     QueueID(newOpaqueID("queue_")),
		ready:                  make([]readyJob, 0, limit),
		leased:                 make(map[LeaseID]leasedJob),
		acked:                  make(map[LeaseID]ackedDelivery),
		limit:                  limit,
		notify:                 make(chan struct{}),
		events:                 make([]QueueEvent, 0, min(limit*2, defaultQueueTraceLimit)),
		traceLimit:             defaultQueueTraceLimit,
		now:                    now,
		ackTombstoneMaxAge:     memoryAckTombstoneMaxAge,
		ackTombstoneMaxEntries: memoryAckTombstoneMaxEntries,
	}
}

func (q *memoryJobQueue) ID() QueueID {
	return q.id
}

func (q *memoryJobQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := q.now()
	q.pruneAckedLocked(now)
	q.pruneExpiredReadyLocked(now)
	return q.depthLocked()
}

func (q *memoryJobQueue) Cap() int {
	return q.limit
}

func (q *memoryJobQueue) Enqueue(job *Job) error {
	return q.push(job, QueueEventQueued, "")
}

// Requeue is retained for callers that still use the pre-lease dequeue path.
func (q *memoryJobQueue) Requeue(job *Job, reason string) error {
	return q.push(job, QueueEventRequeued, reason)
}

func (q *memoryJobQueue) push(job *Job, eventType QueueEventType, reason string) error {
	if job == nil {
		return nil
	}
	if job.JobID == "" || job.QueueID != q.id {
		return ErrQueueIdentityMismatch
	}
	owned := cloneJob(job)
	q.mu.Lock()
	defer q.mu.Unlock()
	now := q.now()
	q.pruneAckedLocked(now)
	q.pruneExpiredReadyLocked(now)
	if owned.ExpiredAt(now) {
		q.recordLocked(owned, QueueEventExpired, "deadline_elapsed", now)
		return ErrJobExpired
	}
	if q.depthLocked() >= q.limit {
		q.recordLocked(owned, QueueEventDropped, "capacity", now)
		return ErrClientQueueFull
	}
	q.ready = append(q.ready, readyJob{job: owned, attempt: 1})
	q.recordLocked(owned, eventType, reason, now)
	q.signalLocked()
	return nil
}

func (q *memoryJobQueue) Reserve(ctx context.Context, leaseDuration time.Duration) (Delivery, error) {
	if leaseDuration <= 0 {
		return Delivery{}, ErrInvalidLeaseDuration
	}
	for {
		if err := ctx.Err(); err != nil {
			return Delivery{}, err
		}

		q.mu.Lock()
		now := q.now()
		q.pruneAckedLocked(now)
		delivery, wait := q.reserveOrWaitLocked(now, leaseDuration)
		q.mu.Unlock()
		if delivery.Job != nil {
			return delivery, nil
		}

		select {
		case <-ctx.Done():
			return Delivery{}, ctx.Err()
		case <-wait:
		}
	}
}

func (q *memoryJobQueue) Ack(delivery Delivery) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := q.now()
	q.pruneAckedLocked(now)
	if err := q.validateDeliveryIdentityLocked(delivery); err != nil {
		return err
	}
	if acked, ok := q.acked[delivery.LeaseID]; ok {
		if acked.queueID == delivery.QueueID && acked.jobID == delivery.JobID {
			return nil
		}
		return ErrLeaseIdentityMismatch
	}
	leased, ok := q.leased[delivery.LeaseID]
	if !ok {
		return ErrLeaseNotFound
	}
	if leased.delivery.JobID != delivery.JobID {
		return ErrLeaseIdentityMismatch
	}
	if !now.Before(leased.delivery.LeaseUntil) {
		if q.expireLeaseLocked(leased, now) {
			q.signalLocked()
		}
		return ErrLeaseExpired
	}

	delete(q.leased, delivery.LeaseID)
	q.addAckedLocked(delivery.LeaseID, delivery.QueueID, delivery.JobID, now)
	q.recordDeliveryLocked(leased.delivery, QueueEventAcked, "", now)
	return nil
}

func (q *memoryJobQueue) Renew(delivery Delivery, leaseDuration time.Duration) (Delivery, error) {
	if leaseDuration <= 0 {
		return Delivery{}, ErrInvalidLeaseDuration
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	now := q.now()
	q.pruneAckedLocked(now)
	leased, err := q.findLeasedLocked(delivery)
	if err != nil {
		return Delivery{}, err
	}
	if !now.Before(leased.delivery.LeaseUntil) {
		if q.expireLeaseLocked(leased, now) {
			q.signalLocked()
		}
		return Delivery{}, ErrLeaseExpired
	}
	leaseUntil := now.Add(leaseDuration)
	if leaseUntil.After(leased.delivery.LeaseUntil) {
		leased.delivery.LeaseUntil = leaseUntil
		q.leased[delivery.LeaseID] = leased
		q.recordDeliveryLocked(leased.delivery, QueueEventLeaseRenewed, "", now)
	}
	return cloneDelivery(leased.delivery), nil
}

func (q *memoryJobQueue) RequeueDelivery(delivery Delivery, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := q.now()
	q.pruneAckedLocked(now)
	leased, err := q.findLeasedLocked(delivery)
	if err != nil {
		return err
	}
	if !now.Before(leased.delivery.LeaseUntil) {
		if q.expireLeaseLocked(leased, now) {
			q.signalLocked()
		}
		return ErrLeaseExpired
	}

	delete(q.leased, delivery.LeaseID)
	if leased.delivery.Job.ExpiredAt(now) {
		q.recordDeliveryLocked(leased.delivery, QueueEventExpired, "deadline_elapsed", now)
		return ErrJobExpired
	}

	nextAttempt := leased.delivery.Attempt + 1
	q.ready = append(q.ready, readyJob{job: leased.delivery.Job, attempt: nextAttempt})
	requeued := leased.delivery
	requeued.Attempt = nextAttempt
	q.recordDeliveryLocked(requeued, QueueEventRequeued, reason, now)
	q.signalLocked()
	return nil
}

func (q *memoryJobQueue) Sweep(leaseExpiryCutoff time.Time) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	authoritativeNow := q.now()
	q.pruneAckedLocked(authoritativeNow)
	q.pruneExpiredReadyLocked(authoritativeNow)

	due := make([]leasedJob, 0)
	for _, leased := range q.leased {
		if !leaseExpiryCutoff.Before(leased.delivery.LeaseUntil) {
			due = append(due, leased)
		}
	}
	sort.Slice(due, func(i, j int) bool {
		left, right := due[i], due[j]
		if left.delivery.LeaseUntil.Equal(right.delivery.LeaseUntil) {
			return left.sequence < right.sequence
		}
		return left.delivery.LeaseUntil.Before(right.delivery.LeaseUntil)
	})

	requeued := false
	for _, leased := range due {
		requeued = q.expireLeaseLocked(leased, authoritativeNow) || requeued
	}
	if requeued {
		q.signalLocked()
	}
	return len(due)
}

func (q *memoryJobQueue) Dequeue(ctx context.Context) (*Job, error) {
	for {
		q.mu.Lock()
		now := q.now()
		q.pruneAckedLocked(now)
		job, wait := q.popOrWaitLocked(now)
		q.mu.Unlock()
		if job != nil {
			return job, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wait:
		}
	}
}

func (q *memoryJobQueue) Events() []QueueEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pruneAckedLocked(q.now())
	result := make([]QueueEvent, len(q.events))
	copy(result, q.events)
	return result
}

func (q *memoryJobQueue) RemoveUnsupported(allowed map[string]struct{}, reason string) []*Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := q.now()
	q.pruneAckedLocked(now)
	if len(q.ready) == 0 {
		return nil
	}
	kept := q.ready[:0]
	removed := make([]*Job, 0)
	for _, ready := range q.ready {
		if ready.job == nil {
			continue
		}
		if _, ok := allowed[ready.job.Action]; ok {
			kept = append(kept, ready)
			continue
		}
		removed = append(removed, cloneJob(ready.job))
		q.recordLocked(ready.job, QueueEventDropped, reason, now)
	}
	for index := len(kept); index < len(q.ready); index++ {
		q.ready[index] = readyJob{}
	}
	q.ready = kept
	return removed
}

func (q *memoryJobQueue) reserveOrWaitLocked(now time.Time, leaseDuration time.Duration) (Delivery, chan struct{}) {
	q.pruneExpiredReadyLocked(now)
	if len(q.ready) == 0 {
		return Delivery{}, q.notify
	}
	ready := q.ready[0]
	q.ready[0] = readyJob{}
	q.ready = q.ready[1:]

	leaseUntil := now.Add(leaseDuration)
	if deadline := ready.job.DeadlineAt; !deadline.IsZero() && deadline.Before(leaseUntil) {
		leaseUntil = deadline
	}
	delivery := Delivery{
		QueueID:    q.id,
		JobID:      ready.job.JobID,
		LeaseID:    q.newLeaseIDLocked(),
		Attempt:    ready.attempt,
		LeaseUntil: leaseUntil,
		Job:        ready.job,
	}
	q.nextSequence++
	q.leased[delivery.LeaseID] = leasedJob{delivery: delivery, sequence: q.nextSequence}
	q.recordDeliveryLocked(delivery, QueueEventReserved, "", now)
	return cloneDelivery(delivery), nil
}

func (q *memoryJobQueue) popOrWaitLocked(now time.Time) (*Job, chan struct{}) {
	q.pruneExpiredReadyLocked(now)
	if len(q.ready) == 0 {
		return nil, q.notify
	}
	ready := q.ready[0]
	q.ready[0] = readyJob{}
	q.ready = q.ready[1:]
	q.recordLocked(ready.job, QueueEventDequeued, "", now)
	return cloneJob(ready.job), nil
}

func (q *memoryJobQueue) expireLeaseLocked(leased leasedJob, now time.Time) bool {
	delete(q.leased, leased.delivery.LeaseID)
	q.recordDeliveryLocked(leased.delivery, QueueEventLeaseExpired, "lease_elapsed", now)
	if leased.delivery.Job.ExpiredAt(now) {
		q.recordDeliveryLocked(leased.delivery, QueueEventExpired, "deadline_elapsed", now)
		return false
	}
	q.ready = append(q.ready, readyJob{
		job:     leased.delivery.Job,
		attempt: leased.delivery.Attempt + 1,
	})
	return true
}

func (q *memoryJobQueue) addAckedLocked(leaseID LeaseID, queueID QueueID, jobID JobID, now time.Time) {
	if q.ackTombstoneMaxAge <= 0 || q.ackTombstoneMaxEntries <= 0 {
		return
	}
	for len(q.acked) >= q.ackTombstoneMaxEntries {
		var oldestLeaseID LeaseID
		var oldestSequence uint64
		for candidateLeaseID, acked := range q.acked {
			if oldestLeaseID == "" || acked.sequence < oldestSequence {
				oldestLeaseID = candidateLeaseID
				oldestSequence = acked.sequence
			}
		}
		delete(q.acked, oldestLeaseID)
	}
	q.nextSequence++
	q.acked[leaseID] = ackedDelivery{
		queueID:   queueID,
		jobID:     jobID,
		expiresAt: now.Add(q.ackTombstoneMaxAge),
		sequence:  q.nextSequence,
	}
}

func (q *memoryJobQueue) pruneAckedLocked(now time.Time) {
	for leaseID, acked := range q.acked {
		if !now.Before(acked.expiresAt) {
			delete(q.acked, leaseID)
		}
	}
}

func (q *memoryJobQueue) pruneExpiredReadyLocked(now time.Time) {
	if len(q.ready) == 0 {
		return
	}
	filtered := q.ready[:0]
	for _, ready := range q.ready {
		if ready.job == nil {
			continue
		}
		if ready.job.ExpiredAt(now) {
			q.recordLocked(ready.job, QueueEventExpired, "deadline_elapsed", now)
			continue
		}
		filtered = append(filtered, ready)
	}
	for index := len(filtered); index < len(q.ready); index++ {
		q.ready[index] = readyJob{}
	}
	q.ready = filtered
}

func (q *memoryJobQueue) findLeasedLocked(delivery Delivery) (leasedJob, error) {
	if err := q.validateDeliveryIdentityLocked(delivery); err != nil {
		return leasedJob{}, err
	}
	leased, ok := q.leased[delivery.LeaseID]
	if !ok {
		return leasedJob{}, ErrLeaseNotFound
	}
	if leased.delivery.JobID != delivery.JobID {
		return leasedJob{}, ErrLeaseIdentityMismatch
	}
	return leased, nil
}

func (q *memoryJobQueue) validateDeliveryIdentityLocked(delivery Delivery) error {
	if delivery.QueueID != q.id {
		return ErrQueueIdentityMismatch
	}
	if delivery.JobID == "" || delivery.LeaseID == "" {
		return ErrLeaseIdentityMismatch
	}
	return nil
}

func (q *memoryJobQueue) newLeaseIDLocked() LeaseID {
	for {
		leaseID := LeaseID(newOpaqueID("lease_"))
		if _, exists := q.leased[leaseID]; exists {
			continue
		}
		if _, exists := q.acked[leaseID]; !exists {
			return leaseID
		}
	}
}

func (q *memoryJobQueue) depthLocked() int {
	return len(q.ready) + len(q.leased)
}

func (q *memoryJobQueue) recordLocked(job *Job, eventType QueueEventType, reason string, at time.Time) {
	q.recordDeliveryLocked(Delivery{Job: job}, eventType, reason, at)
}

func (q *memoryJobQueue) recordDeliveryLocked(delivery Delivery, eventType QueueEventType, reason string, at time.Time) {
	job := delivery.Job
	if q.traceLimit <= 0 || job == nil {
		return
	}
	event := QueueEvent{
		EventID:    newOpaqueID("event_"),
		QueueID:    q.id,
		JobID:      job.JobID,
		RequestID:  job.RequestID,
		Group:      job.Group,
		Action:     job.Action,
		ClientID:   job.ClientID,
		LeaseRef:   leaseReference(delivery.LeaseID),
		Attempt:    delivery.Attempt,
		LeaseUntil: delivery.LeaseUntil,
		Type:       eventType,
		At:         at,
		Reason:     reason,
	}
	if len(q.events) == q.traceLimit {
		copy(q.events, q.events[1:])
		q.events[len(q.events)-1] = event
		return
	}
	q.events = append(q.events, event)
}

func (q *memoryJobQueue) signalLocked() {
	close(q.notify)
	q.notify = make(chan struct{})
}

func cloneDelivery(delivery Delivery) Delivery {
	delivery.Job = cloneJob(delivery.Job)
	return delivery
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	cloned := *job
	cloned.Payload = append([]byte(nil), job.Payload...)
	return &cloned
}
