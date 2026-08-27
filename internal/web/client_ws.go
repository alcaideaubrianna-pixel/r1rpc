package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"r1rpc/internal/auth"
	"r1rpc/internal/rpc"
)

const wsMaxMessageBytes = 20 << 20

type wsEnvelope struct {
	Type               string         `json:"type"`
	Job                *rpc.Job       `json:"job,omitempty"`
	Result             *rpc.JobResult `json:"result,omitempty"`
	RequestID          string         `json:"requestId,omitempty"`
	OK                 bool           `json:"ok,omitempty"`
	State              string         `json:"state,omitempty"`
	Error              string         `json:"error,omitempty"`
	ClientID           string         `json:"clientId,omitempty"`
	Group              string         `json:"group,omitempty"`
	ServerID           string         `json:"serverId,omitempty"`
	Time               string         `json:"time,omitempty"`
	MaxInFlight        int            `json:"maxInFlight,omitempty"`
	Token              string         `json:"token,omitempty"`
	SessionGeneration  uint64         `json:"sessionGeneration,omitempty"`
	SessionIncarnation string         `json:"sessionIncarnation,omitempty"`
}

type clientWSSessions struct {
	admitMu sync.Mutex
	mu      sync.Mutex
	nextID  uint64
	items   map[string]*clientWSBinding
	legacy  map[string]struct{}
	closed  bool
}

type clientWSBinding struct {
	connID      uint64
	generation  uint64
	incarnation string
	cancel      context.CancelFunc
	conn        *websocket.Conn
	done        <-chan struct{}
}

func newClientWSSessions() *clientWSSessions {
	return &clientWSSessions{
		items:  map[string]*clientWSBinding{},
		legacy: map[string]struct{}{},
	}
}

func (s *Server) registerClientWSBinding(claims *auth.Claims, conn *websocket.Conn, cancel context.CancelFunc, register func(replacing bool) (*rpc.ClientSession, error)) (uint64, *rpc.ClientSession, error) {
	return s.registerClientWSBindingWithDone(claims, conn, cancel, nil, register)
}

func (s *Server) registerClientWSBindingWithDone(claims *auth.Claims, conn *websocket.Conn, cancel context.CancelFunc, done <-chan struct{}, register func(replacing bool) (*rpc.ClientSession, error)) (uint64, *rpc.ClientSession, error) {
	if claims == nil {
		return 0, nil, fmt.Errorf("客户端 claims 不能为空")
	}
	clientID := claims.ClientID
	m := s.wsClients
	m.admitMu.Lock()
	defer m.admitMu.Unlock()
	tokenKind, legacy, err := clientWSTokenKind(claims)
	if err != nil {
		return 0, nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, nil, fmt.Errorf("WebSocket 服务正在关闭")
	}

	previous := m.items[clientID]
	replacing := previous != nil
	legacyKey := ""
	if legacy {
		legacyKey = legacyClientTokenKey(claims)
		if _, used := m.legacy[legacyKey]; used {
			return 0, nil, rpc.ErrBootstrapSessionMismatch
		}
	}
	switch tokenKind {
	case auth.TokenKindConnection:
		if err := s.App.Hub.ValidateIncarnation(clientID, claims.SessionIncarnation); err != nil {
			return 0, nil, err
		}
		if previous != nil && previous.incarnation != claims.SessionIncarnation {
			return 0, nil, rpc.ErrSessionIncarnationMismatch
		}
	case auth.TokenKindBootstrap:
		if replacing {
			return 0, nil, rpc.ErrSessionIncarnationMismatch
		}
		if !legacy {
			if err := s.App.Hub.ValidateBootstrap(clientID, claims.BootstrapID); err != nil {
				return 0, nil, err
			}
		}
	default:
		return 0, nil, fmt.Errorf("未知的客户端 token kind")
	}

	if previous != nil {
		if previous.cancel != nil {
			previous.cancel()
		}
		if previous.conn != nil {
			_ = previous.conn.CloseNow()
		}
		if previous.done != nil {
			<-previous.done
		}
	}

	session, err := register(replacing)
	if err != nil {
		if previous != nil {
			delete(m.items, clientID)
			_ = s.App.Hub.UnregisterIncarnationE(clientID, previous.incarnation)
		}
		return 0, nil, err
	}
	m.nextID++
	connID := m.nextID
	m.items[clientID] = &clientWSBinding{connID: connID, generation: session.Generation, incarnation: session.SessionIncarnation, cancel: cancel, conn: conn, done: done}
	if legacy {
		m.legacy[legacyKey] = struct{}{}
	}
	return connID, session, nil
}

