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
	mux.HandleFunc("DELETE /api/v1/data-sources/{id}", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		if err := s.App.DataSources.Delete(r.Context(), r.PathValue("id")); err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 200, map[string]any{"deleted": true})
	}))
	mux.HandleFunc("POST /api/v1/data-sources/{id}/channels/sync", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		count, err := s.App.DataSources.SyncChannels(r.Context(), r.PathValue("id"))
		if err != nil {
			g.Log().Errorf(r.Context(), "同步数据源频道失败 sourceId=%s err=%v", r.PathValue("id"), err)
			writeError(w, 502, err)
			return
		}
		writeJSON(w, 200, map[string]any{"syncedCount": count})
	}))
	mux.HandleFunc("GET /api/v1/data-sources/{id}/channels/{channelId}/notes/preview", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		channelID, err := strconv.ParseInt(r.PathValue("channelId"), 10, 64)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		result, err := s.App.DataSources.PreviewNotes(r.Context(), r.PathValue("id"), channelID, limit, r.URL.Query().Get("nextNo"))
		if err != nil {
			writeError(w, 502, err)
			return
		}
		writeJSON(w, 200, result)
	}))
	mux.HandleFunc("GET /api/v1/source-channels", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		result, err := s.App.DataSources.ListChannels(r.Context(), r.URL.Query().Get("dataSourceId"), r.URL.Query().Get("q"), page, pageSize)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, result)
	}))
	mux.HandleFunc("GET /api/v1/data-sources/{id}/channels/{channelId}/notes", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		channelID, err := strconv.ParseInt(r.PathValue("channelId"), 10, 64)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		result, err := s.App.DataSources.ListNotes(r.Context(), r.PathValue("id"), channelID, r.URL.Query().Get("q"), page, pageSize)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, result)
	}))
	mux.HandleFunc("POST /api/v1/data-sources/{id}/channels/{channelId}/notes/sync", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		channelID, err := strconv.ParseInt(r.PathValue("channelId"), 10, 64)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		item, err := s.App.DataSources.CreateScanTask(r.Context(), datasource.ScanTaskInput{DataSourceID: r.PathValue("id"), ChannelID: channelID, ChannelTitle: r.URL.Query().Get("title"), Mode: "sync", InitialLimit: 10000})
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 202, item)
	}))
	mux.HandleFunc("POST /api/v1/data-sources/{id}/channels/{channelId}/note-batches", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		channelID, err := strconv.ParseInt(r.PathValue("channelId"), 10, 64)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		var input datasource.ScanTaskInput
		if err = decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, 400, err)
			return
		}
		input.DataSourceID, input.ChannelID = r.PathValue("id"), channelID
		result, err := s.App.DataSources.CreateScanTask(r.Context(), input)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 202, result)
	}))
	mux.HandleFunc("PATCH /api/v1/source-channels/{id}/pin", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, _ *auth.Claims) {
		var input struct {
			Pinned bool `json:"pinned"`
		}
		if err := decodeLimitedJSON(w, r, &input); err != nil {
			writeError(w, 400, err)
			return
		}
		if err := s.App.DataSources.SetChannelPinned(r.Context(), r.PathValue("id"), input.Pinned); err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 200, map[string]any{"pinned": input.Pinned})
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
