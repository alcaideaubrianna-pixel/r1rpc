package image_search

import (
	"context"

	"r1rpc/api/image_search/v1"
)

func (c *ControllerV1) GetOne(ctx context.Context, req *v1.GetOneReq) (res *v1.GetOneRes, err error) {
	result, err := c.service.GetOne(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	groups := make([]v1.GroupSummary, 0, len(result.Groups))
	for _, group := range result.Groups {
		groups = append(groups, v1.GroupSummary{
			ID: group.ID, ExternalID: group.ExternalID, SubjectUserID: group.SubjectUserID,
			Status: group.Status, ItemCount: group.ItemCount, QueuedCount: group.QueuedCount,
			RunningCount: group.RunningCount, CompletedCount: group.CompletedCount,
			FailedCount: group.FailedCount,
			BestScore:   group.BestScore, BestMatchID: group.BestMatchID,
		})
	}
	items := make([]v1.ItemSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, v1.ItemSummary{
			ID: item.ID, GroupID: item.GroupID, FileID: item.FileID, FileURL: item.FileURL, Ordinal: item.Ordinal,
			Status: item.Status, ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
		})
	}
	return &v1.GetOneRes{Request: toRequestSummary(*result), Groups: groups, Items: items}, nil
}
