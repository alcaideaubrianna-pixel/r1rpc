package image_search

import (
	"context"

	"r1rpc/api/image_search/v1"
)

func (c *ControllerV1) RetryAnalysis(ctx context.Context, req *v1.RetryAnalysisReq) (res *v1.RetryAnalysisRes, err error) {
	if err := c.service.RetryAnalysis(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.RetryAnalysisRes{OK: true}, nil
}
