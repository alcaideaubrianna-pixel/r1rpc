package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func reserveHubDelivery(t *testing.T, hub *Hub, session *ClientSession, requestID, action string, deadline time.Time) (Delivery, chan JobResult) {
	t.Helper()
	job := &Job{RequestID: requestID, Group: session.Group, Action: action, ClientID: session.ClientID, DeadlineAt: deadline}
	prepareJobForQueue(session.Pending.ID(), job)
	resultCh := make(chan JobResult, 1)
	hub.storeWaiter(requestID, session.ClientID, resultCh)
	if err := session.Pending.Enqueue(job); err != nil {
		t.Fatalf("enqueue %s: %v", requestID, err)
	}
	delivery, err := session.Pending.Reserve(context.Background(), hub.LeaseDuration())
	if err != nil {
		t.Fatalf("reserve %s: %v", requestID, err)
	}
	return delivery, resultCh
}

func TestAcquireExecutionLeaseEnforcesDeviceAndActionLimits(t *testing.T) {
	clock := newQueueTestClock(time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC))
	hub := NewHubWithTiming(16, 2, time.Minute, time.Second, clock.Now)
	session := hub.RegisterCapabilitiesWithLimits("device", "group", 0, "test", 2, true, []string{"a", "b"}, map[string]int{"a": 1, "b": 2})

	a1, _ := reserveHubDelivery(t, hub, session, "a-1", "a", clock.Now().Add(time.Minute))
	leaseA, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, a1, session.Generation)
	if err != nil {
		t.Fatalf("acquire a-1: %v", err)
	}

	a2, _ := reserveHubDelivery(t, hub, session, "a-2", "a", clock.Now().Add(time.Minute))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := hub.AcquireExecutionLease(canceled, session.ClientID, a2, session.Generation); !errors.Is(err, context.Canceled) {
		t.Fatalf("second action a acquire error=%v", err)
	}
	if session.InFlight != 1 || session.ActionInFlight["a"] != 1 {
		t.Fatalf("failed acquire changed counters: device=%d action=%d", session.InFlight, session.ActionInFlight["a"])
	}

	b1, _ := reserveHubDelivery(t, hub, session, "b-1", "b", clock.Now().Add(time.Minute))
	leaseB, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, b1, session.Generation)
	if err != nil {
		t.Fatalf("acquire b-1: %v", err)
	}
	b2, _ := reserveHubDelivery(t, hub, session, "b-2", "b", clock.Now().Add(time.Minute))
	if _, err := hub.AcquireExecutionLease(canceled, session.ClientID, b2, session.Generation); !errors.Is(err, context.Canceled) {
		t.Fatalf("device limit acquire error=%v", err)
	}
	if session.InFlight != 2 || session.ActionInFlight["a"] != 1 || session.ActionInFlight["b"] != 1 {
		t.Fatalf("counters device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}

	if !hub.ReleaseExecutionLease(leaseA) || !hub.ReleaseExecutionLease(leaseB) {
		t.Fatal("exact execution leases were not released")
	}
	if hub.ReleaseExecutionLease(leaseA) {
		t.Fatal("duplicate release changed state")
	}
}

func TestExecutionLeaseOutlivesCallerDeadlineByGrace(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	hub := NewHubWithExecutionTiming(8, 1, 2*time.Second, time.Second, 20*time.Second, clock.Now)
	session := hub.Register("device", "group", 0, "test", 1)
	deadline := start.Add(5 * time.Second)
	delivery, _ := reserveHubDelivery(t, hub, session, "grace", "action", deadline)
	lease, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, delivery, session.Generation)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	wantExpiry := start.Add(20 * time.Second)
	if !lease.ExpiresAt.Equal(wantExpiry) || !lease.Delivery.LeaseUntil.Equal(wantExpiry) {
		t.Fatalf("execution expiry=%s delivery expiry=%s want=%s", lease.ExpiresAt, lease.Delivery.LeaseUntil, wantExpiry)
	}

	clock.Set(deadline)
	hub.expireWaiter(delivery.Job.RequestID)
	hub.Sweep(deadline)
	if session.InFlight != 1 {
		t.Fatalf("deadline released execution slot: %d", session.InFlight)
	}
	clock.Set(wantExpiry)
	hub.Sweep(wantExpiry)
	if session.InFlight != 0 {
		t.Fatalf("execution expiry did not release slot: %d", session.InFlight)
	}
	late, err := hub.SubmitResult(session.ClientID, JobResult{RequestID: delivery.Job.RequestID, SessionIncarnation: session.SessionIncarnation})
	if err != nil || !late.Late || session.InFlight != 0 {
		t.Fatalf("late outcome=%+v err=%v inFlight=%d", late, err, session.InFlight)
	}
}

