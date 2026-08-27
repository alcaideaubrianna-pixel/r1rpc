package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"r1rpc/internal/app"
	"r1rpc/internal/auth"
	"r1rpc/internal/config"
	"r1rpc/internal/rpc"
)

type reserveBlockingQueue struct {
	id      rpc.QueueID
	started chan struct{}
	once    sync.Once
}

func (q *reserveBlockingQueue) ID() rpc.QueueID { return q.id }
func (q *reserveBlockingQueue) Len() int        { return 0 }
func (q *reserveBlockingQueue) Cap() int        { return 1 }
func (q *reserveBlockingQueue) Enqueue(*rpc.Job) error {
	return nil
}
func (q *reserveBlockingQueue) Requeue(*rpc.Job, string) error { return nil }
func (q *reserveBlockingQueue) Dequeue(ctx context.Context) (*rpc.Job, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (q *reserveBlockingQueue) Reserve(ctx context.Context, _ time.Duration) (rpc.Delivery, error) {
	q.once.Do(func() { close(q.started) })
	<-ctx.Done()
	return rpc.Delivery{}, ctx.Err()
}
func (q *reserveBlockingQueue) Ack(rpc.Delivery) error { return nil }
func (q *reserveBlockingQueue) Renew(delivery rpc.Delivery, _ time.Duration) (rpc.Delivery, error) {
	return delivery, nil
}
func (q *reserveBlockingQueue) RequeueDelivery(rpc.Delivery, string) error { return nil }
func (q *reserveBlockingQueue) Sweep(time.Time) int                        { return 0 }
func (q *reserveBlockingQueue) RemoveUnsupported(map[string]struct{}, string) []*rpc.Job {
	return nil
}

func TestClientWSLoopFailureCancelsPeerLoopsImmediately(t *testing.T) {
	writeErr := errors.New("writer failed")
	readerExited := make(chan struct{})
	loops := []func(context.Context) error{
		func(ctx context.Context) error {
			<-ctx.Done()
			close(readerExited)
			return ctx.Err()
		},
		func(context.Context) error { return writeErr },
	}
	errs := coordinateClientWSLoops(context.Background(), func() {}, loops...)
	select {
	case <-readerExited:
	case <-time.After(time.Second):
		t.Fatal("writer failure did not cancel reader")
	}
	if len(errs) != len(loops) || !errors.Is(errs[1], writeErr) {
		t.Fatalf("loop errors=%v", errs)
	}
}

func TestShutdownClientConnectionsStopsAndWaitsForBindings(t *testing.T) {
	server := &Server{wsClients: newClientWSSessions()}
	done := make(chan struct{})
	canceled := make(chan struct{})
	server.wsClients.items["device"] = &clientWSBinding{
		cancel: func() {
			select {
			case <-canceled:
			default:
				close(canceled)
			}
		},
		done: done,
	}

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- server.ShutdownClientConnections(context.Background())
	}()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel binding")
	}
	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before binding stopped: %v", err)
	default:
	}
	close(done)
	if err := <-shutdownDone; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	server.wsClients.mu.Lock()
	closed := server.wsClients.closed
	server.wsClients.mu.Unlock()
	if !closed {
		t.Fatal("websocket admission remained open")
	}
}

