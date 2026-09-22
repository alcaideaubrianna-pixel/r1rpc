package imagesearch

import (
	"context"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *Service) RetryAnalysis(ctx context.Context, requestID string) error {
	detail, err := s.GetOne(ctx, requestID)
	if err != nil {
		return err
	}
	s.queueMu.RLock()
	enqueuer := s.enqueuer
	s.queueMu.RUnlock()
	if enqueuer == nil {
		return gerror.NewCode(gcode.CodeInternalError, "图片分析队列尚未初始化")
	}
	jobIDs := make([]string, 0, len(detail.Items))
	for _, item := range detail.Items {
		itemColumns := dao.ImageSearchItems.Columns()
		var record entity.ImageSearchItems
		if err := dao.ImageSearchItems.Ctx(ctx).Where(itemColumns.Id, item.ID).Scan(&record); err != nil {
			return gerror.Wrap(err, "读取待重试图片检索项失败")
		}
		if record.JobId != "" {
			jobIDs = append(jobIDs, record.JobId)
		}
	}
	if len(jobIDs) == 0 {
		return gerror.NewCode(gcode.CodeInvalidParameter, "当前请求没有可重试的分析任务")
	}
	if err := resetAnalysisRecords(ctx, requestID, detail.Groups, jobIDs); err != nil {
		return err
	}
	for _, jobID := range jobIDs {
		var job entity.ImageJobs
		if err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, jobID).Scan(&job); err != nil {
			return gerror.Wrap(err, "读取待重试设备任务失败")
		}
		if err := enqueuer.EnqueueImageJob(ctx, jobID, job.Priority); err != nil {
			return gerror.Wrap(err, "重新投递图片搜索任务失败")
		}
	}
	return nil
}

func resetAnalysisRecords(ctx context.Context, requestID string, groups []output.ImageSearchGroup, jobIDs []string) error {
	return dao.ImageJobs.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		candidateColumns := dao.ImageCandidates.Columns()
		var candidates []entity.ImageCandidates
		if err := tx.Model(dao.ImageCandidates.Table()).Ctx(ctx).
			WhereIn(candidateColumns.JobId, jobIDs).Scan(&candidates); err != nil {
			return gerror.Wrap(err, "读取待重试图片候选失败")
		}
		candidateIDs := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			candidateIDs = append(candidateIDs, candidate.Id)
		}
		if len(candidateIDs) > 0 {
			if _, err := tx.Model(dao.ImageMatches.Table()).Ctx(ctx).
				WhereIn(dao.ImageMatches.Columns().CandidateId, candidateIDs).Delete(); err != nil {
				return gerror.Wrap(err, "清理旧图片评分失败")
			}
			imageColumns := dao.ImageCandidateImages.Columns()
			if _, err := tx.Model(dao.ImageCandidateImages.Table()).Ctx(ctx).
				WhereIn(imageColumns.CandidateId, candidateIDs).Delete(); err != nil {
				return gerror.Wrap(err, "清理旧候选图片明细失败")
			}
			if _, err := tx.Model(dao.ImageCandidates.Table()).Ctx(ctx).
				WhereIn(candidateColumns.Id, candidateIDs).Delete(); err != nil {
				return gerror.Wrap(err, "清理旧图片候选失败")
			}
		}
		jobColumns := dao.ImageJobs.Columns()
		if _, err := tx.Model(dao.ImageJobs.Table()).Ctx(ctx).WhereIn(jobColumns.Id, jobIDs).
			Data(do.ImageJobs{
				Status: "queued", Stage: "queued", AssignedClientId: "", AssignedSessionIncarnation: "",
				UploadRequestId: "", SearchRequestId: "", UploadHandle: "", NormalizedResponseJson: "",
				ErrorCode: "", ErrorMessage: "", FinishedAt: gdb.Raw("NULL"),
			}).Update(); err != nil {
			return gerror.Wrap(err, "重置图片任务失败")
		}
		itemColumns := dao.ImageSearchItems.Columns()
		if _, err := tx.Model(dao.ImageSearchItems.Table()).Ctx(ctx).WhereIn(itemColumns.JobId, jobIDs).
			Data(do.ImageSearchItems{Status: "queued", ErrorCode: "", ErrorMessage: "", FinishedAt: gdb.Raw("NULL")}).Update(); err != nil {
			return gerror.Wrap(err, "重置图片检索项失败")
		}
		if _, err := tx.Model(dao.ImageSearchResponses.Table()).Ctx(ctx).
			WhereIn(dao.ImageSearchResponses.Columns().JobId, jobIDs).Delete(); err != nil {
			return gerror.Wrap(err, "清理旧图片搜索响应失败")
		}
		groupColumns := dao.ImageSearchGroups.Columns()
		for _, group := range groups {
			if _, err := tx.Model(dao.ImageSearchGroups.Table()).Ctx(ctx).Where(groupColumns.Id, group.ID).
				Data(do.ImageSearchGroups{
					Status: "queued", QueuedCount: group.ItemCount, RunningCount: 0, CompletedCount: 0,
					FailedCount: 0, CancelledCount: 0, BestMatchId: "", BestScore: gdb.Raw("NULL"),
					MatchedAt: gdb.Raw("NULL"), FinishedAt: gdb.Raw("NULL"),
				}).Update(); err != nil {
				return err
			}
		}
		_, err := tx.Model(dao.ImageSearchRequests.Table()).Ctx(ctx).
			Where(dao.ImageSearchRequests.Columns().Id, requestID).
			Data(do.ImageSearchRequests{Status: "queued", MatchedCount: 0, NotMatchedCount: 0, RunningCount: len(groups), FailedCount: 0, FinishedAt: gdb.Raw("NULL")}).Update()
		return gerror.Wrap(err, "重置图片检索请求失败")
	})
}
