package imagesearch

import (
	"context"
	"sort"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *Service) GetCandidates(
	ctx context.Context,
	requestID string,
	page int,
	pageSize int,
	matched *bool,
) (*output.ImageCandidatePage, error) {
	if _, err := s.GetOne(ctx, requestID); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
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
	result := &output.ImageCandidatePage{Page: page, PageSize: pageSize, Items: []output.ImageCandidate{}}
	if len(groupIDs) == 0 {
		return result, nil
	}
	itemColumns := dao.ImageSearchItems.Columns()
	var items []entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Fields(itemColumns.Id).
		WhereIn(itemColumns.GroupId, groupIDs).Scan(&items); err != nil {
		return nil, gerror.Wrap(err, "查询图片检索项失败")
	}
	itemIDs := make([]string, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.Id)
	}
	if len(itemIDs) == 0 {
		return result, nil
	}
	candidateColumns := dao.ImageCandidates.Columns()
	model := dao.ImageCandidates.Ctx(ctx).WhereIn(candidateColumns.SearchItemId, itemIDs)
	if matched != nil {
		model = model.Where(candidateColumns.Matched, *matched)
	}
	total, err := model.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "统计图片候选失败")
	}
	var candidates []entity.ImageCandidates
	if err := model.OrderDesc(candidateColumns.Score).OrderAsc(candidateColumns.RankNo).
		Page(page, pageSize).Scan(&candidates); err != nil {
		return nil, gerror.Wrap(err, "查询图片候选失败")
	}
	candidateIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidateIDs = append(candidateIDs, candidate.Id)
	}
	matches := make(map[string]entity.ImageMatches, len(candidateIDs))
	type imageStats struct {
		Total    int
		Analyzed int
		Failed   int
	}
	stats := make(map[string]imageStats, len(candidateIDs))
	imageMap := make(map[string][]entity.ImageCandidateImages, len(candidateIDs))
	if len(candidateIDs) > 0 {
		matchColumns := dao.ImageMatches.Columns()
		var records []entity.ImageMatches
		if err := dao.ImageMatches.Ctx(ctx).WhereIn(matchColumns.CandidateId, candidateIDs).Scan(&records); err != nil {
			return nil, gerror.Wrap(err, "查询图片候选评分失败")
		}
		for _, record := range records {
			matches[record.CandidateId] = record
		}
		var images []entity.ImageCandidateImages
		imageColumns := dao.ImageCandidateImages.Columns()
		if err := dao.ImageCandidateImages.Ctx(ctx).WhereIn(imageColumns.CandidateId, candidateIDs).Scan(&images); err != nil {
			return nil, gerror.Wrap(err, "查询候选图片分析进度失败")
		}
		for _, image := range images {
			value := stats[image.CandidateId]
			value.Total++
			switch image.DownloadStatus {
			case "analyzed":
				value.Analyzed++
			case "failed":
				value.Failed++
			}
			stats[image.CandidateId] = value
			imageMap[image.CandidateId] = append(imageMap[image.CandidateId], image)
		}
	}
	result.Total = total
	for _, candidate := range candidates {
		match := matches[candidate.Id]
		imageStat := stats[candidate.Id]
		imageURL := ""
		if candidate.ImageFileId != "" && s.files != nil {
			imageURL, _ = s.files.URL(ctx, candidate.ImageFileId)
		}
		item := output.ImageCandidate{
			ID: candidate.Id, SearchItemID: candidate.SearchItemId, Rank: candidate.RankNo,
			ContentID: candidate.ContentId, Title: candidate.Title,
			AuthorID: candidate.AuthorId, AuthorName: candidate.AuthorName,
			CoverURL: candidate.CoverUrl, ImageFileID: candidate.ImageFileId,
			ImageURL:         imageURL,
			DownloadStatus:   candidate.DownloadStatus,
			ErrorMessage:     candidate.ErrorMessage,
			AlgorithmVersion: candidate.AlgorithmVersion, Score: candidate.Score, Matched: candidate.Matched != 0,
			ImageCount: imageStat.Total, AnalyzedImages: imageStat.Analyzed, FailedImages: imageStat.Failed,
		}
		candidateImages := imageMap[candidate.Id]
		sort.SliceStable(candidateImages, func(i, j int) bool {
			if candidateImages[i].DownloadStatus == "analyzed" && candidateImages[j].DownloadStatus != "analyzed" {
				return true
			}
			if candidateImages[i].DownloadStatus != "analyzed" && candidateImages[j].DownloadStatus == "analyzed" {
				return false
			}
			if candidateImages[i].Score != candidateImages[j].Score {
				return candidateImages[i].Score > candidateImages[j].Score
			}
			return candidateImages[i].ImageIndex < candidateImages[j].ImageIndex
		})
		item.Images = make([]output.CandidateImage, 0, len(candidateImages))
		for _, image := range candidateImages {
			URL := ""
			if image.ImageFileId != "" && s.files != nil {
				URL, _ = s.files.URL(ctx, image.ImageFileId)
			}
			entry := output.CandidateImage{ImageIndex: image.ImageIndex, ImageFileID: image.ImageFileId,
				ImageURL: URL, DownloadStatus: image.DownloadStatus, ErrorMessage: image.ErrorMessage,
				Score: image.Score, Matched: image.Matched != 0}
			if image.PhashDistance != 0 || image.DownloadStatus == "analyzed" {
				value := image.PhashDistance
				entry.PHashDistance = &value
			}
			if image.DhashDistance != 0 || image.DownloadStatus == "analyzed" {
				value := image.DhashDistance
				entry.DHashDistance = &value
			}
			if image.AhashDistance != 0 || image.DownloadStatus == "analyzed" {
				value := image.AhashDistance
				entry.AHashDistance = &value
			}
			item.Images = append(item.Images, entry)
		}
		if match.Id != "" {
			item.PHashDistance = &match.PhashDistance
			item.DHashDistance = &match.DhashDistance
			item.AHashDistance = &match.AhashDistance
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}
