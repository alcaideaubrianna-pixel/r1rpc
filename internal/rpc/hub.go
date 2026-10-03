package rpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Job struct {
	JobID      JobID           `json:"jobId,omitempty"`
	QueueID    QueueID         `json:"queueId,omitempty"`
	RequestID  string          `json:"requestId"`
	Group      string          `json:"group"`
	Action     string          `json:"action"`
	ClientID   string          `json:"clientId"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"createdAt"`
	DeadlineAt time.Time       `json:"deadlineAt"`
}

type JobResult struct {
	RequestID             string          `json:"requestId"`
	Status                string          `json:"status"`
	HTTPCode              int             `json:"httpCode"`
	Payload               json.RawMessage `json:"payload"`
	PayloadEncoding       string          `json:"payloadEncoding,omitempty"`
	PayloadRawSize        int             `json:"payloadRawSize,omitempty"`
	PayloadCompressedSize int             `json:"payloadCompressedSize,omitempty"`
	Error                 string          `json:"error"`
	LatencyMS             int64           `json:"latencyMs"`
	SessionGeneration     uint64          `json:"-"`
	SessionIncarnation    string          `json:"-"`
}

type ClientSession struct {
	ClientID           string
	Group              string
	UserID             int64
	Platform           string
	LastSeenAt         time.Time
	Pending            JobQueue
	MaxInFlight        int
	InFlight           int
	Generation         uint64
	SessionIncarnation string
	PendingBootstrap   string
	ActionsKnown       bool
	Actions            map[string]struct{}
	ActionLimits       map[string]int
	ActionInFlight     map[string]int
	dispatchReady      chan struct{}
	ProbeSupported     bool
	ProbeOk            bool
	ProbeLatencyMs     int64
	ProbeAt            time.Time
}

func (j *Job) ExpiredAt(now time.Time) bool {
	if j == nil || j.DeadlineAt.IsZero() {
		return false
	}
	return !now.Before(j.DeadlineAt)
}

type waiterEntry struct {
	ClientID           string
	SessionGeneration  uint64
	SessionIncarnation string
	ResultCh           chan JobResult
	DeadlineAt         time.Time
}

type ExecutionLease struct {
	RequestID          string
	ClientID           string
	Action             string
	QueueID            QueueID
	LeaseID            LeaseID
	Delivery           Delivery
	SessionGeneration  uint64
	SessionIncarnation string
	ExpiresAt          time.Time
	SentAt             time.Time
	queue              JobQueue
}

// WorkflowLease 在多个 RPC 调用之间独占一台设备，避免同一设备交错执行不同图片任务。
type WorkflowLease struct {
	ClientID string
	Token    string
}

type completedEntry struct {
	ClientID           string
	SessionGeneration  uint64
	SessionIncarnation string
	State              string
	FinishedAt         time.Time
}

type SubmitOutcome struct {
	Delivered bool
	Duplicate bool
	Late      bool
}

var (
	ErrResultClientMismatch       = errors.New("结果与请求的客户端不匹配")
	ErrResultNotWaiting           = errors.New("该请求未在等待结果（可能已超时）")
	ErrNoOnlineClient             = errors.New("分组内没有在线设备")
	ErrPreferredClientDown        = errors.New("指定的客户端不在线")
	ErrPreferredClientGroup       = errors.New("指定的客户端不属于请求分组")
	ErrActionNotSupported         = errors.New("设备不支持请求的 action")
	ErrNoCapableClient            = errors.New("分组内没有支持请求 action 的在线设备")
	ErrClientQueueFull            = errors.New("客户端队列已满")
	ErrJobExpired                 = errors.New("任务在入队前已过期")
	ErrGroupSaturated             = errors.New("分组内所有在线设备都已满负载")
	ErrClientSessionGone          = errors.New("客户端会话不存在")
	ErrExecutionLeaseExists       = errors.New("request 已绑定其他执行租约")
	ErrRequestTerminal            = errors.New("request 已进入终态")
	ErrSessionGenerationRequired  = errors.New("客户端 token 缺少 session generation")
	ErrSessionGenerationMismatch  = errors.New("客户端 session generation 已失效")
	ErrSessionIncarnationRequired = errors.New("客户端 token 缺少 session incarnation")
	ErrSessionIncarnationMismatch = errors.New("客户端 session incarnation 已失效")
	ErrBootstrapSessionMismatch   = errors.New("bootstrap token 已失效")
	ErrConnectionTokenRequired    = errors.New("仅 connection token 可执行该操作")
)

type Hub struct {
	mu                 sync.RWMutex
	pendingSize        int
	defaultMaxInFlight int
	sessions           map[string]*ClientSession
	queues             map[string]JobQueue
	queueIdleSince     map[string]time.Time
	queueIdleTTL       time.Duration
	groups             map[string]map[string]*ClientSession
	groupOrder         map[string][]string
	groupCursor        map[string]int
	waiters            map[string]waiterEntry
	completed          map[string]completedEntry
	executionLeases    map[string]*ExecutionLease
	workflowLeases     map[string]string
	leaseDuration      time.Duration
	executionGrace     time.Duration
	reaperInterval     time.Duration
	clock              func() time.Time
	defaultActionLimit int
	nextGeneration     uint64
	reaperStarted      bool
	reaperClosed       bool
	stopReaper         chan struct{}
	reaperDone         chan struct{}
}

func NewHub(pendingSize, defaultMaxInFlight int) *Hub {
	return NewHubWithTiming(pendingSize, defaultMaxInFlight, 2*time.Minute, time.Second, time.Now)
}

func NewHubWithTiming(pendingSize, defaultMaxInFlight int, leaseDuration, reaperInterval time.Duration, clock func() time.Time) *Hub {
	return NewHubWithExecutionTiming(pendingSize, defaultMaxInFlight, leaseDuration, reaperInterval, 30*time.Second, clock)
}

