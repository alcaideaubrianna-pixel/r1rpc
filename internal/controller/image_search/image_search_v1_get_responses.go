package image_search

import (
	"context"
	"time"

	"r1rpc/api/image_search/v1"
)

func (c *ControllerV1) GetResponses(ctx context.Context, req *v1.GetResponsesReq) (res *v1.GetResponsesRes, err error) {
	result, err := c.service.GetResponses(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	items := make([]v1.ResponseSummary, 0, len(result))
	for _, item := range result {
		items = append(items, v1.ResponseSummary{
			ID: item.ID, JobID: item.JobID, RequestID: item.RequestID,
			ItemCount: item.ItemCount, ParseStatus: item.ParseStatus,
			ErrorMessage: item.ErrorMessage, NormalizedJSON: item.NormalizedJSON,
			RawJSON:   item.RawJSON,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	return &v1.GetResponsesRes{Items: items}, nil
}