func clientWSTokenKind(claims *auth.Claims) (kind string, legacy bool, err error) {
	switch claims.TokenKind {
	case "":
		if claims.BootstrapID != "" || claims.SessionIncarnation != "" {
			return "", false, rpc.ErrSessionIncarnationMismatch
		}
		return auth.TokenKindBootstrap, true, nil
	case auth.TokenKindBootstrap:
		if claims.BootstrapID == "" || claims.SessionIncarnation != "" {
			return "", false, rpc.ErrBootstrapSessionMismatch
		}
		return auth.TokenKindBootstrap, false, nil
	case auth.TokenKindConnection:
		if claims.BootstrapID != "" {
			return "", false, rpc.ErrSessionIncarnationMismatch
		}
		if claims.SessionIncarnation == "" {
			return "", false, rpc.ErrSessionIncarnationRequired
		}
		return auth.TokenKindConnection, false, nil
	default:
		return "", false, fmt.Errorf("未知的客户端 token kind")
	}
}

func legacyClientTokenKey(claims *auth.Claims) string {
	if claims.ID == "" {
		return claims.ClientID
	}
	return claims.ClientID + "\x00" + claims.ID
}

func (s *Server) ShutdownClientConnections(ctx context.Context) error {
	if s.wsClients == nil {
		return nil
	}
	m := s.wsClients
	m.admitMu.Lock()
	defer m.admitMu.Unlock()
	m.mu.Lock()
	m.closed = true
	bindings := make([]*clientWSBinding, 0, len(m.items))
	for _, binding := range m.items {
		bindings = append(bindings, binding)
	}
	m.mu.Unlock()

	for _, binding := range bindings {
		if binding.cancel != nil {
			binding.cancel()
		}
		if binding.conn != nil {
			_ = binding.conn.CloseNow()
		}
	}
	for _, binding := range bindings {
		if binding.done == nil {
			continue
		}
		select {
		case <-binding.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (m *clientWSSessions) current(clientID string) (clientWSBinding, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	binding, ok := m.items[clientID]
	if !ok {
		return clientWSBinding{}, false
	}
	return *binding, true
}

func (m *clientWSSessions) clearIfCurrent(clientID string, connID uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.items[clientID]
	if !ok || current.connID != connID {
		return false
	}
	delete(m.items, clientID)
	return true
}

func (s *Server) handleClientWS(w http.ResponseWriter, r *http.Request) {
	claims, err := s.verifyClientWSClaims(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if err := s.App.EnsureGroupActive(r.Context(), claims.Group); err != nil {
		if status, ok := groupErrorHTTPStatus(err); ok {
			writeError(w, status, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		// Accept 失败时已自行写过响应，这里无需再写。
		return
	}
	conn.SetReadLimit(int64(wsMaxMessageBytes))

	ip := s.App.RemoteIP(r)
	ctx, cancel := context.WithCancel(context.Background())
	handlerDone := make(chan struct{})
	connID, session, err := s.registerClientWSBindingWithDone(claims, conn, cancel, handlerDone, func(replacing bool) (*rpc.ClientSession, error) {
		if claims.TokenKind == auth.TokenKindBootstrap && claims.BootstrapID != "" {
			return s.App.Hub.BindBootstrapConnectionCapabilitiesE(claims.ClientID, claims.Group, claims.UserID, "websocket", claims.MaxInFlight, claims.ActionsKnown, claims.Actions, claims.BootstrapID, replacing)
		}
		return s.App.Hub.BindConnectionCapabilitiesE(claims.ClientID, claims.Group, claims.UserID, "websocket", claims.MaxInFlight, claims.ActionsKnown, claims.Actions, replacing)
	})
	if err != nil {
		cancel()
		_ = conn.Close(websocket.StatusPolicyViolation, "session admission failed")
		return
	}
	defer func() {
		cancel()
		_ = conn.CloseNow()
		close(handlerDone)
		if current, ok := s.wsClients.current(claims.ClientID); ok && current.connID == connID {
			if unregisterErr := s.App.Hub.UnregisterIncarnationE(claims.ClientID, session.SessionIncarnation); unregisterErr != nil && !errors.Is(unregisterErr, rpc.ErrClientSessionGone) && !errors.Is(unregisterErr, rpc.ErrSessionIncarnationMismatch) {
				fmt.Printf("client ws cleanup failed: client=%s err=%v\n", claims.ClientID, unregisterErr)
			}
			s.wsClients.clearIfCurrent(claims.ClientID, connID)
		}
	}()
	s.App.TouchClientPresenceIncarnation(context.Background(), claims.ClientID, claims.Group, claims.UserID, claims.MaxInFlight, "websocket", ip, session.SessionIncarnation)
	connectionToken, err := s.App.IssueClientConnectionToken(claims, session.SessionIncarnation, session.Generation)
	if err != nil {
		cancel()
		_ = conn.Close(websocket.StatusInternalError, "connection token failed")
		return
	}
	sessionClaims := *claims
	sessionClaims.TokenKind = auth.TokenKindConnection
	sessionClaims.BootstrapID = ""
	sessionClaims.SessionIncarnation = session.SessionIncarnation
	sessionClaims.SessionGeneration = session.Generation

	if err := writeWSJSON(ctx, conn, wsEnvelope{
		Type:               "welcome",
		ClientID:           claims.ClientID,
		Group:              claims.Group,
		ServerID:           s.App.Config.ServerID,
		Time:               time.Now().Format(time.RFC3339),
		MaxInFlight:        session.MaxInFlight,
		Token:              connectionToken,
		SessionGeneration:  session.Generation,
		SessionIncarnation: session.SessionIncarnation,
	}); err != nil {
		return
	}

	// 对客户端的回包（ack/heartbeatAck，均为非关键消息，客户端本就忽略）走独立的异步
	// 发送协程，绝不让读循环阻塞在写上——读循环一旦被写阻塞，就读不到客户端的心跳/pong，
	// 等于 RPC 数据流干扰了保活。这正是要隔离的：心跳归心跳，RPC 不能拖累保活。
	outCh := make(chan wsEnvelope, 256)
	senderLoop := func(loopCtx context.Context) error {
		for {
			select {
			case <-loopCtx.Done():
				return nil
			case msg := <-outCh:
				if writeErr := writeWSJSONTimeout(loopCtx, conn, msg, 5*time.Second); writeErr != nil {
					return writeErr
				}
			}
		}
	}

	// 探针：每 30s 下发 {type:"probe"}，客户端若支持会回 {type:"probeAck"}。
	// 旧客户端会静默忽略（它们只认 type:"job"），不影响已部署设备。
	probePending := make(chan time.Time, 1)
	sendProbe := func() {
		// 发新探针前检查上次是否超时（probePending 还有未消费的值 = 上次没收到 ack）
		select {
		case oldSent := <-probePending:
			latencyMs := time.Since(oldSent).Milliseconds()
			s.App.Hub.RecordProbeIncarnation(claims.ClientID, session.SessionIncarnation, false, latencyMs)
			groupOnline := s.App.Hub.GroupOnlineCount(claims.Group)
			s.App.ProbeHistory.Record(claims.Group, groupOnline, false, latencyMs)
		default:
		}
		// 只在 outCh 入队成功时才写 probePending（被丢弃的探针不算已发出）
		msg := wsEnvelope{Type: "probe", Time: time.Now().Format(time.RFC3339Nano)}
		select {
		case outCh <- msg:
			select {
			case probePending <- time.Now():
			default:
			}
		default:
		}
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		sendProbe()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendProbe()
			}
		}
	}()

	errs := coordinateClientWSLoops(ctx, func() { _ = conn.CloseNow() },
		func(loopCtx context.Context) error {
			return s.clientWSReaderLoop(loopCtx, &sessionClaims, session.SessionIncarnation, conn, ip, outCh, probePending)
		},
		func(loopCtx context.Context) error { return s.clientWSWriterLoop(loopCtx, session, conn) },
		func(loopCtx context.Context) error {
			return s.clientWSPingLoop(loopCtx, conn, claims.ClientID, session.SessionIncarnation, cancel)
		},
		senderLoop,
	)
	readerErr, writerErr, pingErr := errs[0], errs[1], errs[2]
	if readerErr != nil && !isExpectedWSError(readerErr) {
		fmt.Printf("client ws reader closed: client=%s err=%v\n", claims.ClientID, readerErr)
	}
	if writerErr != nil && !isExpectedWSError(writerErr) {
		fmt.Printf("client ws writer closed: client=%s err=%v\n", claims.ClientID, writerErr)
	}
	if pingErr != nil && !isExpectedWSError(pingErr) {
		fmt.Printf("client ws ping closed: client=%s err=%v\n", claims.ClientID, pingErr)
	}
}

func coordinateClientWSLoops(parent context.Context, closeConn func(), loops ...func(context.Context) error) []error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	type loopResult struct {
		index int
		err   error
	}
	results := make(chan loopResult, len(loops))
	for index, loop := range loops {
		go func(index int, loop func(context.Context) error) {
			results <- loopResult{index: index, err: loop(ctx)}
		}(index, loop)
	}
	errs := make([]error, len(loops))
	if len(loops) == 0 {
		return errs
	}
	first := <-results
	errs[first.index] = first.err
	cancel()
	if closeConn != nil {
		closeConn()
	}
	for range len(loops) - 1 {
		result := <-results
		errs[result.index] = result.err
	}
	return errs
}

// writeWSJSON 序列化并写一条文本帧，带 15s 写超时。
func writeWSJSON(ctx context.Context, conn *websocket.Conn, payload wsEnvelope) error {
	return writeWSJSONTimeout(ctx, conn, payload, 15*time.Second)
}

// writeWSJSONTimeout 同 writeWSJSON，但可指定写超时（异步回包用更短超时，避免长时间占用写锁影响 ping）。
func writeWSJSONTimeout(ctx context.Context, conn *websocket.Conn, payload wsEnvelope, timeout time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, data)
}

// enqueueClientReply 把对客户端的回包投给异步发送协程；队列满则丢弃。
// 这些回包（resultAck/heartbeatAck）客户端本就忽略，且结果早已通过 SubmitClientResult 交付给调用方，
// 丢弃不影响正确性。关键是：绝不阻塞读循环——读被写拖住 = 读不到心跳/pong = 保活被 RPC 干扰。
func enqueueClientReply(outCh chan<- wsEnvelope, msg wsEnvelope) {
	select {
	case outCh <- msg:
	default:
	}
}

func (s *Server) clientWSWriterLoop(ctx context.Context, session *rpc.ClientSession, conn *websocket.Conn) error {
	return clientWSDispatchLoop(ctx, s.App.Hub, session, func(job *rpc.Job) error {
		return writeWSJSON(ctx, conn, wsEnvelope{Type: "job", Job: job})
	})
}

func clientWSDispatchLoop(ctx context.Context, hub *rpc.Hub, session *rpc.ClientSession, writeJob func(*rpc.Job) error) error {
	for {
		if err := hub.ValidateIncarnation(session.ClientID, session.SessionIncarnation); err != nil {
			return err
		}
		delivery, err := session.Pending.Reserve(ctx, hub.LeaseDuration())
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			requeueErr := session.Pending.RequeueDelivery(delivery, "dispatch_canceled")
			if requeueErr != nil {
				return errors.Join(ctx.Err(), requeueErr)
			}
			return nil
		}

		execution, err := hub.AcquireExecutionLeaseIncarnation(ctx, session.ClientID, delivery, session.SessionIncarnation)
		if err != nil {
			switch {
			case errors.Is(err, rpc.ErrRequestTerminal):
				if ackErr := session.Pending.Ack(delivery); ackErr != nil && !errors.Is(ackErr, rpc.ErrLeaseExpired) {
					return errors.Join(err, ackErr)
				}
				continue
			case errors.Is(err, rpc.ErrLeaseExpired):
				if requeueErr := session.Pending.RequeueDelivery(delivery, "dispatch_lease_expired"); requeueErr != nil && !errors.Is(requeueErr, rpc.ErrLeaseExpired) && !errors.Is(requeueErr, rpc.ErrLeaseNotFound) {
					return errors.Join(err, requeueErr)
				}
				continue
			case errors.Is(err, rpc.ErrActionNotSupported), errors.Is(err, rpc.ErrJobExpired):
				if ackErr := session.Pending.Ack(delivery); ackErr != nil && !errors.Is(ackErr, rpc.ErrLeaseExpired) && !errors.Is(ackErr, rpc.ErrLeaseNotFound) {
					return errors.Join(err, ackErr)
				}
				continue
			default:
				requeueErr := session.Pending.RequeueDelivery(delivery, "dispatch_acquire_failed")
				if requeueErr != nil {
					return errors.Join(err, requeueErr)
				}
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}

		if err := writeJob(delivery.Job); err != nil {
			requeueErr := hub.RequeueExecutionLease(execution, "dispatch_write_failed")
			if requeueErr != nil {
				return errors.Join(err, requeueErr)
			}
			return err
		}
		if !hub.MarkExecutionLeaseSent(execution) {
			return rpc.ErrSessionIncarnationMismatch
		}
	}
}

func (s *Server) clientHeartbeatTimeout() time.Duration {
	seconds := s.App.Config.DeviceOfflineSeconds
	if seconds <= 0 {
		if s.App.Config.DeviceOfflineMinutes > 0 {
			seconds = s.App.Config.DeviceOfflineMinutes * 60
		} else {
			seconds = 20
		}
	}
	return time.Duration(seconds) * time.Second
}

func (s *Server) clientWSPingInterval() time.Duration {
	seconds := s.App.Config.HeartbeatIntervalSeconds
	if seconds <= 0 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}

// clientWSPingLoop 是连接唯一的保活判定：每 interval 发一次 WS ping，
// 在 pongTimeout 内收不到 pong 即认定链路已死，调用 onDead 主动拆连接。
// 与 RPC 应用消息完全解耦——再大的调用流量也不会让它误判离线。
func (s *Server) clientWSPingLoop(ctx context.Context, conn *websocket.Conn, clientID, incarnation string, onDead func()) error {
	ticker := time.NewTicker(s.clientWSPingInterval())
	defer ticker.Stop()

	pongTimeout := s.clientHeartbeatTimeout()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, pongTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				onDead() // ping/pong 失败 = 链路真死 → 主动拆连接（之前这里的错误没人理，连接不会被拆）
				return err
			}
			// ping 成功 = 链路活着 → 刷新在线时间。让"在线"判定脱离 RPC 应用消息：
			// 即便大流量把心跳/结果挤住，WS 保活本身就维持设备在线，不再误判离线。
			s.App.Hub.TouchIncarnation(clientID, incarnation)
		}
	}
}