func TestAcquireExecutionLeaseRenewsReservationWhileWaitingForSlot(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	hub := NewHubWithExecutionTiming(8, 1, 2*time.Second, time.Second, 5*time.Second, clock.Now)
	session := hub.RegisterCapabilities("device", "group", 0, "test", 1, true, []string{"action"})
	first, _ := reserveHubDelivery(t, hub, session, "first-renew", "action", start.Add(time.Minute))
	firstLease, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, first, session.Generation)
	if err != nil {
		t.Fatalf("acquire first: %v", err)
	}
	second, _ := reserveHubDelivery(t, hub, session, "second-renew", "action", start.Add(time.Minute))

	type acquireResult struct {
		lease *ExecutionLease
		err   error
	}
	resultCh := make(chan acquireResult, 1)
	go func() {
		lease, acquireErr := hub.AcquireExecutionLease(context.Background(), session.ClientID, second, session.Generation)
		resultCh <- acquireResult{lease: lease, err: acquireErr}
	}()
	clock.Set(second.LeaseUntil.Add(-500 * time.Millisecond))
	hub.RegisterCapabilities(session.ClientID, session.Group, session.UserID, session.Platform, session.MaxInFlight, true, []string{"action"})
	time.Sleep(10 * time.Millisecond)
	clock.Set(second.LeaseUntil.Add(time.Second))
	if !hub.ReleaseExecutionLease(firstLease) {
		t.Fatal("release first")
	}
	select {
	case result := <-resultCh:
		if result.err != nil || result.lease == nil {
			t.Fatalf("waiting acquire lease=%+v err=%v", result.lease, result.err)
		}
		if !result.lease.Delivery.LeaseUntil.After(second.LeaseUntil) {
			t.Fatalf("reservation was not renewed: old=%s new=%s", second.LeaseUntil, result.lease.Delivery.LeaseUntil)
		}
	case <-time.After(time.Second):
		t.Fatal("waiting acquire did not complete")
	}
}

func TestAcquireExecutionLeaseIsAtomic(t *testing.T) {
	clock := newQueueTestClock(time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC))
	hub := NewHubWithExecutionTiming(8, 1, time.Minute, time.Second, 10*time.Second, clock.Now)
	session := hub.RegisterCapabilities("device", "group", 0, "test", 1, true, []string{"a", "b"})
	first, _ := reserveHubDelivery(t, hub, session, "first", "a", clock.Now().Add(time.Minute))
	second, _ := reserveHubDelivery(t, hub, session, "second", "b", clock.Now().Add(time.Minute))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := make(chan struct{})
	type acquireResult struct {
		lease *ExecutionLease
		err   error
	}
	results := make(chan acquireResult, 2)
	var workers sync.WaitGroup
	for _, delivery := range []Delivery{first, second} {
		workers.Add(1)
		go func(delivery Delivery) {
			defer workers.Done()
			<-start
			lease, err := hub.AcquireExecutionLease(ctx, session.ClientID, delivery, session.Generation)
			results <- acquireResult{lease: lease, err: err}
		}(delivery)
	}
	close(start)
	winner := <-results
	if winner.err != nil {
		t.Fatalf("first atomic acquire: %v", winner.err)
	}
	cancel()
	loser := <-results
	workers.Wait()
	if !errors.Is(loser.err, context.Canceled) {
		t.Fatalf("second atomic acquire error=%v", loser.err)
	}
	if session.InFlight != 1 {
		t.Fatalf("device count=%d, want 1", session.InFlight)
	}
	if !hub.ReleaseExecutionLease(winner.lease) {
		t.Fatal("winner release failed")
	}
}

