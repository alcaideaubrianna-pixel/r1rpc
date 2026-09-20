package taskqueue

import (
	"context"
	"sync"
	"time"

	"r1rpc/internal/app"
	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/hibiken/asynq"
)

type Runtime struct {
	server           *asynq.Server
	enqueuer         *Enqueuer
	app              *app.App
	closeOnce        sync.Once
	cancel           context.CancelFunc
	scanInterval     time.Duration
	downloadInterval time.Duration
}

func NewRuntime(application *app.App, redis asynq.RedisClientOpt, apiConcurrency, downloadConcurrency int, scanInterval, downloadInterval time.Duration) *Runtime {
	if apiConcurrency < 1 {
		apiConcurrency = 2
	}
	if downloadConcurrency < 1 {
		downloadConcurrency = 6
	}
	if scanInterval <= 0 {
		scanInterval = 5 * time.Second
	}
	if downloadInterval <= 0 {
		downloadInterval = 2 * time.Second
	}
	return &Runtime{
		app:              application,
		enqueuer:         NewEnqueuer(redis),
		scanInterval:     scanInterval,
		downloadInterval: downloadInterval,
		server: asynq.NewServer(redis, asynq.Config{
			Concurrency: apiConcurrency*2 + downloadConcurrency,
			Queues: map[string]int{
				imageQueue:      apiConcurrency,
				analysisQueue:   apiConcurrency,
				sourceScanQueue: apiConcurrency,
				downloadQueue:   downloadConcurrency,
			},
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
	mux.HandleFunc(TaskTypeSourceScan, processor.ProcessSourceScan)
	mux.HandleFunc(TaskTypeSourcePrepare, processor.ProcessSourcePrepare)
	if err := r.server.Start(mux); err != nil {
		return err
	}
	if err := r.recoverAPI(ctx); err != nil {
		r.server.Shutdown()
		return err
	}
	if err := r.recoverDownloads(ctx); err != nil {
		r.server.Shutdown()
		return err
	}
	if err := r.recoverSourceScans(ctx); err != nil {
		r.server.Shutdown()
		return err
	}
	workerCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.runScanner(workerCtx)
	return nil
}

func (r *Runtime) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		if r.cancel != nil {
			r.cancel()
		}
		r.server.Shutdown()
		closeErr = r.enqueuer.Close()
	})
	return closeErr
}

func (r *Runtime) runScanner(ctx context.Context) {
	apiTicker := time.NewTicker(r.scanInterval)
	downloadTicker := time.NewTicker(r.downloadInterval)
	sourceTicker := time.NewTicker(r.scanInterval)
	defer apiTicker.Stop()
	defer downloadTicker.Stop()
	defer sourceTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-apiTicker.C:
			_ = r.recoverAPI(ctx)
		case <-downloadTicker.C:
			_ = r.recoverDownloads(ctx)
		case <-sourceTicker.C:
			_ = r.recoverSourceScans(ctx)
		}
	}
}

func (r *Runtime) recoverSourceScans(ctx context.Context) error {
	columns := dao.ChannelScanTasks.Columns()
	var tasks []entity.ChannelScanTasks
	if err := dao.ChannelScanTasks.Ctx(ctx).
		WhereIn(columns.Status, []string{"queued", "waiting", "failed"}).
		WhereLTE(columns.NextRunAt, time.Now()).
		OrderDesc(columns.Priority).OrderAsc(columns.NextRunAt).
		Limit(500).Scan(&tasks); err != nil {
		return err
	}
	for _, task := range tasks {
		if err := r.enqueuer.EnqueueSourceScan(ctx, task.Id, task.Priority); err != nil {
			return err
		}
	}
	var preparing []entity.ChannelScanTasks
	if err := dao.ChannelScanTasks.Ctx(ctx).Where(columns.Status, "preparing").OrderAsc(columns.UpdatedAt).Limit(100).Scan(&preparing); err != nil {
		return err
	}
	for _, task := range preparing {
		if err := r.enqueuer.EnqueueSourcePrepare(ctx, task.Id); err != nil {
			return err
		}
	}
	var legacy []entity.ChannelScanTasks
	if err := dao.ChannelScanTasks.Ctx(ctx).
		Where(columns.Status, "completed").Where(columns.ImageSearchRequestId, "").
		OrderAsc(columns.UpdatedAt).Limit(100).Scan(&legacy); err != nil {
		return err
	}
	for _, task := range legacy {
		if _, err := dao.ChannelScanTasks.Ctx(ctx).Where(columns.Id, task.Id).
			Data(do.ChannelScanTasks{Status: "preparing"}).Update(); err != nil {
			return err
		}
		if err := r.enqueuer.EnqueueSourcePrepare(ctx, task.Id); err != nil {
			return err
		}
	}
	g.Log().Info(ctx, g.Map{"event": "source_scan_queue_recovered", "count": len(tasks)})
	return nil
}

func (r *Runtime) recoverAPI(ctx context.Context) error {
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
	g.Log().Info(ctx, g.Map{"event": "image_api_queue_recovered", "count": len(jobs), "analysis_count": len(analysisJobs)})
	return nil
}

func (r *Runtime) recoverDownloads(ctx context.Context) error {
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
	g.Log().Info(ctx, g.Map{"event": "image_download_queue_recovered", "count": len(pendingImages)})
	return nil
}