func NewHubWithExecutionTiming(pendingSize, defaultMaxInFlight int, leaseDuration, reaperInterval, executionGrace time.Duration, clock func() time.Time) *Hub {
	if pendingSize <= 0 {
		pendingSize = 2048
	}
	if defaultMaxInFlight <= 0 {
		defaultMaxInFlight = 256
	}
	if leaseDuration <= 0 {
		leaseDuration = 30 * time.Second
	}
	if reaperInterval <= 0 {
		reaperInterval = 5 * time.Second
	}
	if executionGrace <= 0 {
		executionGrace = 30 * time.Second
	}
	if clock == nil {
		clock = time.Now
	}
	return &Hub{
		pendingSize:        pendingSize,
		defaultMaxInFlight: defaultMaxInFlight,
		sessions:           map[string]*ClientSession{},
		queues:             map[string]JobQueue{},
		queueIdleSince:     map[string]time.Time{},
		queueIdleTTL:       10 * time.Minute,
		groups:             map[string]map[string]*ClientSession{},
		groupOrder:         map[string][]string{},
		groupCursor:        map[string]int{},
		waiters:            map[string]waiterEntry{},
		completed:          map[string]completedEntry{},
		executionLeases:    map[string]*ExecutionLease{},
		workflowLeases:     map[string]string{},
		leaseDuration:      leaseDuration,
		executionGrace:     executionGrace,
		reaperInterval:     reaperInterval,
		clock:              clock,
		defaultActionLimit: 1,
	}
}

func (h *Hub) Register(clientID, group string, userID int64, platform string, maxInFlight int) *ClientSession {
	return h.RegisterCapabilities(clientID, group, userID, platform, maxInFlight, false, nil)
}

func (h *Hub) RegisterCapabilities(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string) *ClientSession {
	return h.registerCapabilities(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, nil, false)
}

func (h *Hub) RegisterCapabilitiesWithLimits(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, actionLimits map[string]int) *ClientSession {
	return h.registerCapabilities(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, actionLimits, false)
}

func (h *Hub) RegisterBootstrapCapabilities(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, bootstrapID string) (*ClientSession, error) {
	session, err := h.registerCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, nil, false)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current, ok := h.sessions[clientID]
	if !ok || current != session {
		return nil, ErrClientSessionGone
	}
	current.PendingBootstrap = bootstrapID
	return current, nil
}

func (h *Hub) RegisterConnectionCapabilities(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string) *ClientSession {
	session, _ := h.RegisterConnectionCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions)
	return session
}

func (h *Hub) RegisterConnectionCapabilitiesE(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string) (*ClientSession, error) {
	return h.registerCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, nil, true)
}

func (h *Hub) BindConnectionCapabilitiesE(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, replacing bool) (*ClientSession, error) {
	if replacing {
		return h.RegisterConnectionCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions)
	}
	return h.bindInitialConnectionCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, "")
}

func (h *Hub) RegisterBootstrapConnectionCapabilitiesE(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, bootstrapID string) (*ClientSession, error) {
	return h.BindBootstrapConnectionCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, bootstrapID, false)
}

func (h *Hub) BindBootstrapConnectionCapabilitiesE(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, bootstrapID string, replacing bool) (*ClientSession, error) {
	if replacing {
		return nil, ErrBootstrapSessionMismatch
	}
	return h.bindInitialConnectionCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, bootstrapID)
}