func TestClientWSRegistrationAtomicallyBindsHubIncarnation(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	logical, err := hub.RegisterBootstrapCapabilities("device", "group", 0, "login", 1, false, nil, "bootstrap")
	if err != nil {
		t.Fatalf("register bootstrap: %v", err)
	}
	logicalIncarnation := logical.SessionIncarnation
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}
	var canceled atomic.Int64
	var workers sync.WaitGroup
	for range 50 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			claims := &auth.Claims{Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindBootstrap, BootstrapID: "bootstrap"}
			if current, ok := server.wsClients.current("device"); ok {
				claims.TokenKind = auth.TokenKindConnection
				claims.SessionIncarnation = current.incarnation
			}
			_, _, err := server.registerClientWSBinding(claims, nil, func() { canceled.Add(1) }, func(replacing bool) (*rpc.ClientSession, error) {
				return hub.BindConnectionCapabilitiesE("device", "group", 0, "websocket", 1, false, nil, replacing)
			})
			if err != nil {
				// A token can become stale between the snapshot above and atomic admission.
				if !errors.Is(err, rpc.ErrSessionIncarnationMismatch) {
					t.Errorf("register: %v", err)
				}
			}
		}()
	}
	workers.Wait()
	current, ok := hub.Session("device")
	if !ok {
		t.Fatal("hub session missing")
	}
	binding, ok := server.wsClients.current("device")
	if !ok || binding.incarnation != current.SessionIncarnation {
		t.Fatalf("binding=%+v ok=%v hub incarnation=%q", binding, ok, current.SessionIncarnation)
	}
	if current.SessionIncarnation == logicalIncarnation {
		t.Fatal("first websocket bind did not create a connection incarnation")
	}
	if got := canceled.Load(); got > 49 {
		t.Fatalf("superseded cancellations=%d want <=49", got)
	}
}

func TestHTTPClientLifecycleAcceptsOnlyCurrentConnectionToken(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	old := hub.RegisterConnectionCapabilities("device", "group", 0, "websocket", 1, false, nil)
	current := hub.RegisterConnectionCapabilities("device", "group", 0, "websocket", 1, false, nil)
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}

	for _, test := range []struct {
		name   string
		claims *auth.Claims
	}{
		{name: "legacy", claims: &auth.Claims{Role: "client", ClientID: "device", Group: "group"}},
		{name: "bootstrap", claims: &auth.Claims{Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindBootstrap}},
		{name: "stale", claims: &auth.Claims{Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindConnection, SessionIncarnation: old.SessionIncarnation}},
	} {
		t.Run(test.name+" logout", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.handleClientLogout(recorder, httptest.NewRequest(http.MethodPost, "/api/client/logout", nil), test.claims)
			if recorder.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if active, ok := hub.Session("device"); !ok || active.SessionIncarnation != current.SessionIncarnation {
				t.Fatal("stale logout removed current session")
			}
		})
		t.Run(test.name+" result", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/client/result", io.NopCloser(strings.NewReader(`{"requestId":"request"}`)))
			server.handleClientResult(recorder, request, test.claims)
			if recorder.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestConnectionTokenCarriesKindIncarnationAndGeneration(t *testing.T) {
	tokens := auth.NewTokenManager("test-secret")
	application := &app.App{Tokens: tokens}
	token, err := application.IssueClientConnectionToken(&auth.Claims{Username: "device", Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindBootstrap}, "incarnation", 42)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := tokens.Parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.SessionGeneration != 42 {
		t.Fatalf("generation=%d", claims.SessionGeneration)
	}
	if claims.SessionIncarnation != "incarnation" {
		t.Fatalf("incarnation=%q", claims.SessionIncarnation)
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	if fields["tokenKind"] != "connection" {
		t.Fatalf("token kind=%v want connection", fields["tokenKind"])
	}
}

func TestClientWSTokenKindsHaveDisjointClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims auth.Claims
		kind   string
		legacy bool
		want   error
	}{
		{name: "legacy", claims: auth.Claims{}, kind: auth.TokenKindBootstrap, legacy: true},
		{name: "legacy with incarnation", claims: auth.Claims{SessionIncarnation: "incarnation"}, want: rpc.ErrSessionIncarnationMismatch},
		{name: "bootstrap", claims: auth.Claims{TokenKind: auth.TokenKindBootstrap, BootstrapID: "bootstrap"}, kind: auth.TokenKindBootstrap},
		{name: "bootstrap without id", claims: auth.Claims{TokenKind: auth.TokenKindBootstrap}, want: rpc.ErrBootstrapSessionMismatch},
		{name: "bootstrap with incarnation", claims: auth.Claims{TokenKind: auth.TokenKindBootstrap, BootstrapID: "bootstrap", SessionIncarnation: "incarnation"}, want: rpc.ErrBootstrapSessionMismatch},
		{name: "connection", claims: auth.Claims{TokenKind: auth.TokenKindConnection, SessionIncarnation: "incarnation"}, kind: auth.TokenKindConnection},
		{name: "connection without incarnation", claims: auth.Claims{TokenKind: auth.TokenKindConnection}, want: rpc.ErrSessionIncarnationRequired},
		{name: "connection with bootstrap id", claims: auth.Claims{TokenKind: auth.TokenKindConnection, BootstrapID: "bootstrap", SessionIncarnation: "incarnation"}, want: rpc.ErrSessionIncarnationMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kind, legacy, err := clientWSTokenKind(&test.claims)
			if !errors.Is(err, test.want) || kind != test.kind || legacy != test.legacy {
				t.Fatalf("kind=%q legacy=%v err=%v", kind, legacy, err)
			}
		})
	}
}

