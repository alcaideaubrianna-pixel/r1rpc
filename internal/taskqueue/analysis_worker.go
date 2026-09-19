package taskqueue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"r1rpc/internal/dao"
	"r1rpc/internal/imaging"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/guid"
	"github.com/hibiken/asynq"
)

type normalizedCandidate struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	AuthorID        string   `json:"authorID"`
	AuthorName      string   `json:"authorName"`
	CoverURL        string   `json:"coverURL"`
	ImageURLs       []string `json:"imageURLs"`
	CoverImageIndex int      `json:"coverImageIndex"`
}

type normalizedSearchResponse struct {
	Items                 []json.RawMessage `json:"items"`
	Raw                   json.RawMessage   `json:"raw"`
	ResponseSchemaVersion int               `json:"responseSchemaVersion"`
	AgentBuild            string            `json:"agentBuild"`
}

func (p *Processor) ProcessImageAnalysis(ctx context.Context, task *asynq.Task) error {
	jobID, err := analysisJobID(task)
	if err != nil {
		return err
	}
	job, asset, item, err := p.loadAnalysisContext(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" {
		return nil
	}
	if _, err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, job.Id).
		Data(do.ImageJobs{Status: "running", Stage: "analyzing"}).Update(); err != nil {
		return err
	}
	_, _, err = p.persistCandidates(ctx, *job, *item)
	if err != nil {
		_ = p.failAnalysis(ctx, job.Id, "RESPONSE_PARSE_FAILED", err)
		return fmt.Errorf("解析搜索响应失败: %w", asynq.SkipRetry)
	}
	sourceHashes, err := p.sourceHashes(ctx, *asset)
	if err != nil {
		_ = p.failAnalysis(ctx, job.Id, "SOURCE_IMAGE_INVALID", err)
		return fmt.Errorf("源图片无法分析: %w", asynq.SkipRetry)
	}
	_ = sourceHashes // 源图哈希已持久化，下载 worker 会复用。
	if err := p.enqueueCandidateDownloads(ctx, job.Id); err != nil {
		return err
	}
	return p.finishDownloadStageIfReady(ctx, job.Id)
}

func (p *Processor) enqueueCandidateDownloads(ctx context.Context, jobID string) error {
	candidateColumns := dao.ImageCandidates.Columns()
	var candidates []entity.ImageCandidates
	if err := dao.ImageCandidates.Ctx(ctx).Fields(candidateColumns.Id).
		Where(candidateColumns.JobId, jobID).Scan(&candidates); err != nil {
		return gerror.Wrap(err, "读取待下载候选失败")
	}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Id)
	}
	if len(ids) == 0 {
		return nil
	}
	columns := dao.ImageCandidateImages.Columns()
	var images []entity.ImageCandidateImages
	if err := dao.ImageCandidateImages.Ctx(ctx).WhereIn(columns.CandidateId, ids).
		WhereIn(columns.DownloadStatus, []string{"pending", "queued"}).Scan(&images); err != nil {
		return gerror.Wrap(err, "读取候选图片下载任务失败")
	}
	for _, image := range images {
		if err := p.enqueuer.EnqueueImageDownload(ctx, image.Id); err != nil {
			return err
		}
		_, err := dao.ImageCandidateImages.Ctx(ctx).Where(columns.Id, image.Id).
			Data(do.ImageCandidateImages{DownloadStatus: "queued", ErrorMessage: ""}).Update()
		if err != nil {
			return err
		}
	}
	return nil
}

func analysisJobID(task *asynq.Task) (string, error) {
	var payload imageJobPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil || strings.TrimSpace(payload.JobID) == "" {
		return "", fmt.Errorf("无效图片分析任务 payload: %w", asynq.SkipRetry)
	}
	return payload.JobID, nil
}

func (p *Processor) loadAnalysisContext(ctx context.Context, jobID string) (*entity.ImageJobs, *entity.ImageAssets, *entity.ImageSearchItems, error) {
	job, asset, err := p.loadJob(ctx, jobID)
	if err != nil {
		return nil, nil, nil, err
	}
	var item entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Where(dao.ImageSearchItems.Columns().JobId, jobID).Scan(&item); err != nil {
		return nil, nil, nil, gerror.Wrap(err, "读取图片检索项失败")
	}
	return job, asset, &item, nil
}

