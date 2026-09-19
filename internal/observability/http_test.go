package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gogf/gf/v2/os/gctx"
)

func TestHTTPMiddlewareInjectsTraceID(t *testing.T) {
	var observed string
	handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = gctx.CtxId(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if observed == "" {
		t.Fatal("请求上下文中缺少 Trace ID")
	}
	if got := recorder.Header().Get(traceIDHeader); got != observed {
		t.Fatalf("响应 Trace ID=%q，上下文 Trace ID=%q", got, observed)
	}
}

func TestHTTPMiddlewareAcceptsOpenTelemetryRequestID(t *testing.T) {
	const requestID = "0123456789abcdef0123456789abcdef"
	var observed string
	handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = gctx.CtxId(r.Context())
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", requestID)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if observed != requestID {
		t.Fatalf("Trace ID=%q，期望=%q", observed, requestID)
	}
}
