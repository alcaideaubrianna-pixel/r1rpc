package observability

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gctx"
)

const traceIDHeader = "Trace-Id"

var openTelemetryTraceID = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)

// HTTPMiddleware 为现有 net/http 服务补充 GoFrame Trace Context 和结构化访问日志。
// WebSocket、静态资源和关闭流程仍由原服务管理，避免基础接入阶段改变协议行为。
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		ctx := gctx.WithCtx(r.Context())
		if requestID := strings.TrimSpace(r.Header.Get("X-Request-ID")); openTelemetryTraceID.MatchString(requestID) {
			if tracedCtx, err := gtrace.WithTraceID(ctx, strings.ToLower(requestID)); err == nil {
				ctx = tracedCtx
			}
		}

		traceID := gctx.CtxId(ctx)
		w.Header().Set(traceIDHeader, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))

		g.Log().Info(ctx, g.Map{
			"event":       "http_request",
			"method":      r.Method,
			"path":        r.URL.Path,
			"duration_ms": time.Since(startedAt).Milliseconds(),
		})
	})
}