func (s *Server) clientWSReaderLoop(ctx context.Context, claims *auth.Claims, sessionIncarnation string, conn *websocket.Conn, ip string, outCh chan<- wsEnvelope, probePending <-chan time.Time) error {
	for {
		// 不再用"应用消息读超时"判活：判活交给 ping/pong（clientWSPingLoop）。
		// 这样再大的 RPC 数据流量/拥塞都不会误杀健康连接——保活与调用彻底解耦。
		msgType, payload, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if msgType != websocket.MessageText {
			continue
		}
		s.App.TouchClientPresenceIncarnation(context.Background(), claims.ClientID, claims.Group, claims.UserID, claims.MaxInFlight, "", ip, sessionIncarnation)

		var envelope wsEnvelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			enqueueClientReply(outCh, wsEnvelope{Type: "error", Error: "invalid json payload"})
			continue
		}
		switch envelope.Type {
		case "probeAck":
			select {
			case sentAt := <-probePending:
				latencyMs := time.Since(sentAt).Milliseconds()
				s.App.Hub.RecordProbeIncarnation(claims.ClientID, sessionIncarnation, true, latencyMs)
				groupSessions := s.App.Hub.GroupOnlineCount(claims.Group)
				s.App.ProbeHistory.Record(claims.Group, groupSessions, true, latencyMs)
			default:
			}
		case "heartbeat":
			enqueueClientReply(outCh, wsEnvelope{Type: "heartbeatAck", Time: time.Now().Format(time.RFC3339)})
		case "result":
			if envelope.Result == nil {
				enqueueClientReply(outCh, wsEnvelope{Type: "resultAck", OK: false, Error: "result payload required"})
				continue
			}
			if strings.TrimSpace(envelope.Result.Status) == "" {
				envelope.Result.Status = "success"
			}
			envelope.Result.SessionIncarnation = sessionIncarnation
			if err := normalizeClientJobResult(envelope.Result); err != nil {
				enqueueClientReply(outCh, wsEnvelope{Type: "resultAck", RequestID: envelope.Result.RequestID, OK: false, Error: err.Error(), State: "rejected"})
				continue
			}
			submitErr := s.App.SubmitClientResult(context.Background(), claims, *envelope.Result)
			ack := wsEnvelope{Type: "resultAck", RequestID: envelope.Result.RequestID, OK: submitErr == nil}
			if submitErr != nil {
				ack.Error = submitErr.Error()
				if errors.Is(submitErr, rpc.ErrResultClientMismatch) {
					ack.State = "rejected"
				} else {
					ack.State = "error"
				}
			} else {
				ack.State = "accepted"
			}
			enqueueClientReply(outCh, ack)
		default:
			enqueueClientReply(outCh, wsEnvelope{Type: "error", Error: "unsupported message type"})
		}
	}
}

