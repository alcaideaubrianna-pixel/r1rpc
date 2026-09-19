package web

import (
	"net/http"
	"strconv"
	"strings"

	imagesearchv1 "r1rpc/api/image_search/v1"
	"r1rpc/internal/auth"
	imagesearchcontroller "r1rpc/internal/controller/image_search"
	"r1rpc/internal/requestctx"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *Server) registerImageSearchRoutes(mux *http.ServeMux) {
	controller := imagesearchcontroller.NewV1WithService(s.App.ImageSearch)
	mux.HandleFunc("POST /api/v1/image-search/requests", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		var req imagesearchv1.CreateReq
		if err := decodeLimitedJSON(w, r, &req); err != nil {
			writeImageSearchError(w, err)
			return
		}
		ctx := requestctx.WithRequester(r.Context(), requestctx.Requester{
			UserID: claims.UserID, Subject: claims.Username,
		})
		res, err := controller.Create(ctx, &req)
		if err != nil {
			writeImageSearchError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, res)
	}))
	mux.HandleFunc("GET /api/v1/image-search/requests/{id}", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		res, err := controller.GetOne(r.Context(), &imagesearchv1.GetOneReq{ID: r.PathValue("id")})
		if err != nil {
			writeImageSearchError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	}))
	mux.HandleFunc("GET /api/v1/image-search/requests/{id}/candidates", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		matched, err := optionalBool(r.URL.Query().Get("matched"))
		if err != nil {
			writeImageSearchError(w, gerror.WrapCode(gcode.CodeInvalidParameter, err, "matched 参数无效"))
			return
		}
		res, err := controller.GetCandidates(r.Context(), &imagesearchv1.GetCandidatesReq{
			ID: r.PathValue("id"), Page: page, PageSize: pageSize, Matched: matched,
		})
		if err != nil {
			writeImageSearchError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	}))
	mux.HandleFunc("GET /api/v1/image-search/requests/{id}/responses", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		res, err := controller.GetResponses(r.Context(), &imagesearchv1.GetResponsesReq{ID: r.PathValue("id")})
		if err != nil {
			writeImageSearchError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	}))
	mux.HandleFunc("POST /api/v1/image-search/requests/{id}/retry-analysis", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		res, err := controller.RetryAnalysis(r.Context(), &imagesearchv1.RetryAnalysisReq{ID: r.PathValue("id")})
		if err != nil {
			writeImageSearchError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, res)
	}))
	mux.HandleFunc("GET /api/v1/image-search/requests", s.requireRole("admin", func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		res, err := controller.GetList(r.Context(), &imagesearchv1.GetListReq{
			Page: page, PageSize: pageSize, Status: r.URL.Query().Get("status"),
		})
		if err != nil {
			writeImageSearchError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	}))
}

func optionalBool(raw string) (*bool, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	return &value, err
}

func writeImageSearchError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch gerror.Code(err) {
	case gcode.CodeInvalidParameter:
		status = http.StatusBadRequest
	case gcode.CodeNotFound:
		status = http.StatusNotFound
	}
	writeError(w, status, err)
}