func (p *Processor) persistCandidates(ctx context.Context, job entity.ImageJobs, item entity.ImageSearchItems) (string, []json.RawMessage, error) {
	var response normalizedSearchResponse
	if err := json.Unmarshal([]byte(job.NormalizedResponseJson), &response); err != nil {
		return "", nil, err
	}
	responseID := guid.S()
	if _, err := dao.ImageSearchResponses.Ctx(ctx).Data(do.ImageSearchResponses{
		Id: responseID, JobId: job.Id, RequestId: job.SearchRequestId,
		NormalizedJson: job.NormalizedResponseJson, RawJson: string(response.Raw),
		ItemCount: len(response.Items), ParseStatus: "parsed",
	}).InsertIgnore(); err != nil {
		return "", nil, gerror.Wrap(err, "保存图片搜索响应失败")
	}
	columns := dao.ImageSearchResponses.Columns()
	value, err := dao.ImageSearchResponses.Ctx(ctx).Where(columns.JobId, job.Id).Value(columns.Id)
	if err != nil || value.IsEmpty() {
		return "", nil, gerror.Wrap(err, "读取图片搜索响应失败")
	}
	responseID = value.String()
	for rank, raw := range response.Items {
		var candidate normalizedCandidate
		if err := json.Unmarshal(raw, &candidate); err != nil || strings.TrimSpace(candidate.ID) == "" {
			continue
		}
		_, err := dao.ImageCandidates.Ctx(ctx).Data(do.ImageCandidates{
			Id: guid.S(), ResponseId: responseID, JobId: job.Id, SearchItemId: item.Id,
			RankNo: rank, ContentId: candidate.ID, Title: candidate.Title,
			AuthorId: candidate.AuthorID, AuthorName: candidate.AuthorName,
			CoverUrl: candidate.CoverURL, RawItemJson: string(raw), DownloadStatus: "pending",
		}).InsertIgnore()
		if err != nil {
			return "", nil, gerror.Wrap(err, "保存图片候选失败")
		}
		candidateColumns := dao.ImageCandidates.Columns()
		var saved entity.ImageCandidates
		if err := dao.ImageCandidates.Ctx(ctx).Where(candidateColumns.JobId, job.Id).
			Where(candidateColumns.RankNo, rank).Scan(&saved); err != nil {
			return "", nil, gerror.Wrap(err, "读取图片候选失败")
		}
		URLs := candidate.ImageURLs
		if len(URLs) == 0 && candidate.CoverURL != "" {
			URLs = []string{candidate.CoverURL}
		}
		for imageIndex, imageURL := range URLs {
			if strings.TrimSpace(imageURL) == "" {
				continue
			}
			_, err = dao.ImageCandidateImages.Ctx(ctx).Data(do.ImageCandidateImages{
				Id: guid.S(), CandidateId: saved.Id, ImageIndex: imageIndex,
				SourceUrl: imageURL, DownloadStatus: "pending",
			}).InsertIgnore()
			if err != nil {
				return "", nil, gerror.Wrap(err, "保存候选图片明细失败")
			}
		}
	}
	return responseID, response.Items, nil
}

func (p *Processor) sourceHashes(ctx context.Context, asset entity.ImageAssets) (imaging.Hashes, error) {
	_, data, err := p.app.Files.Read(ctx, asset.FileId)
	if err != nil {
		return imaging.Hashes{}, err
	}
	decoded, err := imaging.Decode(data)
	if err != nil {
		return imaging.Hashes{}, err
	}
	hashes, err := imaging.ComputeHashes(decoded)
	if err != nil {
		return imaging.Hashes{}, err
	}
	_, err = dao.ImageAssets.Ctx(ctx).Where(dao.ImageAssets.Columns().Id, asset.Id).Data(do.ImageAssets{
		Phash: imaging.FormatHash(hashes.PHash), Dhash: imaging.FormatHash(hashes.DHash),
		Ahash: imaging.FormatHash(hashes.AHash), AlgorithmVersion: imaging.AlgorithmVersion,
	}).Update()
	return hashes, err
}

func imageSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