func (h *Hub) ValidateBootstrap(clientID, bootstrapID string) error {
	// Legacy tokens have neither kind nor bootstrap ID. Their first-binding-only
	// restriction is enforced atomically by the WebSocket binding registry.
	if bootstrapID == "" {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	session, ok := h.sessions[clientID]
	if !ok {
		return ErrClientSessionGone
	}
	if session.PendingBootstrap != bootstrapID {
		return ErrBootstrapSessionMismatch
	}
	return nil
}

func (h *Hub) bindInitialConnectionCapabilitiesE(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, bootstrapID string) (*ClientSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	existing := h.sessions[clientID]
	if bootstrapID != "" && (existing == nil || existing.PendingBootstrap != bootstrapID) {
		return nil, ErrBootstrapSessionMismatch
	}
	incarnation, err := newSessionIncarnation()
	if err != nil {
		return nil, err
	}
	maxInFlight = h.normalizeMaxInFlight(maxInFlight)
	normalizedActions := normalizeActions(actions)
	if existing == nil {
		pending := h.queues[clientID]
		if pending == nil {
			pending = newMemoryJobQueueWithNow(h.pendingSize, h.clock)
			h.queues[clientID] = pending
		}
		h.nextGeneration++
		existing = &ClientSession{
			ClientID:           clientID,
			Group:              group,
			UserID:             userID,
			Platform:           platform,
			LastSeenAt:         h.clock(),
			Pending:            pending,
			MaxInFlight:        maxInFlight,
			Generation:         h.nextGeneration,
			SessionIncarnation: incarnation,
			ActionsKnown:       actionsKnown,
			Actions:            normalizedActions,
			ActionLimits:       normalizeActionLimits(normalizedActions, nil, maxInFlight, h.defaultActionLimit),
			ActionInFlight:     map[string]int{},
			dispatchReady:      make(chan struct{}),
		}
		h.sessions[clientID] = existing
	} else {
		if existing.Group != group {
			h.removeClientFromGroup(existing.Group, clientID)
		}
		if !actionsKnown && existing.ActionsKnown {
			actionsKnown = true
			normalizedActions = cloneActions(existing.Actions)
		}
		if actionsKnown {
			h.failUnsupportedPendingLocked(existing, normalizedActions)
		}
		h.nextGeneration++
		existing.Group = group
		existing.UserID = userID
		existing.Platform = platform
		existing.LastSeenAt = h.clock()
		existing.MaxInFlight = maxInFlight
		existing.Generation = h.nextGeneration
		existing.SessionIncarnation = incarnation
		existing.ActionsKnown = actionsKnown
		existing.Actions = normalizedActions
		existing.ActionLimits = normalizeActionLimits(normalizedActions, nil, maxInFlight, h.defaultActionLimit)
	}
	existing.PendingBootstrap = ""
	h.ensureGroup(group)[clientID] = existing
	h.ensureOrder(group, clientID)
	for requestID, waiter := range h.waiters {
		if waiter.ClientID != clientID {
			continue
		}
		waiter.SessionGeneration = existing.Generation
		waiter.SessionIncarnation = existing.SessionIncarnation
		h.waiters[requestID] = waiter
	}
	delete(h.queueIdleSince, clientID)
	h.signalDispatchLocked(existing)
	return existing, nil
}

func (h *Hub) registerCapabilities(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, actionLimits map[string]int, replaceConnection bool) *ClientSession {
	session, _ := h.registerCapabilitiesE(clientID, group, userID, platform, maxInFlight, actionsKnown, actions, actionLimits, replaceConnection)
	return session
}

func (h *Hub) registerCapabilitiesE(clientID, group string, userID int64, platform string, maxInFlight int, actionsKnown bool, actions []string, actionLimits map[string]int, replaceConnection bool) (*ClientSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	maxInFlight = h.normalizeMaxInFlight(maxInFlight)
	normalizedActions := normalizeActions(actions)
	var incarnation string
	if existing, ok := h.sessions[clientID]; ok {
		if !replaceConnection {
			if existing.Group != group {
				h.removeClientFromGroup(existing.Group, clientID)
			}
			existing.Group = group
			existing.UserID = userID
			existing.Platform = platform
			existing.LastSeenAt = h.clock()
			existing.MaxInFlight = maxInFlight
			if actionsKnown {
				existing.ActionsKnown = true
				existing.Actions = normalizedActions
				existing.ActionLimits = normalizeActionLimits(normalizedActions, actionLimits, maxInFlight, h.defaultActionLimit)
				h.failUnsupportedPendingLocked(existing, normalizedActions)
			}
			h.ensureGroup(group)[clientID] = existing
			h.ensureOrder(group, clientID)
			h.signalDispatchLocked(existing)
			return existing, nil
		}
		var err error
		incarnation, err = newSessionIncarnation()
		if err != nil {
			return nil, err
		}
		if !actionsKnown && existing.ActionsKnown {
			actionsKnown = true
			normalizedActions = cloneActions(existing.Actions)
		}
		if actionsKnown {
			h.failUnsupportedPendingLocked(existing, normalizedActions)
		}
		if err := h.terminateSessionLocked(existing, "session_replaced"); err != nil {
			return nil, err
		}
		h.removeClientFromGroup(existing.Group, clientID)
	} else {
		var err error
		incarnation, err = newSessionIncarnation()
		if err != nil {
			return nil, err
		}
	}

	h.nextGeneration++
	pending := h.queues[clientID]
	if pending == nil {
		pending = newMemoryJobQueueWithNow(h.pendingSize, h.clock)
		h.queues[clientID] = pending
	}
	delete(h.queueIdleSince, clientID)
	session := &ClientSession{
		ClientID:           clientID,
		Group:              group,
		UserID:             userID,
		Platform:           platform,
		LastSeenAt:         h.clock(),
		Pending:            pending,
		MaxInFlight:        maxInFlight,
		Generation:         h.nextGeneration,
		SessionIncarnation: incarnation,
		ActionsKnown:       actionsKnown,
		Actions:            normalizedActions,
		ActionLimits:       normalizeActionLimits(normalizedActions, actionLimits, maxInFlight, h.defaultActionLimit),
		ActionInFlight:     map[string]int{},
		dispatchReady:      make(chan struct{}),
	}
	h.sessions[clientID] = session
	h.ensureGroup(group)[clientID] = session
	h.ensureOrder(group, clientID)
	for requestID, waiter := range h.waiters {
		if waiter.ClientID != clientID {
			continue
		}
		waiter.SessionGeneration = session.Generation
		waiter.SessionIncarnation = session.SessionIncarnation
		h.waiters[requestID] = waiter
	}
	return session, nil
}

func (h *Hub) failUnsupportedPendingLocked(session *ClientSession, allowed map[string]struct{}) {
	removed := session.Pending.RemoveUnsupported(allowed, "capability_removed")
	for _, job := range removed {
		h.failWaiterLocked(session.ClientID, job, ErrActionNotSupported)
	}
}

func (h *Hub) failWaiterLocked(clientID string, job *Job, failure error) {
	waiter, ok := h.waiters[job.RequestID]
	if !ok || waiter.ClientID != clientID {
		return
	}
	delete(h.waiters, job.RequestID)
	h.completed[job.RequestID] = completedEntry{
		ClientID:           clientID,
		SessionGeneration:  waiter.SessionGeneration,
		SessionIncarnation: waiter.SessionIncarnation,
		State:              "completed",
		FinishedAt:         h.clock(),
	}
	select {
	case waiter.ResultCh <- JobResult{
		RequestID: job.RequestID,
		Status:    "error",
		HTTPCode:  409,
		Error:     failure.Error(),
	}:
	default:
	}
}

func (h *Hub) Touch(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if session, ok := h.sessions[clientID]; ok {
		session.LastSeenAt = time.Now()
	}
}

func (h *Hub) TouchGeneration(clientID string, generation uint64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, ok := h.sessions[clientID]
	if !ok || generation == 0 || session.Generation != generation {
		return false
	}
	session.LastSeenAt = h.clock()
	return true
}

func (h *Hub) TouchIncarnation(clientID, incarnation string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, ok := h.sessions[clientID]
	if !ok || incarnation == "" || session.SessionIncarnation != incarnation {
		return false
	}
	session.LastSeenAt = h.clock()
	return true
}

func (h *Hub) RecordProbe(clientID string, ok bool, latencyMs int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if session, found := h.sessions[clientID]; found {
		session.ProbeSupported = true
		session.ProbeOk = ok
		session.ProbeLatencyMs = latencyMs
		session.ProbeAt = time.Now()
	}
}

func (h *Hub) RecordProbeGeneration(clientID string, generation uint64, ok bool, latencyMs int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, found := h.sessions[clientID]
	if !found || generation == 0 || session.Generation != generation {
		return false
	}
	session.ProbeSupported = true
	session.ProbeOk = ok
	session.ProbeLatencyMs = latencyMs
	session.ProbeAt = h.clock()
	return true
}

func (h *Hub) RecordProbeIncarnation(clientID, incarnation string, ok bool, latencyMs int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, found := h.sessions[clientID]
	if !found || incarnation == "" || session.SessionIncarnation != incarnation {
		return false
	}
	session.ProbeSupported = true
	session.ProbeOk = ok
	session.ProbeLatencyMs = latencyMs
	session.ProbeAt = h.clock()
	return true
}

func (h *Hub) GroupOnlineCount(group string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.groups[group])
}

