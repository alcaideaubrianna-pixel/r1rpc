package rpc

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPickSessionFiltersActions(t *testing.T) {
	hub := NewHub(8, 1)
	hub.RegisterCapabilities("text", "xhs", 0, "ios", 1, true, []string{" content.search_notes ", "content.search_notes"})
	hub.RegisterCapabilities("image", "xhs", 0, "ios", 1, true, []string{"content.search_by_image"})

	text, err := hub.pickSession("xhs", "", "content.search_notes")
	if err != nil || text.ClientID != "text" {
		t.Fatalf("text selection client=%v err=%v", text, err)
	}
	image, err := hub.pickSession("xhs", "", "content.search_by_image")
	if err != nil || image.ClientID != "image" {
		t.Fatalf("image selection client=%v err=%v", image, err)
	}
}

func TestAcquireExecutionLeaseRejectsCapabilityRemovedWhileWaiting(t *testing.T) {
	hub := NewHub(8, 1)
	session := hub.RegisterCapabilities("device", "group", 0, "test", 1, true, []string{"keep", "removed"})
	first, _ := reserveHubDelivery(t, hub, session, "cap-first", "keep", time.Now().Add(time.Minute))
	firstLease, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, first, session.Generation)
	if err != nil {
		t.Fatalf("acquire first: %v", err)
	}
	second, waiter := reserveHubDelivery(t, hub, session, "cap-second", "removed", time.Now().Add(time.Minute))

	resultCh := make(chan error, 1)
	go func() {
		_, acquireErr := hub.AcquireExecutionLease(context.Background(), session.ClientID, second, session.Generation)
		resultCh <- acquireErr
	}()
	time.Sleep(10 * time.Millisecond)
	hub.RegisterCapabilities("device", "group", 0, "test", 1, true, []string{"keep"})
	select {
	case acquireErr := <-resultCh:
		if !errors.Is(acquireErr, ErrActionNotSupported) {
			t.Fatalf("acquire error=%v", acquireErr)
		}
	case <-time.After(time.Second):
		t.Fatal("capability narrowing did not wake waiting acquire")
	}
	if result := <-waiter; result.HTTPCode != 409 {
		t.Fatalf("waiter result=%+v", result)
	}
	if session.Pending.Len() != 1 {
		t.Fatalf("only active supported delivery should remain, depth=%d", session.Pending.Len())
	}
	if !hub.ReleaseExecutionLease(firstLease) {
		t.Fatal("release first")
	}
}

func TestAcquireExecutionLeaseStillAllowsLegacyUnknownCapabilities(t *testing.T) {
	hub := NewHub(8, 1)
	session := hub.Register("legacy", "group", 0, "test", 1)
	delivery, _ := reserveHubDelivery(t, hub, session, "legacy-action", "unknown", time.Now().Add(time.Minute))
	lease, err := hub.AcquireExecutionLease(context.Background(), session.ClientID, delivery, session.Generation)
	if err != nil {
		t.Fatalf("legacy acquire: %v", err)
	}
	if !hub.ReleaseExecutionLease(lease) {
		t.Fatal("release legacy lease")
	}
}

func TestPreferredSessionValidatesGroupAndAction(t *testing.T) {
	hub := NewHub(8, 1)
	hub.RegisterCapabilities("device", "group-a", 0, "ios", 1, true, []string{"content.search_notes"})

	if _, err := hub.pickSession("group-b", "device", "content.search_notes"); !errors.Is(err, ErrPreferredClientGroup) {
		t.Fatalf("cross-group error=%v", err)
	}
	if _, err := hub.pickSession("group-a", "device", "content.search_by_image"); !errors.Is(err, ErrActionNotSupported) {
		t.Fatalf("unsupported action error=%v", err)
	}
}

func TestLegacySessionAllowsUnknownActionsButExplicitEmptyDoesNot(t *testing.T) {
	legacy := NewHub(8, 1)
	legacy.Register("legacy", "xhs", 0, "ios", 1)
	if session, err := legacy.pickSession("xhs", "", "content.search_notes"); err != nil || session.ClientID != "legacy" {
		t.Fatalf("legacy selection session=%v err=%v", session, err)
	}

	strict := NewHub(8, 1)
	strict.RegisterCapabilities("strict", "xhs", 0, "ios", 1, true, []string{})
	if _, err := strict.pickSession("xhs", "", "content.search_notes"); !errors.Is(err, ErrNoCapableClient) {
		t.Fatalf("explicit empty error=%v", err)
	}
}

func TestRegisterWithoutCapabilitiesPreservesKnownActions(t *testing.T) {
	hub := NewHub(8, 1)
	hub.RegisterCapabilities("device", "xhs", 0, "ios", 1, true, []string{"content.search_notes"})
	hub.Register("device", "xhs", 0, "websocket", 1)
	session, ok := hub.Session("device")
	if !ok || !sessionSupportsAction(session, "content.search_notes") || !session.ActionsKnown {
		t.Fatalf("capabilities were lost: %+v", session)
	}
}

func TestCapabilityNarrowingDropsUnsupportedPending(t *testing.T) {
	hub := NewHub(8, 1)
	session := hub.RegisterCapabilities("device", "xhs", 0, "ios", 1, true,
		[]string{"content.search_notes", "content.search_by_image"})
	job := &Job{RequestID: "request", Action: "content.search_by_image", ClientID: "device"}
	prepareJobForQueue(session.Pending.ID(), job)
	waiter := make(chan JobResult, 1)
	hub.storeWaiter(job.RequestID, session.ClientID, waiter)
	if err := session.Pending.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	hub.RegisterCapabilities("device", "xhs", 0, "ios", 1, true, []string{"content.search_notes"})
	select {
	case result := <-waiter:
		if result.Status != "error" || result.Error == "" {
			t.Fatalf("result=%+v", result)
		}
	default:
		t.Fatal("pending caller was not completed")
	}
	if session.Pending.Len() != 0 {
		t.Fatalf("pending=%d", session.Pending.Len())
	}
}

func TestRequeueRejectsActionRemovedAfterDequeue(t *testing.T) {
	hub := NewHub(8, 1)
	session := hub.RegisterCapabilities("device", "xhs", 0, "ios", 1, true,
		[]string{"content.search_by_image"})
	job := &Job{RequestID: "request", Action: "content.search_by_image", ClientID: "device"}
	prepareJobForQueue(session.Pending.ID(), job)
	waiter := make(chan JobResult, 1)
	hub.storeWaiter(job.RequestID, session.ClientID, waiter)

	hub.RegisterCapabilities("device", "xhs", 0, "ios", 1, true, []string{})
	if err := hub.Requeue("device", job); !errors.Is(err, ErrActionNotSupported) {
		t.Fatalf("requeue error=%v", err)
	}
	select {
	case result := <-waiter:
		if result.Status != "error" {
			t.Fatalf("result=%+v", result)
		}
	default:
		t.Fatal("waiter was not completed")
	}
}
