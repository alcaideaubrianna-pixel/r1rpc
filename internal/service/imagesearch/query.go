package imagesearch

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *Service) findByExternalID(ctx context.Context, source, externalID string) (*output.ImageSearchRequest, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil
	}
	columns := dao.ImageSearchRequests.Columns()
	var record entity.ImageSearchRequests
	err := dao.ImageSearchRequests.Ctx(ctx).
		Where(columns.Source, strings.TrimSpace(source)).
		Where(columns.ExternalId, externalID).
		Scan(&record)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, gerror.Wrap(err, "查询幂等图片检索请求失败")
	}
	if record.Id == "" {
		return nil, nil
	}
	result := mapRequest(record)
	result.Existed = true
	return &result, nil
}

func (s *Service) GetOne(ctx context.Context, id string) (*output.ImageSearchRequest, error) {
	requestColumns := dao.ImageSearchRequests.Columns()
	var request entity.ImageSearchRequests
	if err := dao.ImageSearchRequests.Ctx(ctx).
		Where(requestColumns.Id, strings.TrimSpace(id)).Scan(&request); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.NewCode(gcode.CodeNotFound, "图片检索请求不存在")
		}
		return nil, gerror.Wrap(err, "查询图片检索请求失败")
	}
	if request.Id == "" {
		return nil, gerror.NewCode(gcode.CodeNotFound, "图片检索请求不存在")
	}

	groupColumns := dao.ImageSearchGroups.Columns()
	var groups []entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).
		Where(groupColumns.RequestId, request.Id).
		OrderAsc(groupColumns.CreatedAt).OrderAsc(groupColumns.Id).
		Scan(&groups); err != nil {
		return nil, gerror.Wrap(err, "查询图片检索组失败")
	}
	result := mapRequest(request)
	result.Groups = make([]output.ImageSearchGroup, 0, len(groups))
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		result.Groups = append(result.Groups, mapGroup(group))
		groupIDs = append(groupIDs, group.Id)
	}
	if len(groupIDs) == 0 {
		return &result, nil
	}
	itemColumns := dao.ImageSearchItems.Columns()
	var items []entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).WhereIn(itemColumns.GroupId, groupIDs).
		OrderAsc(itemColumns.GroupId).OrderAsc(itemColumns.Ordinal).Scan(&items); err != nil {
		return nil, gerror.Wrap(err, "查询图片检索项失败")
	}
	assetIDs := make([]string, 0, len(items))
	for _, item := range items {
		assetIDs = append(assetIDs, item.SourceAssetId)
	}
	assetFiles := make(map[string]string, len(assetIDs))
	if len(assetIDs) > 0 {
		assetColumns := dao.ImageAssets.Columns()
		var assets []entity.ImageAssets
		if err := dao.ImageAssets.Ctx(ctx).Fields(assetColumns.Id, assetColumns.FileId).
			WhereIn(assetColumns.Id, assetIDs).Scan(&assets); err != nil {
			return nil, gerror.Wrap(err, "查询图片资产失败")
		}
		for _, asset := range assets {
			assetFiles[asset.Id] = asset.FileId
		}
	}
	result.Items = make([]output.ImageSearchItem, 0, len(items))
	itemByFileID := make(map[string]output.ImageSearchItem, len(items))
	for _, item := range items {
		fileID := assetFiles[item.SourceAssetId]
		fileURL := ""
		if fileID != "" && s.files != nil {
			fileURL, _ = s.files.URL(ctx, fileID)
		}
		mapped := output.ImageSearchItem{
			ID: item.Id, GroupID: item.GroupId, FileID: fileID, FileURL: fileURL,
			Ordinal: item.Ordinal, Status: item.Status, JobID: item.JobId,
			ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
		}
		result.Items = append(result.Items, mapped)
		if fileID != "" {
			itemByFileID[fileID] = mapped
		}
	}
	if request.Source == "channel_scan_item" {
		if err := s.loadSourceImages(ctx, &result, request.ExternalId, itemByFileID); err != nil {
			return nil, err
		}
	}
	return &result, nil
}