func (h *Hub) Session(clientID string) (*ClientSession, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	session, ok := h.sessions[clientID]
	return session, ok
}

func (h *Hub) Requeue(clientID string, job *Job) error {
	if job == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	session, ok := h.sessions[clientID]
	if !ok {
		return ErrClientSessionGone
	}
	if !sessionSupportsAction(session, job.Action) {
		h.failWaiterLocked(clientID, job, ErrActionNotSupported)
		return ErrActionNotSupported
	}
	if job.QueueID != session.Pending.ID() {
		return ErrQueueIdentityMismatch
	}
	return session.Pending.Requeue(job, "dispatch_failed")
}

// Unregister and generation helpers are retained for internal compatibility.
// Security-sensitive HTTP and WebSocket paths use the incarnation variants.
func (h *Hub) Unregister(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	_ = h.unregisterCurrentLocked(clientID, 0)
}

func (h *Hub) UnregisterGeneration(clientID string, generation uint64) {
	_ = h.UnregisterGenerationE(clientID, generation)
}

func (h *Hub) UnregisterGenerationE(clientID string, generation uint64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if generation == 0 {
		return ErrSessionGenerationRequired
	}
	return h.unregisterCurrentLocked(clientID, generation)
}

func (h *Hub) UnregisterIncarnationE(clientID, incarnation string) error {
	if incarnation == "" {
		return ErrSessionIncarnationRequired
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	session, ok := h.sessions[clientID]
	if !ok {
		return ErrClientSessionGone
	}
	if session.SessionIncarnation != incarnation {
		return ErrSessionIncarnationMismatch
	}
	return h.unregisterSessionLocked(session)
}

func (h *Hub) unregisterSessionLocked(session *ClientSession) error {
	if err := h.terminateSessionLocked(session, "session_unregistered"); err != nil {
		return err
	}
	delete(h.sessions, session.ClientID)
	delete(h.workflowLeases, session.ClientID)
	h.removeClientFromGroup(session.Group, session.ClientID)
	h.signalDispatchLocked(session)
	if session.Pending.Len() == 0 {
		h.queueIdleSince[session.ClientID] = h.clock()
	}
	return nil
}

// AcquireWorkflowLease 以轮询方式为完整业务工作流预留一台支持全部 action 的在线设备。
func (h *Hub) AcquireWorkflowLease(group string, actions ...string) (*WorkflowLease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	order := h.groupOrder[group]
	if len(order) == 0 {
		return nil, ErrNoOnlineClient
	}
	start := h.groupCursor[group]
	if start >= len(order) {
		start = 0
	}
	sawCapable := false
	for offset := 0; offset < len(order); offset++ {
		idx := (start + offset) % len(order)
		clientID := order[idx]
		session, ok := h.sessions[clientID]
		if !ok || session.Group != group {
			continue
		}
		capable := true
		for _, action := range actions {
			if !sessionSupportsAction(session, action) {
				capable = false
				break
			}
		}
		if !capable {
			continue
		}
		sawCapable = true
		if _, busy := h.workflowLeases[clientID]; busy {
			continue
		}
		token := newOpaqueID("workflow_")
		h.workflowLeases[clientID] = token
		h.groupCursor[group] = (idx + 1) % len(order)
		return &WorkflowLease{ClientID: clientID, Token: token}, nil
	}
	if !sawCapable {
		return nil, ErrNoCapableClient
	}
	return nil, ErrGroupSaturated
}

func (h *Hub) ReleaseWorkflowLease(lease *WorkflowLease) {
	if lease == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.workflowLeases[lease.ClientID] == lease.Token {
		delete(h.workflowLeases, lease.ClientID)
	}
}

func (h *Hub) unregisterCurrentLocked(clientID string, generation uint64) error {
	session, ok := h.sessions[clientID]
	if !ok {
		return ErrClientSessionGone
	}
	if generation != 0 && session.Generation != generation {
		return ErrSessionGenerationMismatch
	}
	return h.unregisterSessionLocked(session)
}

func (h *Hub) ValidateGeneration(clientID string, generation uint64) error {
	if generation == 0 {
		return ErrSessionGenerationRequired
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	session, ok := h.sessions[clientID]
	if !ok {
		return ErrClientSessionGone
	}
	if session.Generation != generation {
		return ErrSessionGenerationMismatch
	}
	return nil
}

func (h *Hub) ValidateIncarnation(clientID, incarnation string) error {
	if incarnation == "" {
		return ErrSessionIncarnationRequired
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	session, ok := h.sessions[clientID]
	if !ok {
		return ErrClientSessionGone
	}
	if session.SessionIncarnation != incarnation {
		return ErrSessionIncarnationMismatch
	}
	return nil
}

func (h *Hub) Invoke(ctx context.Context, group, preferredClient string, job *Job) (JobResult, string, error) {
	session, err := h.pickSession(group, preferredClient, job.Action)
	if err != nil {
		return JobResult{}, "", err
	}
	job.ClientID = session.ClientID

	waiter := make(chan JobResult, 1)
	h.storeWaiter(job.RequestID, session.ClientID, waiter)

	if ctx.Err() != nil {
		h.dropWaiter(job.RequestID)
		return JobResult{}, session.ClientID, ctx.Err()
	}
	if err := h.enqueueSelectedSession(session, group, job); err != nil {
		h.dropWaiter(job.RequestID)
		return JobResult{}, session.ClientID, err
	}

	select {
	case result := <-waiter:
		return result, session.ClientID, nil
	case <-ctx.Done():
		h.expireWaiter(job.RequestID)
		return JobResult{}, session.ClientID, ctx.Err()
	}
}

func (h *Hub) enqueueSelectedSession(selected *ClientSession, group string, job *Job) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	current, ok := h.sessions[selected.ClientID]
	if !ok || current != selected {
		return ErrClientSessionGone
	}
	if current.Group != group {
		return ErrPreferredClientGroup
	}
	if !sessionSupportsAction(current, job.Action) {
		return ErrActionNotSupported
	}
	prepareJobForQueue(current.Pending.ID(), job)
	if err := current.Pending.Enqueue(job); err != nil {
		return err
	}
	if waiter, ok := h.waiters[job.RequestID]; ok && waiter.ClientID == current.ClientID {
		waiter.DeadlineAt = job.DeadlineAt
		h.waiters[job.RequestID] = waiter
	}
	return nil
}

func (h *Hub) AcquireExecutionLease(ctx context.Context, clientID string, delivery Delivery, generation uint64) (*ExecutionLease, error) {
	return h.acquireExecutionLease(ctx, clientID, delivery, generation, "")
}

func (h *Hub) AcquireExecutionLeaseIncarnation(ctx context.Context, clientID string, delivery Delivery, incarnation string) (*ExecutionLease, error) {
	if incarnation == "" {
		return nil, ErrSessionIncarnationRequired
	}
	return h.acquireExecutionLease(ctx, clientID, delivery, 0, incarnation)
}

func (h *Hub) acquireExecutionLease(ctx context.Context, clientID string, delivery Delivery, generation uint64, incarnation string) (*ExecutionLease, error) {
	if delivery.Job == nil {
		return nil, ErrLeaseIdentityMismatch
	}
	for {
		h.mu.Lock()
		now := h.clock()
		session, ok := h.sessions[clientID]
		if !ok || (generation != 0 && session.Generation != generation) || (incarnation != "" && session.SessionIncarnation != incarnation) {
			h.mu.Unlock()
			return nil, ErrClientSessionGone
		}
		if delivery.QueueID != session.Pending.ID() || delivery.Job.ClientID != clientID {
			h.mu.Unlock()
			return nil, ErrQueueIdentityMismatch
		}
		action := strings.TrimSpace(delivery.Job.Action)
		if !sessionSupportsAction(session, action) {
			ackErr := session.Pending.Ack(delivery)
			h.failWaiterLocked(clientID, delivery.Job, ErrActionNotSupported)
			h.mu.Unlock()
			if ackErr != nil && !errors.Is(ackErr, ErrLeaseExpired) && !errors.Is(ackErr, ErrLeaseNotFound) {
				return nil, errors.Join(ErrActionNotSupported, ackErr)
			}
			return nil, ErrActionNotSupported
		}
		if delivery.Job.ExpiredAt(now) {
			ackErr := session.Pending.Ack(delivery)
			h.failWaiterLocked(clientID, delivery.Job, ErrJobExpired)
			h.mu.Unlock()
			if ackErr != nil && !errors.Is(ackErr, ErrLeaseExpired) && !errors.Is(ackErr, ErrLeaseNotFound) {
				return nil, errors.Join(ErrJobExpired, ackErr)
			}
			return nil, ErrJobExpired
		}
		if !now.Before(delivery.LeaseUntil) {
			h.mu.Unlock()
			return nil, ErrLeaseExpired
		}
		waiter, waiting := h.waiters[delivery.Job.RequestID]
		if !waiting || waiter.ClientID != clientID || waiter.SessionIncarnation != session.SessionIncarnation {
			h.mu.Unlock()
			return nil, ErrRequestTerminal
		}
		if existing, exists := h.executionLeases[delivery.Job.RequestID]; exists {
			if existing.LeaseID == delivery.LeaseID && existing.SessionIncarnation == session.SessionIncarnation {
				h.mu.Unlock()
				return existing, nil
			}
			h.mu.Unlock()
			return nil, ErrExecutionLeaseExists
		}
		limit := h.actionLimitLocked(session, action)
		if session.InFlight < session.MaxInFlight && session.ActionInFlight[action] < limit {
			expiresAt := now.Add(h.executionGrace)
			if deadline := delivery.Job.DeadlineAt; deadline.After(expiresAt) {
				expiresAt = deadline
			}
			renewed, err := session.Pending.Renew(delivery, expiresAt.Sub(now))
			if err != nil {
				h.mu.Unlock()
				return nil, err
			}
			delivery = renewed
			session.InFlight++
			session.ActionInFlight[action]++
			lease := &ExecutionLease{RequestID: delivery.Job.RequestID, ClientID: clientID, Action: action, QueueID: delivery.QueueID, LeaseID: delivery.LeaseID, Delivery: delivery, SessionGeneration: session.Generation, SessionIncarnation: session.SessionIncarnation, ExpiresAt: expiresAt, queue: session.Pending}
			h.executionLeases[lease.RequestID] = lease
			h.mu.Unlock()
			return lease, nil
		}
		waitCh := session.dispatchReady
		remaining := delivery.LeaseUntil.Sub(now)
		renewLead := h.leaseDuration / 2
		if renewLead <= 0 {
			renewLead = time.Millisecond
		}
		if remaining <= renewLead {
			renewFor := h.leaseDuration
			if deadline := delivery.Job.DeadlineAt; !deadline.IsZero() && deadline.Sub(now) < renewFor {
				renewFor = deadline.Sub(now)
			}
			if renewFor <= 0 {
				h.mu.Unlock()
				return nil, ErrJobExpired
			}
			renewed, err := session.Pending.Renew(delivery, renewFor)
			h.mu.Unlock()
			if err != nil {
				return nil, err
			}
			delivery = renewed
			continue
		}
		wakeAfter := remaining - renewLead
		h.mu.Unlock()
		timer := time.NewTimer(wakeAfter)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-waitCh:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
		}
	}
}

func (h *Hub) LeaseDuration() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.leaseDuration
}