func TestSubmitResultAcksDeliveryAndReleasesOnce(t *testing.T) {
	for _, status := range []string{"success", "error"} {
		t.Run(status, func(t *testing.T) {
			clock := newQueueTestClock(time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC))
			hub := NewHubWithTiming(8, 1, time.Minute, time.Second, clock.Now)
			session := hub.Register("device", "group", 0, "test", 1)
			delivery, resultCh := reserveHubDelivery(t, hub, session, "request-"+status, "action", clock.Now().Add(time.Minute))
			if _, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, delivery, session.Generation); err != nil {
				t.Fatalf("acquire: %v", err)
			}
			result := JobResult{RequestID: delivery.Job.RequestID, Status: status, SessionIncarnation: session.SessionIncarnation}
			outcome, err := hub.SubmitResult(session.ClientID, result)
			if err != nil || !outcome.Delivered {
				t.Fatalf("submit outcome=%+v err=%v", outcome, err)
			}
			if got := <-resultCh; got.Status != status {
				t.Fatalf("waiter result=%+v", got)
			}
			if session.InFlight != 0 || session.ActionInFlight["action"] != 0 {
				t.Fatalf("counters device=%d actions=%v", session.InFlight, session.ActionInFlight)
			}
			duplicate, err := hub.SubmitResult(session.ClientID, result)
			if err != nil || !duplicate.Duplicate || session.InFlight != 0 {
				t.Fatalf("duplicate outcome=%+v err=%v inFlight=%d", duplicate, err, session.InFlight)
			}
			events := session.Pending.Events()
			if events[len(events)-1].Type != QueueEventAcked {
				t.Fatalf("last queue event=%+v", events[len(events)-1])
			}
		})
	}
}