func TestHTTPClientLifecycleRejectsBootstrapToken(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	session := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}
	claims := &auth.Claims{
		Role:               "client",
		ClientID:           session.ClientID,
		Group:              session.Group,
		TokenKind:          auth.TokenKindBootstrap,
		SessionIncarnation: session.SessionIncarnation,
	}
	recorder := httptest.NewRecorder()
	server.handleClientLogout(recorder, httptest.NewRequest(http.MethodPost, "/api/client/logout", nil), claims)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("bootstrap logout status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestWSAdmissionRejectsConnectionTokenAfterHubReconstruction(t *testing.T) {
	tokens := auth.NewTokenManager("test-secret")
	oldHub := rpc.NewHub(8, 1)
	old := oldHub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	token, err := (&app.App{Tokens: tokens}).IssueClientConnectionToken(&auth.Claims{
		Role:     "client",
		ClientID: old.ClientID,
		Group:    old.Group,
	}, old.SessionIncarnation, old.Generation)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	rebuiltHub := rpc.NewHub(8, 1)
	rebuiltHub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	server := &Server{App: &app.App{Tokens: tokens, Hub: rebuiltHub}, wsClients: newClientWSSessions()}
	request := httptest.NewRequest(http.MethodGet, "/api/client/ws?token="+token, nil)
	if _, err := server.verifyClientWSClaims(request); err == nil {
		t.Fatal("stale connection token passed websocket admission after hub reconstruction")
	}
}

func TestLegacyTokenIsBootstrapOnlyForFirstWebSocket(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}
	legacy := &auth.Claims{Role: "client", ClientID: "device", Group: "group", MaxInFlight: 1}

	_, first, err := server.registerClientWSBinding(legacy, nil, func() {}, func(replacing bool) (*rpc.ClientSession, error) {
		return hub.BindConnectionCapabilitiesE("device", "group", 0, "websocket", 1, false, nil, replacing)
	})
	if err != nil {
		t.Fatalf("first legacy websocket: %v", err)
	}
	called := false
	_, _, err = server.registerClientWSBinding(legacy, nil, func() {}, func(bool) (*rpc.ClientSession, error) {
		called = true
		return nil, nil
	})
	if err == nil || called {
		t.Fatalf("legacy token replaced active binding: err=%v called=%v", err, called)
	}
	current, _ := hub.Session("device")
	if current.SessionIncarnation != first.SessionIncarnation {
		t.Fatal("legacy replacement changed the active session")
	}
}

func TestLegacyTokenCannotBeReusedAfterDisconnect(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}
	legacy := &auth.Claims{Role: "client", ClientID: "device", Group: "group", MaxInFlight: 1}

	connID, _, err := server.registerClientWSBinding(legacy, nil, func() {}, func(replacing bool) (*rpc.ClientSession, error) {
		return hub.BindConnectionCapabilitiesE("device", "group", 0, "websocket", 1, false, nil, replacing)
	})
	if err != nil {
		t.Fatalf("first legacy websocket: %v", err)
	}
	server.wsClients.clearIfCurrent("device", connID)
	called := false
	_, _, err = server.registerClientWSBinding(legacy, nil, func() {}, func(bool) (*rpc.ClientSession, error) {
		called = true
		return nil, nil
	})
	if !errors.Is(err, rpc.ErrBootstrapSessionMismatch) || called {
		t.Fatalf("legacy token reuse err=%v called=%v", err, called)
	}
}

