package taskqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"r1rpc/internal/dao"
	"r1rpc/internal/imaging"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/hibiken/asynq"
)

// ProcessImageDownload 独立处理一张候选图，使下载吞吐不占用设备搜索 worker。
func (p *Processor) ProcessImageDownload(ctx context.Context, task *asynq.Task) error {
	var payload imageDownloadPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil || strings.TrimSpace(payload.ImageID) == "" {
		return fmt.Errorf("无效候选图片下载 payload: %w", asynq.SkipRetry)
	}
	columns := dao.ImageCandidateImages.Columns()
	var image entity.ImageCandidateImages
	if err := dao.ImageCandidateImages.Ctx(ctx).Where(columns.Id, payload.ImageID).Scan(&image); err != nil {
		return gerror.Wrap(err, "读取候选图片下载任务失败")
	}
	if image.Id == "" || image.DownloadStatus == "analyzed" || image.DownloadStatus == "failed" {
		return nil
	}
	_, err := dao.ImageCandidateImages.Ctx(ctx).Where(columns.Id, image.Id).
		Data(do.ImageCandidateImages{DownloadStatus: "downloading", ErrorMessage: ""}).Update()
	if err != nil {
		return err
	}

	candidate, item, source, policy, err := loadDownloadContext(ctx, image.CandidateId)
	if err != nil {
		return err
	}
	result, ok := p.analyzeCandidateImage(ctx, image, imaging.NewDownloader(), source, policy)
	if ok {
		if candidate.ImageFileId == "" || result.Comparison.Score > candidate.Score {
			if err := saveCandidateImageScore(ctx, candidate.Id, result); err != nil {
				return err
			}
		}
	}
	if err := p.finalizeCandidateIfReady(ctx, candidate.Id, item.Id); err != nil {
		return err
	}
	return p.finishDownloadStageIfReady(ctx, candidate.JobId)
}

func loadDownloadContext(ctx context.Context, candidateID string) (entity.ImageCandidates, entity.ImageSearchItems, imaging.Hashes, matchPolicy, error) {
	var candidate entity.ImageCandidates
	if err := dao.ImageCandidates.Ctx(ctx).Where(dao.ImageCandidates.Columns().Id, candidateID).Scan(&candidate); err != nil {
		return candidate, entity.ImageSearchItems{}, imaging.Hashes{}, matchPolicy{}, err
	}
	var item entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Where(dao.ImageSearchItems.Columns().Id, candidate.SearchItemId).Scan(&item); err != nil {
		return candidate, item, imaging.Hashes{}, matchPolicy{}, err
	}
	var job entity.ImageJobs
	if err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, candidate.JobId).Scan(&job); err != nil {
		return candidate, item, imaging.Hashes{}, matchPolicy{}, err
	}
	var asset entity.ImageAssets
	if err := dao.ImageAssets.Ctx(ctx).Where(dao.ImageAssets.Columns().Id, job.SourceAssetId).Scan(&asset); err != nil {
		return candidate, item, imaging.Hashes{}, matchPolicy{}, err
	}
	hashes, err := imaging.ParseHashes(asset.Phash, asset.Dhash, asset.Ahash)
	if err != nil {
		return candidate, item, hashes, matchPolicy{}, gerror.Wrap(err, "解析源图片哈希失败")
	}
	policy, err := loadMatchPolicy(ctx, item.GroupId)
	return candidate, item, hashes, policy, err
}

func (p *Processor) finalizeCandidateIfReady(ctx context.Context, candidateID, itemID string) error {
	columns := dao.ImageCandidateImages.Columns()
	active, err := dao.ImageCandidateImages.Ctx(ctx).Where(columns.CandidateId, candidateID).
		WhereIn(columns.DownloadStatus, []string{"pending", "queued", "downloading"}).Count()
	if err != nil || active > 0 {
		return err
	}
	var bestImage entity.ImageCandidateImages
	if err := dao.ImageCandidateImages.Ctx(ctx).Where(columns.CandidateId, candidateID).
		Where(columns.DownloadStatus, "analyzed").OrderDesc(columns.Score).Limit(1).Scan(&bestImage); err != nil {
		return err
	}
	if bestImage.Id == "" {
		return markCandidateDownloadFailed(ctx, candidateID, gerror.New("候选笔记的全部图片均分析失败"))
	}
	comparison := imaging.Comparison{PHashDistance: bestImage.PhashDistance, DHashDistance: bestImage.DhashDistance,
		AHashDistance: bestImage.AhashDistance, Score: bestImage.Score}
	hashes, err := imaging.ParseHashes(bestImage.Phash, bestImage.Dhash, bestImage.Ahash)
	if err != nil {
		return gerror.Wrap(err, "解析最佳候选图片哈希失败")
	}
	if err := saveCandidateImageScore(ctx, candidateID, candidateImageResult{FileID: bestImage.ImageFileId,
		SHA256: bestImage.ImageSha256, Hashes: hashes, Comparison: comparison, Matched: bestImage.Matched != 0}); err != nil {
		return err
	}
	return insertCandidateMatch(ctx, itemID, candidateID, comparison, bestImage.Matched != 0)
}

func (p *Processor) finishDownloadStageIfReady(ctx context.Context, jobID string) error {
	candidateColumns := dao.ImageCandidates.Columns()
	var candidates []entity.ImageCandidates
	if err := dao.ImageCandidates.Ctx(ctx).Where(candidateColumns.JobId, jobID).Scan(&candidates); err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Id)
	}
	imageColumns := dao.ImageCandidateImages.Columns()
	active, err := dao.ImageCandidateImages.Ctx(ctx).WhereIn(imageColumns.CandidateId, ids).
		WhereIn(imageColumns.DownloadStatus, []string{"pending", "queued", "downloading"}).Count()
	if err != nil || active > 0 {
		return err
	}
	var job entity.ImageJobs
	if err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, jobID).Scan(&job); err != nil {
		return err
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" {
		return nil
	}
	var item entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Where(dao.ImageSearchItems.Columns().JobId, jobID).Scan(&item); err != nil {
		return err
	}
	best := bestCandidate{}
	for _, candidate := range candidates {
		if candidate.DownloadStatus == "analyzed" && (best.ID == "" || candidate.Score > best.Score) {
			best = bestCandidate{ID: candidate.Id, Score: candidate.Score, Matched: candidate.Matched != 0}
		}
	}
	if best.ID == "" {
		return p.failAnalysis(ctx, jobID, "CANDIDATE_ANALYSIS_FAILED", gerror.New("所有候选图片均分析失败"))
	}
	_, _ = dao.ImageSearchResponses.Ctx(ctx).Where(dao.ImageSearchResponses.Columns().JobId, jobID).
		Data(do.ImageSearchResponses{ParseStatus: "analyzed"}).Update()
	return p.finalizeAnalysis(ctx, job, item, best)
}