func TestSweepReclaimsTimedOutExecutionAndLateResultDoesNotDoubleRelease(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	hub := NewHubWithExecutionTiming(8, 1, time.Minute, time.Second, 10*time.Second, clock.Now)
	session := hub.Register("device", "group", 0, "test", 1)
	deadline := start.Add(10 * time.Second)
	delivery, resultCh := reserveHubDelivery(t, hub, session, "timeout", "action", deadline)
	if _, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, delivery, session.Generation); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	clock.Set(deadline)
	if swept := hub.Sweep(deadline); swept != 1 {
		t.Fatalf("swept=%d, want 1", swept)
	}
	if session.InFlight != 0 || session.ActionInFlight["action"] != 0 {
		t.Fatalf("counters after sweep device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}
	if result := <-resultCh; result.Status != "error" || result.HTTPCode != 504 {
		t.Fatalf("reaper result=%+v", result)
	}
	late, err := hub.SubmitResult(session.ClientID, JobResult{RequestID: delivery.Job.RequestID, SessionIncarnation: session.SessionIncarnation})
	if err != nil || !late.Late || session.InFlight != 0 {
		t.Fatalf("late outcome=%+v err=%v inFlight=%d", late, err, session.InFlight)
	}
}

func TestWaiterTimeoutDefersReleaseUntilSweep(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	hub := NewHubWithExecutionTiming(8, 1, time.Minute, time.Second, 10*time.Second, clock.Now)
	session := hub.Register("device", "group", 0, "test", 1)
	deadline := start.Add(10 * time.Second)
	delivery, _ := reserveHubDelivery(t, hub, session, "caller-timeout", "action", deadline)
	if _, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, delivery, session.Generation); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	hub.expireWaiter(delivery.Job.RequestID)
	if session.InFlight != 1 || session.ActionInFlight["action"] != 1 {
		t.Fatalf("waiter timeout released lease early: device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}
	clock.Set(deadline)
	hub.Sweep(deadline)
	if session.InFlight != 0 || session.ActionInFlight["action"] != 0 {
		t.Fatalf("sweep did not reclaim lease: device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}

	late, err := hub.SubmitResult(session.ClientID, JobResult{RequestID: delivery.Job.RequestID, SessionIncarnation: session.SessionIncarnation})
	if err != nil || !late.Late {
		t.Fatalf("late outcome=%+v err=%v", late, err)
	}
	if session.InFlight != 0 {
		t.Fatalf("late result decremented twice: inFlight=%d", session.InFlight)
	}
}

func TestSessionGenerationProtectsReplacementAndDisconnectReclaims(t *testing.T) {
	clock := newQueueTestClock(time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC))
	hub := NewHubWithTiming(8, 1, time.Minute, time.Second, clock.Now)
	oldSession := hub.Register("device", "group", 0, "test", 1)
	delivery, resultCh := reserveHubDelivery(t, hub, oldSession, "disconnect", "action", clock.Now().Add(time.Minute))
	if _, err := hub.AcquireExecutionLease(context.Background(), oldSession.ClientID, delivery, oldSession.Generation); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	newSession := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	if newSession.Generation == oldSession.Generation {
		t.Fatal("replacement reused session generation")
	}
	if result := <-resultCh; result.Status != "error" || result.HTTPCode != 503 {
		t.Fatalf("replacement result=%+v", result)
	}
	if oldSession.InFlight != 0 {
		t.Fatalf("old session inFlight=%d", oldSession.InFlight)
	}
	if newSession.Pending != oldSession.Pending {
		t.Fatal("replacement did not reuse the client queue")
	}
	retried, err := newSession.Pending.Reserve(context.Background(), hub.LeaseDuration())
	if err != nil {
		t.Fatalf("reserve replacement retry: %v", err)
	}
	if retried.JobID != delivery.JobID || retried.Attempt != delivery.Attempt+1 {
		t.Fatalf("replacement retry=%+v original=%+v", retried, delivery)
	}
	if _, err := hub.AcquireExecutionLease(context.Background(), newSession.ClientID, retried, newSession.Generation); err != nil {
		t.Fatalf("replacement acquire: %v", err)
	}
	if outcome, err := hub.SubmitResult(newSession.ClientID, JobResult{RequestID: retried.Job.RequestID, SessionIncarnation: newSession.SessionIncarnation}); err != nil || !outcome.Delivered {
		t.Fatalf("replacement result outcome=%+v err=%v", outcome, err)
	}

	hub.UnregisterGeneration("device", oldSession.Generation)
	current, ok := hub.Session("device")
	if !ok || current != newSession {
		t.Fatalf("old unregister removed replacement: current=%p ok=%v", current, ok)
	}
	hub.UnregisterGeneration("device", newSession.Generation)
	if _, ok := hub.Session("device"); ok {
		t.Fatal("current generation remained registered")
	}
}

func TestReplacementRecordsAtLeastOnceRiskAfterPhysicalSend(t *testing.T) {
	hub := NewHub(8, 1)
	session := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	delivery, _ := reserveHubDelivery(t, hub, session, "sent-before-replacement", "action", time.Now().Add(time.Minute))
	lease, err := hub.AcquireExecutionLeaseIncarnation(context.Background(), session.ClientID, delivery, session.SessionIncarnation)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if !hub.MarkExecutionLeaseSent(lease) {
		t.Fatal("mark sent failed")
	}

	hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	events := session.Pending.Events()
	if len(events) == 0 {
		t.Fatal("queue trace is empty")
	}
	last := events[len(events)-1]
	if last.Type != QueueEventRequeued || last.Reason != "session_replaced_after_send_at_least_once" {
		t.Fatalf("replacement trace=%+v", last)
	}
}

func TestUnregisterCurrentSessionTerminatesInFlightRequest(t *testing.T) {
	clock := newQueueTestClock(time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC))
	hub := NewHubWithTiming(8, 1, time.Minute, time.Second, clock.Now)
	session := hub.Register("device", "group", 0, "test", 1)
	delivery, resultCh := reserveHubDelivery(t, hub, session, "disconnect-current", "action", clock.Now().Add(time.Minute))
	if _, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, delivery, session.Generation); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	hub.UnregisterGeneration(session.ClientID, session.Generation)
	if session.InFlight != 0 || session.ActionInFlight["action"] != 0 {
		t.Fatalf("disconnect leaked slots: device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}
	if result := <-resultCh; result.Status != "error" || result.HTTPCode != 503 {
		t.Fatalf("disconnect result=%+v", result)
	}
	if _, ok := hub.Session(session.ClientID); ok {
		t.Fatal("disconnected session is still online")
	}
	replacement := hub.RegisterConnectionCapabilities(session.ClientID, session.Group, 0, "test", 1, false, nil)
	if replacement.Pending != session.Pending {
		t.Fatal("reconnect did not retain the disconnected queue")
	}
	retried, err := replacement.Pending.Reserve(context.Background(), hub.LeaseDuration())
	if err != nil {
		t.Fatalf("reserve reconnect retry: %v", err)
	}
	if retried.JobID != delivery.JobID || retried.Attempt != delivery.Attempt+1 {
		t.Fatalf("reconnect retry=%+v original=%+v", retried, delivery)
	}
}

func TestReadyJobSurvivesDisconnectAndReconnect(t *testing.T) {
	hub := NewHub(8, 1)
	session := hub.Register("device", "group", 0, "test", 1)
	job := &Job{RequestID: "ready-disconnect", Group: session.Group, Action: "action", ClientID: session.ClientID, DeadlineAt: time.Now().Add(time.Minute)}
	prepareJobForQueue(session.Pending.ID(), job)
	waiter := make(chan JobResult, 1)
	hub.storeWaiter(job.RequestID, session.ClientID, waiter)
	if err := session.Pending.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	hub.UnregisterGeneration(session.ClientID, session.Generation)
	if result := <-waiter; result.HTTPCode != 503 {
		t.Fatalf("disconnect result=%+v", result)
	}
	replacement := hub.RegisterConnectionCapabilities(session.ClientID, session.Group, 0, "test", 1, false, nil)
	if replacement.Pending != session.Pending || replacement.Pending.Len() != 1 {
		t.Fatalf("replacement queue=%p old=%p depth=%d", replacement.Pending, session.Pending, replacement.Pending.Len())
	}
	delivery, err := replacement.Pending.Reserve(context.Background(), hub.LeaseDuration())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := hub.AcquireExecutionLease(context.Background(), replacement.ClientID, delivery, replacement.Generation); err != nil {
		t.Fatalf("acquire: %v", err)
	}
}

func TestDisconnectedReadyJobTerminatesAtDeadline(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	hub := NewHubWithExecutionTiming(8, 1, 5*time.Second, time.Second, 10*time.Second, clock.Now)
	session := hub.Register("device", "group", 0, "test", 1)
	deadline := start.Add(10 * time.Second)
	job := &Job{RequestID: "ready-deadline", Group: session.Group, Action: "action", ClientID: session.ClientID, DeadlineAt: deadline}
	waiter := make(chan JobResult, 1)
	hub.storeWaiter(job.RequestID, session.ClientID, waiter)
	if err := hub.enqueueSelectedSession(session, session.Group, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	hub.UnregisterGeneration(session.ClientID, session.Generation)
	if result := <-waiter; result.HTTPCode != 503 {
		t.Fatalf("disconnect result=%+v", result)
	}

	clock.Set(deadline)
	hub.Sweep(deadline)
	if session.Pending.Len() != 0 {
		t.Fatalf("expired ready depth=%d", session.Pending.Len())
	}
	hub.mu.RLock()
	_, waiting := hub.waiters[job.RequestID]
	completed := hub.completed[job.RequestID]
	hub.mu.RUnlock()
	if waiting || completed.State != "expired" {
		t.Fatalf("waiting=%v completed=%+v", waiting, completed)
	}

	replacement := hub.RegisterConnectionCapabilities(session.ClientID, session.Group, 0, "test", 1, false, nil)
	reserveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := replacement.Pending.Reserve(reserveCtx, hub.LeaseDuration()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired job remained consumable: %v", err)
	}
}

func TestPresenceRefreshRequiresMatchingGeneration(t *testing.T) {
	clock := newQueueTestClock(time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC))
	hub := NewHubWithTiming(8, 1, time.Minute, time.Second, clock.Now)
	old := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	current := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	clock.Set(clock.Now().Add(time.Minute))
	if hub.TouchGeneration(old.ClientID, old.Generation) {
		t.Fatal("stale generation refreshed current presence")
	}
	if current.LastSeenAt.Equal(clock.Now()) {
		t.Fatal("stale touch changed current timestamp")
	}
	if !hub.TouchGeneration(current.ClientID, current.Generation) || !current.LastSeenAt.Equal(clock.Now()) {
		t.Fatal("current generation did not refresh presence")
	}
}

func TestHubStartReaperAndCloseAreConcurrentIdempotent(t *testing.T) {
	hub := NewHubWithTiming(8, 1, time.Minute, time.Millisecond, time.Now)
	var starters sync.WaitGroup
	for range 50 {
		starters.Add(1)
		go func() {
			defer starters.Done()
			hub.StartReaper()
		}()
	}
	starters.Wait()

	var closers sync.WaitGroup
	for range 50 {
		closers.Add(1)
		go func() {
			defer closers.Done()
			hub.Close()
		}()
	}
	closers.Wait()
	hub.StartReaper()
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if !hub.reaperClosed || hub.reaperStarted {
		t.Fatalf("reaper state started=%v closed=%v", hub.reaperStarted, hub.reaperClosed)
	}
}

func TestPresenceRegisterRefreshDoesNotReplaceConnectionGeneration(t *testing.T) {
	clock := newQueueTestClock(time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC))
	hub := NewHubWithTiming(8, 1, time.Minute, time.Second, clock.Now)
	session := hub.RegisterConnectionCapabilities("device", "group", 0, "websocket", 1, true, []string{"action"})
	delivery, _ := reserveHubDelivery(t, hub, session, "presence-refresh", "action", clock.Now().Add(time.Minute))
	lease, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, delivery, session.Generation)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	refreshed := hub.Register(session.ClientID, session.Group, session.UserID, "", session.MaxInFlight)
	if refreshed != session || refreshed.Generation != session.Generation {
		t.Fatalf("presence refresh replaced connection: old=%p/%d new=%p/%d", session, session.Generation, refreshed, refreshed.Generation)
	}
	if session.InFlight != 1 || session.ActionInFlight["action"] != 1 {
		t.Fatalf("presence refresh changed execution counters: device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}
	if err := hub.RequeueExecutionLease(lease, "test_cleanup"); err != nil {
		t.Fatalf("cleanup requeue: %v", err)
	}
}

func TestSessionFenceRejectsTokenAfterHubReconstruction(t *testing.T) {
	oldHub := NewHub(8, 1)
	oldSession := oldHub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)

	rebuiltHub := NewHub(8, 1)
	current := rebuiltHub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	if oldSession.Generation != current.Generation {
		t.Fatalf("test requires the current restart collision, old=%d current=%d", oldSession.Generation, current.Generation)
	}
	if oldSession.SessionIncarnation == current.SessionIncarnation {
		t.Fatal("independent hubs reused a session incarnation")
	}
	if err := rebuiltHub.ValidateIncarnation("device", oldSession.SessionIncarnation); err == nil {
		t.Fatal("old session fence was accepted after hub reconstruction")
	}
}

func TestHubSweepReclaimsIdleClientQueuesAfterChurn(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	hub := NewHubWithTiming(8, 1, time.Minute, time.Second, clock.Now)

	const clients = 200
	for index := range clients {
		clientID := fmt.Sprintf("device-%d", index)
		session := hub.RegisterConnectionCapabilities(clientID, "group", 0, "test", 1, false, nil)
		hub.UnregisterGeneration(clientID, session.Generation)
	}
	if got := len(hub.queues); got != clients {
		t.Fatalf("queues before retention sweep=%d want=%d", got, clients)
	}

	clock.Set(start.Add(24 * time.Hour))
	hub.Sweep(clock.Now())
	if got := len(hub.queues); got != 0 {
		t.Fatalf("idle queues retained after churn=%d", got)
	}
}

func TestHubQueueRetentionRequiresNoReadyLeasedOrWaiter(t *testing.T) {
	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	clock := newQueueTestClock(start)
	hub := NewHubWithTiming(8, 1, time.Hour, time.Second, clock.Now)
	hub.ConfigureQueueRetention(time.Minute)

	ready := hub.RegisterConnectionCapabilities("ready", "group", 0, "test", 1, false, nil)
	readyJob := &Job{RequestID: "ready", ClientID: ready.ClientID, Group: ready.Group, Action: "action"}
	prepareJobForQueue(ready.Pending.ID(), readyJob)
	if err := ready.Pending.Enqueue(readyJob); err != nil {
		t.Fatalf("enqueue ready: %v", err)
	}
	hub.UnregisterIncarnationE(ready.ClientID, ready.SessionIncarnation)

	leased := hub.RegisterConnectionCapabilities("leased", "group", 0, "test", 1, false, nil)
	leasedJob := &Job{RequestID: "leased", ClientID: leased.ClientID, Group: leased.Group, Action: "action"}
	prepareJobForQueue(leased.Pending.ID(), leasedJob)
	if err := leased.Pending.Enqueue(leasedJob); err != nil {
		t.Fatalf("enqueue leased: %v", err)
	}
	if _, err := leased.Pending.Reserve(context.Background(), time.Hour); err != nil {
		t.Fatalf("reserve leased: %v", err)
	}
	hub.UnregisterIncarnationE(leased.ClientID, leased.SessionIncarnation)

	waiting := hub.RegisterConnectionCapabilities("waiting", "group", 0, "test", 1, false, nil)
	hub.storeWaiter("waiting", waiting.ClientID, make(chan JobResult, 1))
	hub.UnregisterIncarnationE(waiting.ClientID, waiting.SessionIncarnation)

	clock.Set(start.Add(2 * time.Minute))
	hub.mu.Lock()
	hub.cleanupIdleQueuesLocked(clock.Now())
	hub.mu.Unlock()
	for _, clientID := range []string{ready.ClientID, leased.ClientID, waiting.ClientID} {
		if _, ok := hub.queues[clientID]; !ok {
			t.Fatalf("queue %q reclaimed with active state", clientID)
		}
	}
}

func TestActionDefaultAndOverrideNeverExceedDeviceLimit(t *testing.T) {
	hub := NewHub(8, 4)
	hub.ConfigureExecutionPolicy(time.Minute, time.Second, 3)
	hub.RegisterCapabilitiesWithLimits("device", "group", 0, "test", 2, true, []string{"default", "override"}, map[string]int{"override": 9})
	if limit, _ := hub.ActionLimit("device", "default"); limit != 2 {
		t.Fatalf("default action limit=%d, want device cap 2", limit)
	}
	if limit, _ := hub.ActionLimit("device", "override"); limit != 2 {
		t.Fatalf("override action limit=%d, want device cap 2", limit)
	}

	legacy := hub.Register("legacy", "group", 0, "test", 4)
	if limit, _ := hub.ActionLimit(legacy.ClientID, "unknown.action"); limit != 3 {
		t.Fatalf("legacy action limit=%d, want configured default 3", limit)
	}
}
