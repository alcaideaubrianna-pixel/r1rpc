package image_search

import (
	"context"

	"r1rpc/api/image_search/v1"
)

func (c *ControllerV1) GetList(ctx context.Context, req *v1.GetListReq) (res *v1.GetListRes, err error) {
	result, err := c.service.GetList(ctx, req.Page, req.PageSize, req.Status)
	if err != nil {
		return nil, err
	}
	items := make([]v1.RequestSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toRequestSummary(item))
	}
	return &v1.GetListRes{
		Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total,
	}, nil
}
