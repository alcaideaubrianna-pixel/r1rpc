package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TaskTypeImageProcess  = "image:process:v1"
	TaskTypeImageAnalyze  = "image:analyze:v1"
	TaskTypeImageDownload = "image:download:v1"
	TaskTypeSourceScan    = "source:scan:v1"
	imageQueue            = "image"
	analysisQueue         = "image_analysis"
	downloadQueue         = "image_download"
	sourceScanQueue       = "source_scan"
)

type imageJobPayload struct {
	JobID string `json:"jobId"`
}

type imageDownloadPayload struct {
	ImageID string `json:"imageId"`
}
type sourceScanPayload struct {
	TaskID string `json:"taskId"`
}

// Enqueuer 将图片业务任务写入 Redis，Payload 只包含数据库任务 ID。
type Enqueuer struct {
	client *asynq.Client
}

func NewEnqueuer(redis asynq.RedisClientOpt) *Enqueuer {
	return &Enqueuer{client: asynq.NewClient(redis)}
}

func (e *Enqueuer) Close() error {
	return e.client.Close()
}

func (e *Enqueuer) EnqueueImageJob(ctx context.Context, jobID string, priority int) error {
	return e.enqueue(ctx, TaskTypeImageProcess, imageQueue, jobID)
}

func (e *Enqueuer) EnqueueImageAnalysis(ctx context.Context, jobID string) error {
	return e.enqueue(ctx, TaskTypeImageAnalyze, analysisQueue, jobID)
}

func (e *Enqueuer) EnqueueImageDownload(ctx context.Context, imageID string) error {
	payload, err := json.Marshal(imageDownloadPayload{ImageID: strings.TrimSpace(imageID)})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskTypeImageDownload, payload)
	_, err = e.client.EnqueueContext(ctx, task, asynq.Queue(downloadQueue), asynq.MaxRetry(3),
		asynq.Timeout(90*time.Second), asynq.Unique(10*time.Minute))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}
func (e *Enqueuer) EnqueueSourceScan(ctx context.Context, taskID string, priority int) error {
	payload, err := json.Marshal(sourceScanPayload{TaskID: taskID})
	if err != nil {
		return err
	}
	_, err = e.client.EnqueueContext(ctx, asynq.NewTask(TaskTypeSourceScan, payload), asynq.Queue(sourceScanQueue), asynq.MaxRetry(5), asynq.Timeout(2*time.Minute), asynq.Unique(4*time.Minute))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

func (e *Enqueuer) enqueue(ctx context.Context, taskType, queue, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	payload, err := json.Marshal(imageJobPayload{JobID: jobID})
	if err != nil {
		return err
	}
	task := asynq.NewTask(taskType, payload)
	options := []asynq.Option{
		asynq.Queue(queue),
		asynq.MaxRetry(5),
		asynq.Timeout(2 * time.Minute),
	}
	if taskType == TaskTypeImageProcess {
		options = append(options, asynq.Unique(10*time.Minute))
	}
	_, err = e.client.EnqueueContext(ctx, task, options...)
	// 相同任务 ID 已存在表示幂等入队成功，数据库状态仍可推进到 queued。
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

func RedisOptions(addr, password string, db int) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{Addr: addr, Password: password, DB: db}
}