func (s *Server) verifyClientWSClaims(r *http.Request) (*auth.Claims, error) {
	var (
		claims *auth.Claims
		err    error
	)
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		claims, err = s.App.Tokens.Parse(token)
	} else {
		claims, err = s.App.VerifyTokenFromRequest(r)
	}
	if err != nil {
		return nil, err
	}
	if claims.Role != "client" {
		return nil, fmt.Errorf("权限不足")
	}
	tokenKind, legacy, err := clientWSTokenKind(claims)
	if err != nil {
		return nil, err
	}
	switch tokenKind {
	case auth.TokenKindBootstrap:
		if _, active := s.wsClients.current(claims.ClientID); active {
			return nil, rpc.ErrBootstrapSessionMismatch
		}
		if !legacy {
			if err := s.App.Hub.ValidateBootstrap(claims.ClientID, claims.BootstrapID); err != nil {
				return nil, err
			}
		}
	case auth.TokenKindConnection:
		if err := s.App.Hub.ValidateIncarnation(claims.ClientID, claims.SessionIncarnation); err != nil {
			return nil, err
		}
		session, ok := s.App.Hub.Session(claims.ClientID)
		if !ok || session.Group != claims.Group {
			return nil, rpc.ErrSessionIncarnationMismatch
		}
	default:
		return nil, fmt.Errorf("未知的客户端 token kind")
	}
	return claims, nil
}

func (s *Server) clientWSURL(r *http.Request, token string) string {
	scheme := "ws"
	forwardedProto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if forwardedProto == "https" || r.TLS != nil {
		scheme = "wss"
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s://%s/api/client/ws?token=%s", scheme, host, url.QueryEscape(token))
}

func isExpectedWSError(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	switch websocket.CloseStatus(err) {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway, websocket.StatusNoStatusRcvd, websocket.StatusAbnormalClosure:
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "use of closed network connection") ||
		strings.Contains(message, "eof") ||
		strings.Contains(message, "broken pipe")
}