func (h *Hub) ExecutionGrace() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.executionGrace
}

func (h *Hub) ReleaseExecutionLease(lease *ExecutionLease) bool {
	if lease == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current, ok := h.executionLeases[lease.RequestID]
	if !ok || current.LeaseID != lease.LeaseID || current.SessionIncarnation != lease.SessionIncarnation {
		return false
	}
	return h.releaseExecutionLeaseLocked(lease.RequestID)
}

func (h *Hub) MarkExecutionLeaseSent(lease *ExecutionLease) bool {
	if lease == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current, ok := h.executionLeases[lease.RequestID]
	if !ok || current.LeaseID != lease.LeaseID || current.SessionIncarnation != lease.SessionIncarnation {
		return false
	}
	if current.SentAt.IsZero() {
		current.SentAt = h.clock()
	}
	lease.SentAt = current.SentAt
	return true
}

func (h *Hub) releaseExecutionLeaseLocked(requestID string) bool {
	lease, ok := h.executionLeases[requestID]
	if !ok {
		return false
	}
	delete(h.executionLeases, requestID)
	if session, exists := h.sessions[lease.ClientID]; exists && session.SessionIncarnation == lease.SessionIncarnation {
		if session.InFlight > 0 {
			session.InFlight--
		}
		if session.ActionInFlight[lease.Action] > 0 {
			session.ActionInFlight[lease.Action]--
		}
		h.signalDispatchLocked(session)
	}
	return true
}

