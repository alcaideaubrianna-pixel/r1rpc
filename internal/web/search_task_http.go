package web

import (
	"net/http"
	"strconv"

	"r1rpc/internal/auth"
	"r1rpc/internal/service/searchtask"
)

func (s *Server) registerSearchTaskRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/search-configs", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		items, err := s.App.SearchTasks.ListConfigs(r.Context())
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))
	mux.HandleFunc("POST /api/v1/search-configs", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		var input searchtask.ConfigInput
		if err := decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, 400, err)
			return
		}
		item, err := s.App.SearchTasks.SaveConfig(r.Context(), input)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 201, item)
	}))
	mux.HandleFunc("PUT /api/v1/search-configs/{id}", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		var input searchtask.ConfigInput
		if err := decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, 400, err)
			return
		}
		input.ID = r.PathValue("id")
		item, err := s.App.SearchTasks.SaveConfig(r.Context(), input)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 200, item)
	}))
	mux.HandleFunc("GET /api/v1/search-tasks", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		result, err := s.App.SearchTasks.ListTasks(r.Context(), page, pageSize)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, result)
	}))
	mux.HandleFunc("GET /api/v1/search-task-items", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		result, err := s.App.SearchTasks.ListIndependentItems(r.Context(), page, pageSize, r.URL.Query().Get("q"))
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, result)
	}))
	mux.HandleFunc("GET /api/v1/search-tasks/{id}/items", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		result, err := s.App.SearchTasks.ListItems(r.Context(), r.PathValue("id"), page, pageSize)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, result)
	}))
}
