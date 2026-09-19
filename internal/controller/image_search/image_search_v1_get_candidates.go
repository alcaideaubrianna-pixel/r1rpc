package image_search

import (
	"context"

	"r1rpc/api/image_search/v1"
)

func (c *ControllerV1) GetCandidates(ctx context.Context, req *v1.GetCandidatesReq) (res *v1.GetCandidatesRes, err error) {
	result, err := c.service.GetCandidates(ctx, req.ID, req.Page, req.PageSize, req.Matched)
	if err != nil {
		return nil, err
	}
	items := make([]v1.CandidateSummary, 0, len(result.Items))
	for _, item := range result.Items {
		images := make([]v1.CandidateImageSummary, 0, len(item.Images))
		for _, image := range item.Images {
			images = append(images, v1.CandidateImageSummary{ImageIndex: image.ImageIndex, ImageFileID: image.ImageFileID,
				ImageURL: image.ImageURL, DownloadStatus: image.DownloadStatus, ErrorMessage: image.ErrorMessage,
				Score: image.Score, Matched: image.Matched, PHashDistance: image.PHashDistance,
				DHashDistance: image.DHashDistance, AHashDistance: image.AHashDistance})
		}
		items = append(items, v1.CandidateSummary{
			ID: item.ID, SearchItemID: item.SearchItemID, Rank: item.Rank,
			ContentID: item.ContentID, Title: item.Title, AuthorID: item.AuthorID,
			AuthorName: item.AuthorName, CoverURL: item.CoverURL,
			ImageFileID:    item.ImageFileID,
			ImageURL:       item.ImageURL,
			DownloadStatus: item.DownloadStatus, ErrorMessage: item.ErrorMessage,
			AlgorithmVersion: item.AlgorithmVersion,
			PHashDistance:    item.PHashDistance, DHashDistance: item.DHashDistance,
			AHashDistance: item.AHashDistance, Score: item.Score, Matched: item.Matched,
			ImageCount: item.ImageCount, AnalyzedImages: item.AnalyzedImages, FailedImages: item.FailedImages,
			Images: images,
		})
	}
	return &v1.GetCandidatesRes{
		Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total,
	}, nil
}