func TestStaleConnectionTokenCannotReplaceCurrentBinding(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	stale := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	current := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}
	var canceled atomic.Bool
	server.wsClients.items["device"] = &clientWSBinding{connID: 1, incarnation: current.SessionIncarnation, cancel: func() { canceled.Store(true) }}
	claims := &auth.Claims{Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindConnection, SessionIncarnation: stale.SessionIncarnation}
	called := false

	_, _, err := server.registerClientWSBinding(claims, nil, func() {}, func(bool) (*rpc.ClientSession, error) {
		called = true
		return nil, nil
	})
	if !errors.Is(err, rpc.ErrSessionIncarnationMismatch) || called || canceled.Load() {
		t.Fatalf("stale admission err=%v called=%v canceled=%v", err, called, canceled.Load())
	}
	binding, ok := server.wsClients.current("device")
	if !ok || binding.incarnation != current.SessionIncarnation {
		t.Fatalf("current binding was replaced: %+v ok=%v", binding, ok)
	}
}

func TestFirstWebSocketBindDoesNotFailLoginWindowWaiter(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	logical, err := hub.RegisterBootstrapCapabilities("device", "group", 0, "login", 1, false, nil, "bootstrap")
	if err != nil {
		t.Fatalf("register bootstrap: %v", err)
	}
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan rpc.JobResult, 1)
	go func() {
		result, _, _ := hub.Invoke(ctx, "group", "device", &rpc.Job{RequestID: "login-window", Action: "action", DeadlineAt: time.Now().Add(time.Minute)})
		done <- result
	}()
	deadline := time.Now().Add(time.Second)
	for logical.Pending.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	claims := &auth.Claims{Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindBootstrap, BootstrapID: "bootstrap"}
	_, bound, err := server.registerClientWSBinding(claims, nil, func() {}, func(replacing bool) (*rpc.ClientSession, error) {
		return hub.BindBootstrapConnectionCapabilitiesE("device", "group", 0, "websocket", 1, false, nil, claims.BootstrapID, replacing)
	})
	if err != nil {
		t.Fatalf("first bind: %v", err)
	}
	if bound != logical {
		t.Fatal("first bind replaced rather than upgraded the logical session")
	}
	select {
	case result := <-done:
		t.Fatalf("first bind completed waiter unexpectedly: %+v", result)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestClientWSReplacementStopsOldBindingBeforeHubMigration(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}

	var oldStopped atomic.Bool
	current, _ := hub.Session("device")
	server.wsClients.items["device"] = &clientWSBinding{incarnation: current.SessionIncarnation, cancel: func() { oldStopped.Store(true) }}
	claims := &auth.Claims{Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindConnection, SessionIncarnation: current.SessionIncarnation}
	_, _, err := server.registerClientWSBinding(claims, nil, func() {}, func(replacing bool) (*rpc.ClientSession, error) {
		if !oldStopped.Load() {
			return nil, errors.New("old binding still running during hub migration")
		}
		return hub.BindConnectionCapabilitiesE("device", "group", 0, "test", 1, false, nil, replacing)
	})
	if err != nil {
		t.Fatalf("replacement ordering: %v", err)
	}
}

func TestClientWSReplacementPublishesBindingAtomically(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	current := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	server := &Server{App: &app.App{Config: &config.Config{}, Hub: hub}, wsClients: newClientWSSessions()}
	server.wsClients.items["device"] = &clientWSBinding{connID: 1, incarnation: current.SessionIncarnation, cancel: func() {}}
	claims := &auth.Claims{Role: "client", ClientID: "device", Group: "group", TokenKind: auth.TokenKindConnection, SessionIncarnation: current.SessionIncarnation}

	registerEntered := make(chan struct{})
	continueRegister := make(chan struct{})
	registerDone := make(chan *rpc.ClientSession, 1)
	go func() {
		_, session, err := server.registerClientWSBinding(claims, nil, func() {}, func(replacing bool) (*rpc.ClientSession, error) {
			close(registerEntered)
			<-continueRegister
			return hub.BindConnectionCapabilitiesE("device", "group", 0, "test", 1, false, nil, replacing)
		})
		if err != nil {
			t.Errorf("replace binding: %v", err)
		}
		registerDone <- session
	}()
	<-registerEntered

	lookupDone := make(chan clientWSBinding, 1)
	go func() {
		binding, _ := server.wsClients.current("device")
		lookupDone <- binding
	}()
	select {
	case binding := <-lookupDone:
		t.Fatalf("binding became observable during migration: %+v", binding)
	case <-time.After(20 * time.Millisecond):
	}
	close(continueRegister)
	session := <-registerDone
	binding := <-lookupDone
	if session == nil || binding.incarnation != session.SessionIncarnation || binding.incarnation == current.SessionIncarnation {
		t.Fatalf("binding=%+v session=%+v old=%q", binding, session, current.SessionIncarnation)
	}
}

