package imagesearch

import (
	"context"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
)

type Enqueuer interface {
	EnqueueImageJob(ctx context.Context, jobID string, priority int) error
	EnqueueImageAnalysis(ctx context.Context, jobID string) error
}

type pendingJob struct {
	ID       string
	ItemID   string
	GroupID  string
	Priority int
}

func (s *Service) SetEnqueuer(enqueuer Enqueuer) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	s.enqueuer = enqueuer
}

func (s *Service) enqueueJobs(ctx context.Context, requestID string, jobs []pendingJob) error {
	s.queueMu.RLock()
	enqueuer := s.enqueuer
	s.queueMu.RUnlock()
	if enqueuer == nil {
		return nil
	}
	groups := make(map[string]int)
	for _, job := range jobs {
		if err := markJobQueued(ctx, job); err != nil {
			return err
		}
		groups[job.GroupID]++
	}
	for groupID, count := range groups {
		_, err := dao.ImageSearchGroups.Ctx(ctx).
			Where(dao.ImageSearchGroups.Columns().Id, groupID).
			Data(do.ImageSearchGroups{Status: "queued", QueuedCount: count}).Update()
		if err != nil {
			return gerror.Wrap(err, "更新图片检索组队列状态失败")
		}
	}
	_, err := dao.ImageSearchRequests.Ctx(ctx).
		Where(dao.ImageSearchRequests.Columns().Id, requestID).
		Data(do.ImageSearchRequests{Status: "queued"}).Update()
	if err != nil {
		return gerror.Wrap(err, "更新图片检索请求队列状态失败")
	}
	for _, job := range jobs {
		if err := enqueuer.EnqueueImageJob(ctx, job.ID, job.Priority); err != nil {
			return gerror.Wrap(err, "图片检索任务入队失败")
		}
	}
	return nil
}

func markJobQueued(ctx context.Context, job pendingJob) error {
	if _, err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, job.ID).
		Where(dao.ImageJobs.Columns().Status, statusCreated).
		Data(do.ImageJobs{Status: "queued", Stage: "queued"}).Update(); err != nil {
		return gerror.Wrap(err, "更新设备任务队列状态失败")
	}
	_, err := dao.ImageSearchItems.Ctx(ctx).Where(dao.ImageSearchItems.Columns().Id, job.ItemID).
		Where(dao.ImageSearchItems.Columns().Status, statusCreated).
		Data(do.ImageSearchItems{Status: "queued"}).Update()
	return gerror.Wrap(err, "更新图片检索项队列状态失败")
}

func (s *Service) enqueuePendingRequest(ctx context.Context, requestID string) error {
	s.queueMu.RLock()
	enqueuer := s.enqueuer
	s.queueMu.RUnlock()
	if enqueuer == nil {
		return nil
	}
	groupColumns := dao.ImageSearchGroups.Columns()
	var groups []entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Where(groupColumns.RequestId, requestID).Scan(&groups); err != nil {
		return gerror.Wrap(err, "读取待恢复图片检索组失败")
	}
	for _, group := range groups {
		itemColumns := dao.ImageSearchItems.Columns()
		var items []entity.ImageSearchItems
		if err := dao.ImageSearchItems.Ctx(ctx).Where(itemColumns.GroupId, group.Id).
			WhereIn(itemColumns.Status, []string{"created", "queued", "retry_wait"}).Scan(&items); err != nil {
			return gerror.Wrap(err, "读取待恢复图片检索项失败")
		}
		for _, item := range items {
			jobColumns := dao.ImageJobs.Columns()
			var job entity.ImageJobs
			if err := dao.ImageJobs.Ctx(ctx).Where(jobColumns.Id, item.JobId).Scan(&job); err != nil {
				return gerror.Wrap(err, "读取待恢复设备任务失败")
			}
			if job.Id != "" {
				if err := enqueuer.EnqueueImageJob(ctx, job.Id, job.Priority); err != nil {
					return gerror.Wrap(err, "恢复图片检索任务入队失败")
				}
			}
		}
	}
	return nil
}
