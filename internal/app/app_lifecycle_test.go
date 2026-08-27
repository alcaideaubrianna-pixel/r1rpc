package app

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"r1rpc/internal/config"
	"r1rpc/internal/rpc"
)

func TestStartBackgroundJobsStartsHubReaper(t *testing.T) {
	hub := rpc.NewHubWithTiming(8, 1, 10*time.Millisecond, time.Millisecond, time.Now)
	application := &App{
		Config:       &config.Config{PersistWorkers: 1},
		Hub:          hub,
		persistCh:    make(chan persistTask),
		ProbeHistory: newProbeHistory(nil),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	application.StartBackgroundJobs(ctx)
	defer application.Close()

	session := hub.Register("device", "group", 0, "test", 1)
	invokeCtx, cancelInvoke := context.WithCancel(context.Background())
	defer cancelInvoke()
	go func() {
		_, _, _ = hub.Invoke(invokeCtx, session.Group, session.ClientID, &rpc.Job{
			RequestID:  "app-reaper",
			Action:     "action",
			DeadlineAt: time.Now().Add(time.Second),
		})
	}()
	deadline := time.Now().Add(time.Second)
	for session.Pending.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if session.Pending.Len() == 0 {
		t.Fatal("invoke was not queued")
	}
	if _, err := session.Pending.Reserve(context.Background(), 10*time.Millisecond); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	for time.Now().Before(deadline) {
		for _, event := range session.Pending.Events() {
			if event.Type == rpc.QueueEventLeaseExpired {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("app-started reaper did not sweep reservation")
}

func TestCloseDrainsPersistTasksAfterRunContextCancellation(t *testing.T) {
	var processed atomic.Int64
	application := &App{
		Config:       &config.Config{PersistWorkers: 1},
		Hub:          rpc.NewHub(8, 1),
		persistCh:    make(chan persistTask, 128),
		ProbeHistory: newProbeHistory(nil),
		persistBatchRunner: func(tasks []persistTask) error {
			processed.Add(int64(len(tasks)))
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	application.StartBackgroundJobs(ctx)
	cancel()
	time.Sleep(20 * time.Millisecond)

	for index := range 100 {
		application.enqueuePersist(persistTask{Kind: persistTaskKind("test"), RequestID: fmt.Sprintf("request-%d", index)})
	}
	if err := application.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if got := len(application.persistCh); got != 0 {
		t.Fatalf("persist tasks left after close=%d", got)
	}
	if got := processed.Load(); got != 100 {
		t.Fatalf("processed persist tasks=%d want=100", got)
	}

	application.enqueuePersist(persistTask{Kind: persistTaskKind("after-close")})
	if got := len(application.persistCh); got != 0 {
		t.Fatalf("producer wrote after close: queued=%d", got)
	}
}
