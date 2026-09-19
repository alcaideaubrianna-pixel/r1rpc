package web

import (
	"net/http"

	"r1rpc/internal/auth"
	"r1rpc/internal/service/storagesetting"
)

func (s *Server) registerStorageSettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/storage/settings", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		setting, err := s.App.StorageSettings.Get(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, setting)
	}))
	mux.HandleFunc("PUT /api/v1/storage/settings", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		var input storagesetting.SaveInput
		if err := decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		setting, err := s.App.StorageSettings.Save(r.Context(), input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, setting)
	}))
}
