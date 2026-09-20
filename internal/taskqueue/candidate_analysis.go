package taskqueue

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"r1rpc/internal/dao"
	"r1rpc/internal/imaging"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/guid"
)

type bestCandidate struct {
	ID      string
	Score   float64
	Matched bool
}

type analysisStats struct {
	Valid    int
	Analyzed int
}

type matchPolicy struct {
	ScoreThreshold   float64 `json:"scoreThreshold"`
	MaxPHashDistance int     `json:"maxPHashDistance"`
	MaxDHashDistance int     `json:"maxDHashDistance"`
	MaxAHashDistance int     `json:"maxAHashDistance"`
	MaxCandidates    int     `json:"maxCandidates"`
}

func (p *Processor) analyzeCandidates(
	ctx context.Context,
	responseID string,
	item entity.ImageSearchItems,
	rawItems []json.RawMessage,
	source imaging.Hashes,
) (bestCandidate, analysisStats, error) {
	downloader := imaging.NewDownloader()
	var (
		best  bestCandidate
		stats analysisStats
	)
	policy, err := loadMatchPolicy(ctx, item.GroupId)
	if err != nil {
		return best, stats, err
	}
	for rank, raw := range rawItems {
		var normalized normalizedCandidate
		if err := json.Unmarshal(raw, &normalized); err != nil || strings.TrimSpace(normalized.ID) == "" {
			continue
		}
		stats.Valid++
		candidate, err := findCandidate(ctx, item.JobId, rank)
		if err != nil {
			return best, stats, err
		}
		if candidate.Id == "" || candidate.DownloadStatus == "analyzed" {
			if candidate.DownloadStatus == "analyzed" {
				stats.Analyzed++
			}
			if candidate.Score > best.Score {
				best = bestCandidate{ID: candidate.Id, Score: candidate.Score, Matched: candidate.Matched != 0}
			}
			continue
		}
		imageResult, ok, err := p.analyzeCandidateImages(ctx, candidate.Id, downloader, source, policy)
		if err != nil {
			return best, stats, err
		}
		if !ok {
			_ = markCandidateDownloadFailed(ctx, candidate.Id, gerror.New("候选笔记的全部图片均分析失败"))
			continue
		}
		if err := saveCandidateImageScore(ctx, candidate.Id, imageResult); err != nil {
			return best, stats, err
		}
		stats.Analyzed++
		if err := insertCandidateMatch(ctx, item.Id, candidate.Id, imageResult.Comparison, imageResult.Matched); err != nil {
			return best, stats, err
		}
		if imageResult.Comparison.Score > best.Score {
			best = bestCandidate{ID: candidate.Id, Score: imageResult.Comparison.Score, Matched: imageResult.Matched}
		}
	}
	_, err = dao.ImageSearchResponses.Ctx(ctx).Where(dao.ImageSearchResponses.Columns().Id, responseID).
		Data(do.ImageSearchResponses{ParseStatus: "analyzed"}).Update()
	return best, stats, gerror.Wrap(err, "更新图片搜索响应状态失败")
}

type candidateImageResult struct {
	FileID     string
	SHA256     string
	Hashes     imaging.Hashes
	Comparison imaging.Comparison
	Matched    bool
}

func (p *Processor) analyzeCandidateImages(
	ctx context.Context,
	candidateID string,
	downloader *imaging.Downloader,
	source imaging.Hashes,
	policy matchPolicy,
) (candidateImageResult, bool, error) {
	columns := dao.ImageCandidateImages.Columns()
	var images []entity.ImageCandidateImages
	if err := dao.ImageCandidateImages.Ctx(ctx).Where(columns.CandidateId, candidateID).
		OrderAsc(columns.ImageIndex).Scan(&images); err != nil {
		return candidateImageResult{}, false, gerror.Wrap(err, "读取候选图片明细失败")
	}
	var best candidateImageResult
	found := false
	for _, image := range images {
		result, ok := p.analyzeCandidateImage(ctx, image, downloader, source, policy)
		if !ok {
			continue
		}
		if !found || result.Comparison.Score > best.Comparison.Score {
			best, found = result, true
			if err := saveCandidateImageScore(ctx, candidateID, best); err != nil {
				return candidateImageResult{}, false, err
			}
		}
	}
	return best, found, nil
}