func (s *Service) loadSourceImages(ctx context.Context, result *output.ImageSearchRequest, externalID string, itemByFileID map[string]output.ImageSearchItem) error {
	parts := strings.Split(externalID, ":")
	if len(parts) != 3 || parts[0] != "source-note" || parts[2] == "" {
		return nil
	}
	columns := dao.SourceNoteImages.Columns()
	var records []entity.SourceNoteImages
	if err := dao.SourceNoteImages.Ctx(ctx).Where(columns.NoteId, parts[2]).OrderAsc(columns.ImageIndex).Scan(&records); err != nil {
		return gerror.Wrap(err, "查询资料源图片失败")
	}
	defaultGroupID := ""
	if len(result.Groups) > 0 {
		defaultGroupID = result.Groups[0].ID
	}
	result.SourceImages = make([]output.SourceImage, 0, len(records))
	for _, record := range records {
		fileURL := ""
		if record.FileId != "" && s.files != nil {
			fileURL, _ = s.files.URL(ctx, record.FileId)
		}
		groupID, searchItemID := defaultGroupID, ""
		if item, ok := itemByFileID[record.FileId]; ok {
			groupID, searchItemID = item.GroupID, item.ID
		}
		result.SourceImages = append(result.SourceImages, output.SourceImage{
			ID: record.Id, GroupID: groupID, SearchItemID: searchItemID,
			FileID: record.FileId, FileURL: fileURL, ImageIndex: record.ImageIndex,
			DownloadStatus: record.DownloadStatus, PreprocessStatus: record.PreprocessStatus,
			FilterDecision: record.FilterDecision, FilterReason: record.FilterReason,
		})
	}
	return nil
}

func (s *Service) GetList(ctx context.Context, page, pageSize int, status string) (*output.ImageSearchRequestPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	columns := dao.ImageSearchRequests.Columns()
	model := dao.ImageSearchRequests.Ctx(ctx)
	if status = strings.TrimSpace(status); status != "" {
		model = model.Where(columns.Status, status)
	}
	total, err := model.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "统计图片检索请求失败")
	}
	var records []entity.ImageSearchRequests
	if err := model.OrderDesc(columns.CreatedAt).OrderDesc(columns.Id).
		Page(page, pageSize).Scan(&records); err != nil {
		return nil, gerror.Wrap(err, "查询图片检索请求列表失败")
	}
	result := &output.ImageSearchRequestPage{
		Items: make([]output.ImageSearchRequest, 0, len(records)),
		Page:  page, PageSize: pageSize, Total: total,
	}
	for _, record := range records {
		result.Items = append(result.Items, mapRequest(record))
	}
	return result, nil
}

func mapRequest(record entity.ImageSearchRequests) output.ImageSearchRequest {
	return output.ImageSearchRequest{
		ID: record.Id, ExternalID: record.ExternalId, Status: record.Status,
		PipelineName: record.PipelineName, PipelineVersion: record.PipelineVersion,
		GroupCount: record.GroupCount, MatchedCount: record.MatchedCount,
		NotMatchedCount: record.NotMatchedCount, RunningCount: record.RunningCount,
		FailedCount: record.FailedCount, CreatedAt: record.CreatedAt,
	}
}

func mapGroup(record entity.ImageSearchGroups) output.ImageSearchGroup {
	result := output.ImageSearchGroup{
		ID: record.Id, ExternalID: record.ExternalId, SubjectUserID: record.SubjectUserId,
		Status: record.Status, ItemCount: record.ItemCount, QueuedCount: record.QueuedCount,
		RunningCount: record.RunningCount, CompletedCount: record.CompletedCount,
		FailedCount: record.FailedCount, BestMatchID: record.BestMatchId,
	}
	if record.BestMatchId != "" {
		result.BestScore = &record.BestScore
	}
	return result
}