func (h *Hub) RequeueExecutionLease(lease *ExecutionLease, reason string) error {
	if lease == nil {
		return ErrLeaseNotFound
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current, ok := h.executionLeases[lease.RequestID]
	if !ok || current.LeaseID != lease.LeaseID || current.SessionIncarnation != lease.SessionIncarnation {
		return ErrLeaseNotFound
	}
	err := current.queue.RequeueDelivery(current.Delivery, reason)
	if err == nil || errors.Is(err, ErrLeaseExpired) || errors.Is(err, ErrLeaseNotFound) {
		h.releaseExecutionLeaseLocked(lease.RequestID)
	}
	return err
}

func (h *Hub) SubmitResult(clientID string, result JobResult) (SubmitOutcome, error) {
	if result.SessionIncarnation == "" {
		return SubmitOutcome{}, ErrSessionIncarnationRequired
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.clock()
	h.cleanupCompletedLocked(now)
	if lease, ok := h.executionLeases[result.RequestID]; ok {
		if lease.ClientID != clientID || result.SessionIncarnation != lease.SessionIncarnation {
			return SubmitOutcome{}, ErrResultClientMismatch
		}
		if err := lease.queue.Ack(lease.Delivery); err != nil {
			if errors.Is(err, ErrLeaseExpired) || errors.Is(err, ErrLeaseNotFound) {
				h.releaseExecutionLeaseLocked(result.RequestID)
				if lease.Delivery.Job.ExpiredAt(now) {
					h.finishExpiredWaiterLocked(lease, now)
				}
				return SubmitOutcome{Late: true}, nil
			}
			return SubmitOutcome{}, err
		}
		h.releaseExecutionLeaseLocked(result.RequestID)
		if waiter, waiting := h.waiters[result.RequestID]; waiting {
			if waiter.ClientID != clientID || waiter.SessionIncarnation != lease.SessionIncarnation {
				return SubmitOutcome{}, ErrResultClientMismatch
			}
			delete(h.waiters, result.RequestID)
			h.completed[result.RequestID] = completedEntry{ClientID: clientID, SessionGeneration: lease.SessionGeneration, SessionIncarnation: lease.SessionIncarnation, State: "completed", FinishedAt: now}
			select {
			case waiter.ResultCh <- result:
			default:
			}
			return SubmitOutcome{Delivered: true}, nil
		}
		h.completed[result.RequestID] = completedEntry{ClientID: clientID, SessionGeneration: lease.SessionGeneration, SessionIncarnation: lease.SessionIncarnation, State: "late", FinishedAt: now}
		return SubmitOutcome{Late: true}, nil
	}

	if completed, ok := h.completed[result.RequestID]; ok {
		if completed.ClientID != clientID || result.SessionIncarnation != completed.SessionIncarnation {
			return SubmitOutcome{}, ErrResultClientMismatch
		}
		switch completed.State {
		case "completed":
			return SubmitOutcome{Duplicate: true}, nil
		case "expired":
			return SubmitOutcome{Late: true}, nil
		case "late":
			return SubmitOutcome{Duplicate: true}, nil
		default:
			return SubmitOutcome{Duplicate: true}, nil
		}
	}

	return SubmitOutcome{}, ErrResultNotWaiting
}

func (h *Hub) OnlineClients() []ClientSession {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]ClientSession, 0, len(h.sessions))
	for _, item := range h.sessions {
		cloned := *item
		cloned.Actions = cloneActions(item.Actions)
		cloned.ActionLimits = make(map[string]int, len(item.ActionLimits))
		for action, limit := range item.ActionLimits {
			cloned.ActionLimits[action] = limit
		}
		cloned.ActionInFlight = make(map[string]int, len(item.ActionInFlight))
		for action, count := range item.ActionInFlight {
			cloned.ActionInFlight[action] = count
		}
		result = append(result, cloned)
	}
	return result
}

func (h *Hub) ensureGroup(group string) map[string]*ClientSession {
	groupSessions, ok := h.groups[group]
	if !ok {
		groupSessions = map[string]*ClientSession{}
		h.groups[group] = groupSessions
	}
	return groupSessions
}

func (h *Hub) ensureOrder(group, clientID string) {
	order := h.groupOrder[group]
	for _, existing := range order {
		if existing == clientID {
			return
		}
	}
	h.groupOrder[group] = append(order, clientID)
}