func (p *Processor) analyzeCandidateImage(
	ctx context.Context,
	image entity.ImageCandidateImages,
	downloader *imaging.Downloader,
	source imaging.Hashes,
	policy matchPolicy,
) (candidateImageResult, bool) {
	startedAt := time.Now()
	if image.ImageFileId != "" && image.AlgorithmVersion == imaging.AlgorithmVersion {
		if hashes, err := imaging.ParseHashes(image.Phash, image.Dhash, image.Ahash); err == nil {
			comparison := imaging.Compare(source, hashes)
			matched := isMatch(comparison, policy)
			_ = updateCandidateImage(ctx, image.Id, image.ImageFileId, image.ImageSha256, hashes, comparison, matched)
			log.Printf("image_analysis reused candidate=%s image=%d score=%.6f elapsed_ms=%d",
				image.CandidateId, image.ImageIndex, comparison.Score, time.Since(startedAt).Milliseconds())
			return candidateImageResult{image.ImageFileId, image.ImageSha256, hashes, comparison, matched}, true
		}
	}
	data, err := downloader.Download(ctx, image.SourceUrl)
	if err != nil {
		_ = markCandidateImageFailed(ctx, image.Id, err)
		log.Printf("image_analysis download_failed candidate=%s image=%d elapsed_ms=%d error=%v",
			image.CandidateId, image.ImageIndex, time.Since(startedAt).Milliseconds(), err)
		return candidateImageResult{}, false
	}
	decoded, err := imaging.Decode(data)
	if err != nil {
		_ = markCandidateImageFailed(ctx, image.Id, err)
		return candidateImageResult{}, false
	}
	hashes, err := imaging.ComputeHashes(decoded)
	if err != nil {
		_ = markCandidateImageFailed(ctx, image.Id, err)
		return candidateImageResult{}, false
	}
	stored, err := p.app.Files.SaveBytes(ctx, data, "xhs-candidate.jpg", 0)
	if err != nil {
		_ = markCandidateImageFailed(ctx, image.Id, err)
		return candidateImageResult{}, false
	}
	comparison := imaging.Compare(source, hashes)
	matched := isMatch(comparison, policy)
	sha256Value := imageSHA256(data)
	if err := updateCandidateImage(ctx, image.Id, stored.ID, sha256Value, hashes, comparison, matched); err != nil {
		return candidateImageResult{}, false
	}
	log.Printf("image_analysis completed candidate=%s image=%d score=%.6f matched=%t elapsed_ms=%d",
		image.CandidateId, image.ImageIndex, comparison.Score, matched, time.Since(startedAt).Milliseconds())
	return candidateImageResult{stored.ID, sha256Value, hashes, comparison, matched}, true
}

func updateCandidateImage(ctx context.Context, imageID, fileID, sha256Value string, hashes imaging.Hashes, comparison imaging.Comparison, matched bool) error {
	_, err := dao.ImageCandidateImages.Ctx(ctx).Where(dao.ImageCandidateImages.Columns().Id, imageID).
		Data(do.ImageCandidateImages{
			ImageFileId: fileID, DownloadStatus: "analyzed", ImageSha256: sha256Value,
			Phash: imaging.FormatHash(hashes.PHash), Dhash: imaging.FormatHash(hashes.DHash),
			Ahash: imaging.FormatHash(hashes.AHash), AlgorithmVersion: imaging.AlgorithmVersion,
			Score: comparison.Score, PhashDistance: comparison.PHashDistance,
			DhashDistance: comparison.DHashDistance, AhashDistance: comparison.AHashDistance,
			Matched: matched, ErrorMessage: "",
		}).Update()
	return gerror.Wrap(err, "保存候选图片评分失败")
}

func markCandidateImageFailed(ctx context.Context, imageID string, cause error) error {
	_, err := dao.ImageCandidateImages.Ctx(ctx).Where(dao.ImageCandidateImages.Columns().Id, imageID).
		Data(do.ImageCandidateImages{DownloadStatus: "failed", ErrorMessage: cause.Error()}).Update()
	return err
}