func TestStaleWSWriterDoesNotReserveSharedQueue(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	old := hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)

	invokeCtx, cancelInvoke := context.WithCancel(context.Background())
	defer cancelInvoke()
	go func() {
		_, _, _ = hub.Invoke(invokeCtx, old.Group, old.ClientID, &rpc.Job{
			RequestID:  "stale-reserve",
			Action:     "action",
			DeadlineAt: time.Now().Add(time.Minute),
		})
	}()
	deadline := time.Now().Add(time.Second)
	for old.Pending.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if old.Pending.Len() == 0 {
		t.Fatal("job was not queued")
	}
	hub.RegisterConnectionCapabilities("device", "group", 0, "test", 1, false, nil)
	reservedBefore := countQueueEvents(old.Pending.Events(), rpc.QueueEventReserved)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_ = clientWSDispatchLoop(ctx, hub, old, func(*rpc.Job) error {
		t.Fatal("stale writer sent a job")
		return nil
	})
	reservedAfter := countQueueEvents(old.Pending.Events(), rpc.QueueEventReserved)
	if reservedAfter != reservedBefore {
		t.Fatalf("stale writer reserved shared queue: before=%d after=%d", reservedBefore, reservedAfter)
	}
}

func countQueueEvents(events []rpc.QueueEvent, eventType rpc.QueueEventType) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}
func (q *reserveBlockingQueue) Events() []rpc.QueueEvent { return nil }

type enqueueSignalQueue struct {
	rpc.JobQueue
	enqueued chan struct{}
	once     sync.Once
}

type requeueErrorQueue struct {
	rpc.JobQueue
	err error
}

func (q *requeueErrorQueue) RequeueDelivery(rpc.Delivery, string) error { return q.err }

func (q *enqueueSignalQueue) Enqueue(job *rpc.Job) error {
	err := q.JobQueue.Enqueue(job)
	if err == nil {
		q.once.Do(func() { close(q.enqueued) })
	}
	return err
}

func TestClientWSDispatchLoopDoesNotOccupySlotForEmptyQueue(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	session := hub.Register("device", "group", 0, "test", 1)
	queue := &reserveBlockingQueue{id: session.Pending.ID(), started: make(chan struct{})}
	session.Pending = queue

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- clientWSDispatchLoop(ctx, hub, session, func(*rpc.Job) error {
			t.Error("empty queue attempted a write")
			return nil
		})
	}()
	<-queue.started
	if session.InFlight != 0 || len(session.ActionInFlight) != 0 {
		t.Fatalf("empty queue occupied execution slots: device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("dispatch loop cancellation: %v", err)
	}
}

