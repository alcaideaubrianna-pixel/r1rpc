package imagetask

import (
	"context"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/errors/gerror"
)

// Enqueuer 隔离 Service 与具体任务中间件，Telegram 和 HTTP 不感知 Asynq。
type Enqueuer interface {
	EnqueueImageJob(ctx context.Context, jobID string, priority int) error
}

func (s *Service) SetEnqueuer(enqueuer Enqueuer) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	s.enqueuer = enqueuer
}

func (s *Service) enqueue(ctx context.Context, job *output.ImageJob, priority int) error {
	s.queueMu.RLock()
	enqueuer := s.enqueuer
	s.queueMu.RUnlock()
	if enqueuer == nil {
		return nil
	}
	if err := enqueuer.EnqueueImageJob(ctx, job.ID, priority); err != nil {
		return gerror.Wrap(err, "图片任务入队失败")
	}
	columns := dao.ImageJobs.Columns()
	_, err := dao.ImageJobs.Ctx(ctx).Where(columns.Id, job.ID).Where(columns.Status, statusCreated).
		Data(do.ImageJobs{Status: "queued", Stage: "queued"}).Update()
	return gerror.Wrap(err, "更新图片任务队列状态失败")
}
