package web

import (
	"context"
	"encoding/json"
	"net/http"

	"r1rpc/internal/app"
	"r1rpc/internal/auth"
)

func (s *Server) registerXHSProfileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/xhs/users/{userId}", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		profile, err := s.App.XHSProfiles.Get(r.Context(), r.PathValue("userId"), r.URL.Query().Get("candidateId"),
			func(ctx context.Context, clientID string, payload json.RawMessage) (json.RawMessage, error) {
				result, _, _, err := s.App.InvokeRPC(ctx, nil, app.DeviceGroup, "network.request", app.InvokeRequest{ClientID: clientID, Payload: payload, Timeout: 20})
				return result.Payload, err
			})
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, profile)
	}))
}
