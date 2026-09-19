package taskqueue

import (
	"context"
	"sync"

	"r1rpc/internal/app"
	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/hibiken/asynq"
)

type Runtime struct {
	server    *asynq.Server
	enqueuer  *Enqueuer
	app       *app.App
	closeOnce sync.Once
}

func NewRuntime(application *app.App, redis asynq.RedisClientOpt, concurrency int) *Runtime {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Runtime{
		app:      application,
		enqueuer: NewEnqueuer(redis),
		server: asynq.NewServer(redis, asynq.Config{
			Concurrency: concurrency,
			Queues:      map[string]int{imageQueue: 2, analysisQueue: 2, downloadQueue: 6},
		}),
	}
}

func (r *Runtime) Enqueuer() *Enqueuer {
	return r.enqueuer
}

func (r *Runtime) Start(ctx context.Context) error {
	processor := NewProcessor(r.app, r.enqueuer)
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskTypeImageProcess, processor.ProcessImageJob)
	mux.HandleFunc(TaskTypeImageAnalyze, processor.ProcessImageAnalysis)
	mux.HandleFunc(TaskTypeImageDownload, processor.ProcessImageDownload)
	if err := r.server.Start(mux); err != nil {
		return err
	}
	if err := r.recoverPending(ctx); err != nil {
		r.server.Shutdown()
		return err
	}
	return nil
}

func (r *Runtime) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		r.server.Shutdown()
		closeErr = r.enqueuer.Close()
	})
	return closeErr
}

func (r *Runtime) recoverPending(ctx context.Context) error {
	columns := dao.ImageJobs.Columns()
	var jobs []entity.ImageJobs
	if err := dao.ImageJobs.Ctx(ctx).
		WhereIn(columns.Status, []string{"created", "queued", "retry_wait"}).
		OrderAsc(columns.CreatedAt).Limit(1000).Scan(&jobs); err != nil {
		return err
	}
	for _, job := range jobs {
		if err := r.enqueuer.EnqueueImageJob(ctx, job.Id, job.Priority); err != nil {
			return err
		}
		_, err := dao.ImageJobs.Ctx(ctx).Where(columns.Id, job.Id).
			WhereIn(columns.Status, []string{"created", "queued", "retry_wait"}).
			Data(do.ImageJobs{Status: "queued", Stage: "queued"}).Update()
		if err != nil {
			return err
		}
	}
	var analysisJobs []entity.ImageJobs
	if err := dao.ImageJobs.Ctx(ctx).
		Where(columns.Source, "image_search").
		WhereIn(columns.Stage, []string{"response_received", "analyzing"}).
		OrderAsc(columns.CreatedAt).Limit(1000).Scan(&analysisJobs); err != nil {
		return err
	}
	for _, job := range analysisJobs {
		if err := r.enqueuer.EnqueueImageAnalysis(ctx, job.Id); err != nil {
			return err
		}
	}
	imageColumns := dao.ImageCandidateImages.Columns()
	var pendingImages []entity.ImageCandidateImages
	if err := dao.ImageCandidateImages.Ctx(ctx).
		WhereIn(imageColumns.DownloadStatus, []string{"pending", "queued", "downloading"}).
		OrderAsc(imageColumns.CreatedAt).Limit(5000).Scan(&pendingImages); err != nil {
		return err
	}
	for _, image := range pendingImages {
		if err := r.enqueuer.EnqueueImageDownload(ctx, image.Id); err != nil {
			return err
		}
		_, _ = dao.ImageCandidateImages.Ctx(ctx).Where(imageColumns.Id, image.Id).
			Data(do.ImageCandidateImages{DownloadStatus: "queued"}).Update()
	}
	g.Log().Info(ctx, g.Map{"event": "image_queue_recovered", "count": len(jobs)})
	g.Log().Info(ctx, g.Map{"event": "image_analysis_queue_recovered", "count": len(analysisJobs)})
	g.Log().Info(ctx, g.Map{"event": "image_download_queue_recovered", "count": len(pendingImages)})
	return nil
}