func (h *Hub) removeClientFromGroup(group, clientID string) {
	if groupSessions, ok := h.groups[group]; ok {
		delete(groupSessions, clientID)
		if len(groupSessions) == 0 {
			delete(h.groups, group)
		}
	}
	order := h.groupOrder[group]
	filtered := order[:0]
	for _, existing := range order {
		if existing != clientID {
			filtered = append(filtered, existing)
		}
	}
	if len(filtered) == 0 {
		delete(h.groupOrder, group)
		delete(h.groupCursor, group)
		return
	}
	h.groupOrder[group] = append([]string(nil), filtered...)
	if cursor := h.groupCursor[group]; cursor >= len(filtered) {
		h.groupCursor[group] = 0
	}
}

func (h *Hub) pickSession(group, preferredClient, action string) (*ClientSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if preferredClient != "" {
		session, ok := h.sessions[preferredClient]
		if !ok {
			return nil, ErrPreferredClientDown
		}
		if session.Group != group {
			return nil, ErrPreferredClientGroup
		}
		if !sessionSupportsAction(session, action) {
			return nil, ErrActionNotSupported
		}
		if session.Pending.Len() >= session.Pending.Cap() {
			return nil, ErrGroupSaturated
		}
		return session, nil
	}

	order := h.groupOrder[group]
	if len(order) == 0 {
		return nil, ErrNoOnlineClient
	}

	start := h.groupCursor[group]
	if start >= len(order) {
		start = 0
	}
	sawOnline := false
	sawCapable := false
	for offset := 0; offset < len(order); offset++ {
		idx := (start + offset) % len(order)
		clientID := order[idx]
		session, ok := h.sessions[clientID]
		if !ok || session.Group != group {
			continue
		}
		sawOnline = true
		if !sessionSupportsAction(session, action) {
			continue
		}
		sawCapable = true
		if session.Pending.Len() >= session.Pending.Cap() {
			continue
		}
		if _, busy := h.workflowLeases[clientID]; busy {
			continue
		}
		h.groupCursor[group] = (idx + 1) % len(order)
		return session, nil
	}
	if sawOnline {
		if !sawCapable {
			return nil, ErrNoCapableClient
		}
		return nil, ErrGroupSaturated
	}
	return nil, ErrNoOnlineClient
}

func normalizeActions(actions []string) map[string]struct{} {
	result := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		action = strings.TrimSpace(action)
		if action != "" {
			result[action] = struct{}{}
		}
	}
	return result
}

func cloneActions(actions map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(actions))
	for action := range actions {
		result[action] = struct{}{}
	}
	return result
}

func sessionSupportsAction(session *ClientSession, action string) bool {
	if session == nil || !session.ActionsKnown {
		return true
	}
	_, ok := session.Actions[strings.TrimSpace(action)]
	return ok
}

func (h *Hub) storeWaiter(requestID, clientID string, resultCh chan JobResult) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupCompletedLocked(h.clock())
	session := h.sessions[clientID]
	var generation uint64
	var incarnation string
	if session != nil {
		generation = session.Generation
		incarnation = session.SessionIncarnation
	}
	h.waiters[requestID] = waiterEntry{ClientID: clientID, SessionGeneration: generation, SessionIncarnation: incarnation, ResultCh: resultCh}
}

func (h *Hub) dropWaiter(requestID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.waiters, requestID)
}

func (h *Hub) expireWaiter(requestID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	waiter, ok := h.waiters[requestID]
	if !ok {
		return
	}
	delete(h.waiters, requestID)
	h.completed[requestID] = completedEntry{
		ClientID:           waiter.ClientID,
		SessionGeneration:  waiter.SessionGeneration,
		SessionIncarnation: waiter.SessionIncarnation,
		State:              "expired",
		FinishedAt:         h.clock(),
	}
	h.cleanupCompletedLocked(h.clock())
}

func (h *Hub) cleanupCompletedLocked(now time.Time) {
	const retention = 10 * time.Minute
	for requestID, item := range h.completed {
		if now.Sub(item.FinishedAt) > retention {
			delete(h.completed, requestID)
		}
	}
}

func (h *Hub) normalizeMaxInFlight(maxInFlight int) int {
	if maxInFlight <= 0 {
		return h.defaultMaxInFlight
	}
	if maxInFlight > 1024 {
		return 1024
	}
	return maxInFlight
}

func (h *Hub) Sweep(now time.Time) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if now.IsZero() {
		now = h.clock()
	}
	count := 0
	for _, queue := range h.queues {
		count += queue.Sweep(now)
	}
	for requestID, lease := range h.executionLeases {
		if now.Before(lease.ExpiresAt) {
			continue
		}
		h.releaseExecutionLeaseLocked(requestID)
		if lease.Delivery.Job.ExpiredAt(now) {
			h.finishExpiredWaiterLocked(lease, now)
		}
	}
	for requestID, waiter := range h.waiters {
		if waiter.DeadlineAt.IsZero() || now.Before(waiter.DeadlineAt) {
			continue
		}
		delete(h.waiters, requestID)
		h.completed[requestID] = completedEntry{ClientID: waiter.ClientID, SessionGeneration: waiter.SessionGeneration, SessionIncarnation: waiter.SessionIncarnation, State: "expired", FinishedAt: now}
		select {
		case waiter.ResultCh <- JobResult{RequestID: requestID, Status: "error", HTTPCode: 504, Error: "任务已过期"}:
		default:
		}
	}
	h.cleanupCompletedLocked(now)
	h.cleanupIdleQueuesLocked(now)
	return count
}

func (h *Hub) ConfigureQueueRetention(idleTTL time.Duration) {
	if idleTTL <= 0 {
		return
	}
	h.mu.Lock()
	h.queueIdleTTL = idleTTL
	h.mu.Unlock()
}

func (h *Hub) cleanupIdleQueuesLocked(now time.Time) {
	for clientID, queue := range h.queues {
		if _, online := h.sessions[clientID]; online || queue.Len() != 0 || h.clientHasActiveWorkLocked(clientID) {
			delete(h.queueIdleSince, clientID)
			continue
		}
		idleSince, tracked := h.queueIdleSince[clientID]
		if !tracked {
			h.queueIdleSince[clientID] = now
			continue
		}
		if h.queueIdleTTL > 0 && now.Sub(idleSince) >= h.queueIdleTTL {
			delete(h.queues, clientID)
			delete(h.queueIdleSince, clientID)
		}
	}
}

