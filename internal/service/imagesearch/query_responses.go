package imagesearch

import (
	"context"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *Service) GetResponses(ctx context.Context, requestID string) ([]output.ImageSearchResponse, error) {
	if _, err := s.GetOne(ctx, requestID); err != nil {
		return nil, err
	}
	groupColumns := dao.ImageSearchGroups.Columns()
	var groups []entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Fields(groupColumns.Id).
		Where(groupColumns.RequestId, requestID).Scan(&groups); err != nil {
		return nil, gerror.Wrap(err, "查询图片检索组失败")
	}
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.Id)
	}
	if len(groupIDs) == 0 {
		return []output.ImageSearchResponse{}, nil
	}
	itemColumns := dao.ImageSearchItems.Columns()
	var items []entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Fields(itemColumns.JobId).
		WhereIn(itemColumns.GroupId, groupIDs).Scan(&items); err != nil {
		return nil, gerror.Wrap(err, "查询图片检索项失败")
	}
	jobIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.JobId != "" {
			jobIDs = append(jobIDs, item.JobId)
		}
	}
	if len(jobIDs) == 0 {
		return []output.ImageSearchResponse{}, nil
	}
	columns := dao.ImageSearchResponses.Columns()
	var records []entity.ImageSearchResponses
	if err := dao.ImageSearchResponses.Ctx(ctx).WhereIn(columns.JobId, jobIDs).
		OrderAsc(columns.CreatedAt).Scan(&records); err != nil {
		return nil, gerror.Wrap(err, "查询设备图片搜索响应失败")
	}
	result := make([]output.ImageSearchResponse, 0, len(records))
	for _, record := range records {
		result = append(result, output.ImageSearchResponse{
			ID: record.Id, JobID: record.JobId, RequestID: record.RequestId,
			ItemCount: record.ItemCount, ParseStatus: record.ParseStatus,
			ErrorMessage: record.ErrorMessage, NormalizedJSON: record.NormalizedJson,
			RawJSON:   record.RawJson,
			CreatedAt: record.CreatedAt,
		})
	}
	return result, nil
}