func TestClientWSDispatchLoopRequeuesOnWriteFailure(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	session := hub.Register("device", "group", 0, "test", 1)
	queue := &enqueueSignalQueue{JobQueue: session.Pending, enqueued: make(chan struct{})}
	session.Pending = queue

	invokeCtx, cancelInvoke := context.WithCancel(context.Background())
	defer cancelInvoke()
	invokeDone := make(chan error, 1)
	go func() {
		_, _, err := hub.Invoke(invokeCtx, session.Group, session.ClientID, &rpc.Job{
			RequestID:  "write-failure",
			Action:     "action",
			DeadlineAt: time.Now().Add(time.Minute),
		})
		invokeDone <- err
	}()
	<-queue.enqueued

	writeErr := errors.New("forced websocket write failure")
	err := clientWSDispatchLoop(context.Background(), hub, session, func(*rpc.Job) error { return writeErr })
	if !errors.Is(err, writeErr) {
		t.Fatalf("dispatch error=%v", err)
	}
	if session.InFlight != 0 || session.ActionInFlight["action"] != 0 {
		t.Fatalf("write failure leaked slots: device=%d actions=%v", session.InFlight, session.ActionInFlight)
	}
	if queue.Len() != 1 {
		t.Fatalf("requeued depth=%d, want 1", queue.Len())
	}
	events := queue.Events()
	if events[len(events)-1].Type != rpc.QueueEventRequeued || events[len(events)-1].Reason != "dispatch_write_failed" {
		t.Fatalf("last queue event=%+v", events[len(events)-1])
	}

	cancelInvoke()
	if err := <-invokeDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("invoke cancellation error=%v", err)
	}
}

func TestClientWSDispatchLoopReturnsRequeueFailure(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	session := hub.Register("device", "group", 0, "test", 1)
	enqueued := &enqueueSignalQueue{JobQueue: session.Pending, enqueued: make(chan struct{})}
	requeueErr := errors.New("requeue failed")
	queue := &requeueErrorQueue{JobQueue: enqueued, err: requeueErr}
	session.Pending = queue

	invokeCtx, cancelInvoke := context.WithCancel(context.Background())
	defer cancelInvoke()
	go func() {
		_, _, _ = hub.Invoke(invokeCtx, session.Group, session.ClientID, &rpc.Job{
			RequestID:  "requeue-failure",
			Action:     "action",
			DeadlineAt: time.Now().Add(time.Minute),
		})
	}()
	<-enqueued.enqueued

	writeErr := errors.New("write failed")
	err := clientWSDispatchLoop(context.Background(), hub, session, func(*rpc.Job) error { return writeErr })
	if !errors.Is(err, writeErr) || !errors.Is(err, requeueErr) {
		t.Fatalf("dispatch error=%v", err)
	}
}

func TestClientWSDispatchLoopKeepsLeaseAfterSuccessfulWriteUntilResult(t *testing.T) {
	hub := rpc.NewHub(8, 1)
	session := hub.Register("device", "group", 0, "test", 1)
	queue := &enqueueSignalQueue{JobQueue: session.Pending, enqueued: make(chan struct{})}
	session.Pending = queue

	invokeCtx, cancelInvoke := context.WithCancel(context.Background())
	defer cancelInvoke()
	invokeDone := make(chan error, 1)
	go func() {
		_, _, err := hub.Invoke(invokeCtx, session.Group, session.ClientID, &rpc.Job{
			RequestID:  "write-success",
			Action:     "action",
			DeadlineAt: time.Now().Add(time.Minute),
		})
		invokeDone <- err
	}()
	<-queue.enqueued

	dispatchCtx, cancelDispatch := context.WithCancel(context.Background())
	err := clientWSDispatchLoop(dispatchCtx, hub, session, func(*rpc.Job) error {
		cancelDispatch()
		return nil
	})
	if err != nil {
		t.Fatalf("dispatch loop: %v", err)
	}
	if session.InFlight != 1 || session.ActionInFlight["action"] != 1 || queue.Len() != 1 {
		t.Fatalf("successful write did not retain lease: device=%d actions=%v depth=%d", session.InFlight, session.ActionInFlight, queue.Len())
	}

	outcome, err := hub.SubmitResult(session.ClientID, rpc.JobResult{RequestID: "write-success", Status: "success", SessionIncarnation: session.SessionIncarnation})
	if err != nil || !outcome.Delivered {
		t.Fatalf("submit outcome=%+v err=%v", outcome, err)
	}
	if err := <-invokeDone; err != nil {
		t.Fatalf("invoke result error=%v", err)
	}
	if session.InFlight != 0 || queue.Len() != 0 {
		t.Fatalf("result did not ack/release: inFlight=%d depth=%d", session.InFlight, queue.Len())
	}
}
