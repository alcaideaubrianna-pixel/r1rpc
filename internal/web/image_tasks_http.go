package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"r1rpc/internal/app"
	"r1rpc/internal/auth"
	"r1rpc/internal/model/input"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

type createImageJobRequest struct {
	FileID       string `json:"fileId"`
	ExternalID   string `json:"externalId"`
	Priority     int    `json:"priority"`
	ForceRefresh bool   `json:"forceRefresh"`
}

type createImageBatchRequest struct {
	FileIDs      []string `json:"fileIds"`
	ExternalID   string   `json:"externalId"`
	Priority     int      `json:"priority"`
	ForceRefresh bool     `json:"forceRefresh"`
}

func (s *Server) handleCreateImageJob(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	var request createImageJobRequest
	if err := decodeLimitedJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.App.ImageTasks.CreateJob(r.Context(), input.CreateImageJob{
		Source:       "admin",
		ExternalID:   request.ExternalID,
		FileID:       request.FileID,
		Priority:     request.Priority,
		ForceRefresh: request.ForceRefresh,
		RequestedBy:  map[string]any{"userId": claims.UserID, "username": claims.Username},
	})
	if err != nil {
		writeImageTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleCreateImageBatch(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	var request createImageBatchRequest
	if err := decodeLimitedJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.App.ImageTasks.CreateBatch(r.Context(), input.CreateImageBatch{
		Source:       "admin",
		ExternalID:   request.ExternalID,
		FileIDs:      request.FileIDs,
		Priority:     request.Priority,
		ForceRefresh: request.ForceRefresh,
		RequestedBy:  map[string]any{"userId": claims.UserID, "username": claims.Username},
	})
	if err != nil {
		writeImageTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleListImageBatches(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	result, err := s.App.ImageTasks.ListBatches(r.Context(), page, pageSize)
	if err != nil {
		writeImageTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRecognizeAction(
	w http.ResponseWriter,
	r *http.Request,
	claims *auth.Claims,
	request app.InvokeRequest,
) {
	fileID, forceRefresh, priority, err := fileIDFromRecognizePayload(request.Payload)
	if err != nil {
		writeImageTaskError(w, err)
		return
	}
	requestedBy := map[string]any{"role": "caller"}
	if claims != nil {
		requestedBy = map[string]any{"userId": claims.UserID, "username": claims.Username, "role": claims.Role}
	}
	result, err := s.App.ImageTasks.CreateJob(r.Context(), input.CreateImageJob{
		Source:       "api",
		ExternalID:   request.RequestID,
		FileID:       fileID,
		Priority:     priority,
		ForceRefresh: forceRefresh,
		RequestedBy:  requestedBy,
	})
	if err != nil {
		writeImageTaskError(w, err)
		return
	}
	writeEnvelope(w, http.StatusAccepted, true, "queued", map[string]any{
		"requestId": request.RequestID,
		"job":       result,
	})
}

func decodeLimitedJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return gerror.WrapCode(gcode.CodeInvalidParameter, err, "请求 JSON 无效")
	}
	return nil
}

func writeImageTaskError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch gerror.Code(err) {
	case gcode.CodeInvalidParameter:
		status = http.StatusBadRequest
	case gcode.CodeNotFound:
		status = http.StatusNotFound
	}
	writeError(w, status, err)
}

func fileIDFromRecognizePayload(payload json.RawMessage) (string, bool, int, error) {
	var request struct {
		Image struct {
			FileID string `json:"fileId"`
		} `json:"image"`
		ForceRefresh bool `json:"forceRefresh"`
		Priority     int  `json:"priority"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return "", false, 0, gerror.WrapCode(gcode.CodeInvalidParameter, err, "识图 payload 无效")
	}
	return strings.TrimSpace(request.Image.FileID), request.ForceRefresh, request.Priority, nil
}