func (h *Hub) clientHasActiveWorkLocked(clientID string) bool {
	for _, waiter := range h.waiters {
		if waiter.ClientID == clientID {
			return true
		}
	}
	for _, lease := range h.executionLeases {
		if lease.ClientID == clientID {
			return true
		}
	}
	return false
}

func (h *Hub) finishExpiredWaiterLocked(lease *ExecutionLease, now time.Time) {
	waiter, ok := h.waiters[lease.RequestID]
	if !ok || waiter.ClientID != lease.ClientID || waiter.SessionIncarnation != lease.SessionIncarnation {
		return
	}
	delete(h.waiters, lease.RequestID)
	h.completed[lease.RequestID] = completedEntry{ClientID: lease.ClientID, SessionGeneration: lease.SessionGeneration, SessionIncarnation: lease.SessionIncarnation, State: "expired", FinishedAt: now}
	select {
	case waiter.ResultCh <- JobResult{RequestID: lease.RequestID, Status: "error", HTTPCode: 504, Error: "执行租约已过期"}:
	default:
	}
}

func (h *Hub) reaperLoop() {
	h.mu.RLock()
	interval := h.reaperInterval
	stop := h.stopReaper
	done := h.reaperDone
	h.mu.RUnlock()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(done)
	for {
		select {
		case <-ticker.C:
			h.Sweep(h.clock())
		case <-stop:
			return
		}
	}
}

func (h *Hub) ConfigureExecutionPolicy(leaseDuration, reaperInterval time.Duration, defaultActionLimit int) {
	h.ConfigureExecutionPolicyWithGrace(leaseDuration, reaperInterval, 0, defaultActionLimit)
}

func (h *Hub) ConfigureExecutionPolicyWithGrace(leaseDuration, reaperInterval, executionGrace time.Duration, defaultActionLimit int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if leaseDuration > 0 {
		h.leaseDuration = leaseDuration
	}
	if reaperInterval > 0 && !h.reaperStarted {
		h.reaperInterval = reaperInterval
	}
	if executionGrace > 0 {
		h.executionGrace = executionGrace
	}
	if defaultActionLimit > 0 {
		h.defaultActionLimit = defaultActionLimit
	}
}

func (h *Hub) StartReaper() {
	h.mu.Lock()
	if h.reaperStarted || h.reaperClosed {
		h.mu.Unlock()
		return
	}
	h.reaperStarted = true
	h.stopReaper = make(chan struct{})
	h.reaperDone = make(chan struct{})
	h.mu.Unlock()
	go h.reaperLoop()
}

func (h *Hub) Close() {
	h.mu.Lock()
	if h.reaperClosed {
		done := h.reaperDone
		h.mu.Unlock()
		if done != nil {
			<-done
		}
		return
	}
	h.reaperClosed = true
	if !h.reaperStarted {
		h.mu.Unlock()
		return
	}
	stop := h.stopReaper
	done := h.reaperDone
	close(stop)
	h.mu.Unlock()
	<-done
	h.mu.Lock()
	h.reaperStarted = false
	h.mu.Unlock()
}

func normalizeActionLimits(actions map[string]struct{}, limits map[string]int, deviceLimit, defaultLimit int) map[string]int {
	result := make(map[string]int, len(actions))
	for action := range actions {
		limit := defaultLimit
		if limits != nil && limits[action] > 0 {
			limit = limits[action]
		}
		if limit > deviceLimit {
			limit = deviceLimit
		}
		if limit < 1 {
			limit = 1
		}
		result[action] = limit
	}
	return result
}

func (h *Hub) actionLimitLocked(session *ClientSession, action string) int {
	limit := session.ActionLimits[action]
	if limit <= 0 {
		limit = h.defaultActionLimit
	}
	if limit <= 0 {
		limit = 1
	}
	if limit > session.MaxInFlight {
		limit = session.MaxInFlight
	}
	return limit
}

func (h *Hub) ActionLimit(clientID, action string) (int, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	session, ok := h.sessions[clientID]
	if !ok {
		return 0, false
	}
	return h.actionLimitLocked(session, strings.TrimSpace(action)), true
}

func (h *Hub) terminateSessionLocked(session *ClientSession, reason string) error {
	var terminateErrors []error
	for requestID, lease := range h.executionLeases {
		if lease.ClientID != session.ClientID || lease.SessionIncarnation != session.SessionIncarnation {
			continue
		}
		// 设备断联后的重试由持久化业务任务统一负责，RPC 层不得再次投递旧调用。
		err := lease.queue.Ack(lease.Delivery)
		switch {
		case err == nil, errors.Is(err, ErrLeaseExpired), errors.Is(err, ErrLeaseNotFound):
			h.releaseExecutionLeaseLocked(requestID)
		case errors.Is(err, ErrJobExpired):
			h.releaseExecutionLeaseLocked(requestID)
			h.finishExpiredWaiterLocked(lease, h.clock())
		default:
			terminateErrors = append(terminateErrors, err)
		}
	}
	for requestID, waiter := range h.waiters {
		if waiter.ClientID != session.ClientID || waiter.SessionIncarnation != session.SessionIncarnation {
			continue
		}
		delete(h.waiters, requestID)
		h.completed[requestID] = completedEntry{ClientID: session.ClientID, SessionGeneration: waiter.SessionGeneration, SessionIncarnation: waiter.SessionIncarnation, State: "completed", FinishedAt: h.clock()}
		select {
		case waiter.ResultCh <- JobResult{RequestID: requestID, Status: "error", HTTPCode: 503, Error: reason}:
		default:
		}
	}
	session.InFlight = 0
	session.ActionInFlight = map[string]int{}
	h.signalDispatchLocked(session)
	return errors.Join(terminateErrors...)
}

func (h *Hub) signalDispatchLocked(session *ClientSession) {
	if session == nil || session.dispatchReady == nil {
		return
	}
	close(session.dispatchReady)
	session.dispatchReady = make(chan struct{})
}

func newSessionIncarnation() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate session incarnation: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
