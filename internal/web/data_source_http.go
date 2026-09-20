package web

import (
	"net/http"
	"strconv"

	"r1rpc/internal/auth"
	"r1rpc/internal/service/datasource"

	"github.com/gogf/gf/v2/frame/g"
)

func (s *Server) registerDataSourceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/data-sources", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		items, err := s.App.DataSources.List(r.Context())
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))
	mux.HandleFunc("POST /api/v1/data-sources", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		var input datasource.SaveInput
		if err := decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, 400, err)
			return
		}
		item, err := s.App.DataSources.Save(r.Context(), input)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 201, item)
	}))
	mux.HandleFunc("PATCH /api/v1/data-sources/{id}", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		var input datasource.SaveInput
		if err := decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, 400, err)
			return
		}
		item, err := s.App.DataSources.Update(r.Context(), r.PathValue("id"), input)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 200, item)
	}))
	mux.HandleFunc("GET /api/v1/data-sources/{id}/channels", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		result, err := s.App.DataSources.Channels(r.Context(), r.PathValue("id"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			g.Log().Errorf(r.Context(), "读取数据源频道失败 sourceId=%s err=%v", r.PathValue("id"), err)
			writeError(w, 502, err)
			return
		}
		writeJSON(w, 200, result)
	}))
	mux.HandleFunc("GET /api/v1/channel-scan-tasks", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		result, err := s.App.DataSources.ListScanTasks(r.Context(), page, pageSize)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, result)
	}))
	mux.HandleFunc("POST /api/v1/channel-scan-tasks", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		var input datasource.ScanTaskInput
		if err := decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, 400, err)
			return
		}
		item, err := s.App.DataSources.CreateScanTask(r.Context(), input)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 202, item)
	}))
}
