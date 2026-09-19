// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package image_search

import (
	"context"

	"r1rpc/api/image_search/v1"
)

type IImageSearchV1 interface {
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	GetCandidates(ctx context.Context, req *v1.GetCandidatesReq) (res *v1.GetCandidatesRes, err error)
	GetList(ctx context.Context, req *v1.GetListReq) (res *v1.GetListRes, err error)
	GetOne(ctx context.Context, req *v1.GetOneReq) (res *v1.GetOneRes, err error)
	GetResponses(ctx context.Context, req *v1.GetResponsesReq) (res *v1.GetResponsesRes, err error)
	RetryAnalysis(ctx context.Context, req *v1.RetryAnalysisReq) (res *v1.RetryAnalysisRes, err error)
}
