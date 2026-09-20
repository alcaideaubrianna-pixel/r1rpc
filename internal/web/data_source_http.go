package web

import (
	"net/http"

	"r1rpc/internal/auth"
	"r1rpc/internal/service/datasource"
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
	mux.HandleFunc("GET /api/v1/data-sources/{id}/channels", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		items, err := s.App.DataSources.Channels(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, 502, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))
}