func saveCandidateImageScore(ctx context.Context, candidateID string, result candidateImageResult) error {
	_, err := dao.ImageCandidates.Ctx(ctx).Where(dao.ImageCandidates.Columns().Id, candidateID).
		Data(do.ImageCandidates{
			DownloadStatus: "analyzed", ImageFileId: result.FileID, ImageSha256: result.SHA256,
			Phash: imaging.FormatHash(result.Hashes.PHash), Dhash: imaging.FormatHash(result.Hashes.DHash),
			Ahash: imaging.FormatHash(result.Hashes.AHash), AlgorithmVersion: imaging.AlgorithmVersion,
			Score: result.Comparison.Score, Matched: result.Matched, ErrorMessage: "",
		}).Update()
	return gerror.Wrap(err, "保存图片候选最佳评分失败")
}

func insertCandidateMatch(ctx context.Context, itemID, candidateID string, comparison imaging.Comparison, matched bool) error {
	decision := "not_matched"
	if matched {
		decision = "matched"
	}
	_, err := dao.ImageMatches.Ctx(ctx).Data(do.ImageMatches{
		Id: guid.S(), SearchItemId: itemID, CandidateId: candidateID,
		Algorithm: "perceptual_hash", AlgorithmVersion: imaging.AlgorithmVersion,
		PhashDistance: comparison.PHashDistance, DhashDistance: comparison.DHashDistance,
		AhashDistance: comparison.AHashDistance, Score: comparison.Score, Decision: decision,
	}).InsertIgnore()
	return gerror.Wrap(err, "保存图片匹配评分失败")
}

func isMatch(comparison imaging.Comparison, policy matchPolicy) bool {
	return comparison.Score >= policy.ScoreThreshold &&
		comparison.PHashDistance <= policy.MaxPHashDistance &&
		comparison.DHashDistance <= policy.MaxDHashDistance &&
		comparison.AHashDistance <= policy.MaxAHashDistance
}

func loadMatchPolicy(ctx context.Context, groupID string) (matchPolicy, error) {
	policy := defaultMatchPolicy()
	columns := dao.ImageSearchGroups.Columns()
	value, err := dao.ImageSearchGroups.Ctx(ctx).Where(columns.Id, groupID).Value(columns.MatchPolicyJson)
	if err != nil {
		return policy, gerror.Wrap(err, "读取图片匹配策略失败")
	}
	if raw := strings.TrimSpace(value.String()); raw != "" && raw != "null" {
		if err := json.Unmarshal([]byte(raw), &policy); err != nil {
			return policy, gerror.Wrap(err, "解析图片匹配策略失败")
		}
	}
	if policy.ScoreThreshold <= 0 || policy.ScoreThreshold > 1 {
		policy.ScoreThreshold = 0.82
	}
	if policy.MaxPHashDistance <= 0 || policy.MaxPHashDistance > 64 {
		policy.MaxPHashDistance = 12
	}
	if policy.MaxDHashDistance <= 0 || policy.MaxDHashDistance > 64 {
		policy.MaxDHashDistance = 16
	}
	if policy.MaxAHashDistance <= 0 || policy.MaxAHashDistance > 64 {
		policy.MaxAHashDistance = 16
	}
	if policy.MaxCandidates <= 0 || policy.MaxCandidates > 100 {
		policy.MaxCandidates = 20
	}
	return policy, nil
}

func defaultMatchPolicy() matchPolicy {
	return matchPolicy{ScoreThreshold: 0.82, MaxPHashDistance: 12, MaxDHashDistance: 16, MaxAHashDistance: 16, MaxCandidates: 20}
}

func findCandidate(ctx context.Context, jobID string, rank int) (entity.ImageCandidates, error) {
	columns := dao.ImageCandidates.Columns()
	var candidate entity.ImageCandidates
	err := dao.ImageCandidates.Ctx(ctx).Where(columns.JobId, jobID).Where(columns.RankNo, rank).Scan(&candidate)
	return candidate, gerror.Wrap(err, "读取图片候选失败")
}

func markCandidateDownloadFailed(ctx context.Context, candidateID string, cause error) error {
	_, err := dao.ImageCandidates.Ctx(ctx).Where(dao.ImageCandidates.Columns().Id, candidateID).
		Data(do.ImageCandidates{DownloadStatus: "failed", ErrorMessage: cause.Error()}).Update()
	return err
}
