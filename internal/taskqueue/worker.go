package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"r1rpc/internal/app"
	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/rpc"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/hibiken/asynq"
)

type Processor struct {
	app      *app.App
	enqueuer *Enqueuer
}

func NewProcessor(application *app.App, enqueuer *Enqueuer) *Processor {
	return &Processor{app: application, enqueuer: enqueuer}
}

func (p *Processor) ProcessImageJob(ctx context.Context, task *asynq.Task) error {
	var payload imageJobPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil || strings.TrimSpace(payload.JobID) == "" {
		return fmt.Errorf("无效图片任务 payload: %w", asynq.SkipRetry)
	}
	job, asset, err := p.loadJob(ctx, payload.JobID)
	if err != nil {
		return err
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" || job.Status == "paused" {
		return nil
	}
	if job.Source == "image_search" && (job.Stage == "response_received" || job.Stage == "analyzing") {
		return p.enqueuer.EnqueueImageAnalysis(ctx, job.Id)
	}
	workflow, err := p.acquireDeviceWorkflow(ctx)
	if err != nil {
		return p.markRetry(ctx, job.Id, "DEVICE_UNAVAILABLE", err)
	}
	defer p.app.Hub.ReleaseWorkflowLease(workflow)
	attempt, err := p.markRunning(ctx, job.Id, "uploading_to_device")
	if err != nil {
		return err
	}
	if err := p.syncSearchJob(ctx, job.Id, "running", "", ""); err != nil {
		return err
	}

	upload, uploadRequestID, clientID, err := p.app.InvokeRPC(ctx, nil, app.DeviceGroup, "media.upload_image", app.InvokeRequest{
		RequestID: stageRequestID(job.Id, "u", attempt),
		ClientID:  workflow.ClientID,
		Payload: mustJSON(map[string]any{
			"image":   map[string]any{"fileId": asset.FileId},
			"purpose": "image_search",
		}),
		Timeout: 60,
	})
	if err != nil {
		return p.markRetry(ctx, job.Id, "DEVICE_UPLOAD_FAILED", err)
	}
	var uploadResponse struct {
		UploadHandle string `json:"uploadHandle"`
	}
	if err := json.Unmarshal(upload.Payload, &uploadResponse); err != nil || strings.TrimSpace(uploadResponse.UploadHandle) == "" {
		_ = p.markFailed(ctx, job.Id, "UPLOAD_HANDLE_INVALID", "设备未返回有效 uploadHandle")
		return fmt.Errorf("设备未返回有效 uploadHandle: %w", asynq.SkipRetry)
	}
	if err := p.markSearching(ctx, job.Id, clientID, uploadRequestID, uploadResponse.UploadHandle); err != nil {
		return err
	}

	search, searchRequestID, _, err := p.app.InvokeRPC(ctx, nil, app.DeviceGroup, "content.search_by_image", app.InvokeRequest{
		RequestID: stageRequestID(job.Id, "s", attempt),
		ClientID:  clientID,
		Payload: mustJSON(map[string]any{
			"uploadHandle":  uploadResponse.UploadHandle,
			"selectionMode": "native_auto",
			"limit":         20,
		}),
		Timeout: 60,
	})
	if err != nil {
		return p.markRetry(ctx, job.Id, "DEVICE_SEARCH_FAILED", err)
	}
	if job.Source == "image_search" {
		if err := p.markResponseReceived(ctx, job.Id, searchRequestID, search.Payload); err != nil {
			return err
		}
		return p.enqueuer.EnqueueImageAnalysis(ctx, job.Id)
	}
	if err := p.markCompleted(ctx, job.Id, searchRequestID, search.Payload); err != nil {
		return err
	}
	return p.refreshBatch(ctx, job.BatchId)
}

func (p *Processor) acquireDeviceWorkflow(ctx context.Context) (*rpc.WorkflowLease, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		lease, err := p.app.Hub.AcquireWorkflowLease(app.DeviceGroup, "media.upload_image", "content.search_by_image")
		if err == nil {
			return lease, nil
		}
		if !errors.Is(err, rpc.ErrNoOnlineClient) && !errors.Is(err, rpc.ErrNoCapableClient) && !errors.Is(err, rpc.ErrGroupSaturated) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *Processor) markResponseReceived(ctx context.Context, jobID, requestID string, response json.RawMessage) error {
	_, err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, jobID).Data(do.ImageJobs{
		Status: "running", Stage: "response_received", SearchRequestId: requestID,
		NormalizedResponseJson: string(response),
	}).Update()
	if err != nil {
		return err
	}
	return p.syncSearchJob(ctx, jobID, "analyzing", "", "")
}

func (p *Processor) loadJob(ctx context.Context, jobID string) (*entity.ImageJobs, *entity.ImageAssets, error) {
	var job entity.ImageJobs
	if err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, jobID).Scan(&job); err != nil {
		return nil, nil, gerror.Wrap(err, "读取图片任务失败")
	}
	var asset entity.ImageAssets
	if err := dao.ImageAssets.Ctx(ctx).Where(dao.ImageAssets.Columns().Id, job.SourceAssetId).Scan(&asset); err != nil {
		_ = p.markFailed(ctx, jobID, "ASSET_NOT_FOUND", "图片资产不存在")
		return nil, nil, fmt.Errorf("图片资产不存在: %w", asynq.SkipRetry)
	}
	return &job, &asset, nil
}

func (p *Processor) markRunning(ctx context.Context, jobID, stage string) (int, error) {
	cols := dao.ImageJobs.Columns()
	result, err := dao.ImageJobs.Ctx(ctx).Where(cols.Id, jobID).
		WhereIn(cols.Status, []string{"created", "queued", "retry_wait", "running"}).
		Data(do.ImageJobs{Status: "running", Stage: stage, Attempt: gdb.Raw("attempt + 1"), ErrorCode: "", ErrorMessage: ""}).Update()
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return 0, errors.New("图片任务状态不允许执行")
	}
	value, err := dao.ImageJobs.Ctx(ctx).Where(cols.Id, jobID).Value(cols.Attempt)
	if err != nil {
		return 0, err
	}
	return value.Int(), nil
}

func (p *Processor) markSearching(ctx context.Context, jobID, clientID, requestID, handle string) error {
	_, err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, jobID).Data(do.ImageJobs{
		Stage: "searching", AssignedClientId: clientID, UploadRequestId: requestID, UploadHandle: handle,
	}).Update()
	return err
}

func (p *Processor) markCompleted(ctx context.Context, jobID, requestID string, response json.RawMessage) error {
	_, err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, jobID).Data(do.ImageJobs{
		Status: "completed", Stage: "completed", SearchRequestId: requestID,
		NormalizedResponseJson: string(response), FinishedAt: time.Now(),
	}).Update()
	if err != nil {
		return err
	}
	return p.syncSearchJob(ctx, jobID, "search_completed", "", "")
}

func (p *Processor) markRetry(ctx context.Context, jobID, code string, cause error) error {
	retried, hasRetried := asynq.GetRetryCount(ctx)
	maxRetry, hasMaxRetry := asynq.GetMaxRetry(ctx)
	if hasRetried && hasMaxRetry && retried >= maxRetry {
		if err := p.markFailed(ctx, jobID, code, cause.Error()); err != nil {
			return err
		}
		return cause
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, updateErr := dao.ImageJobs.Ctx(persistCtx).Where(dao.ImageJobs.Columns().Id, jobID).Data(do.ImageJobs{
		Status: "retry_wait", Stage: "retry_wait", ErrorCode: code, ErrorMessage: cause.Error(),
	}).Update()
	if updateErr != nil {
		return errors.Join(cause, updateErr)
	}
	if syncErr := p.syncSearchJob(persistCtx, jobID, "retry_wait", code, cause.Error()); syncErr != nil {
		return errors.Join(cause, syncErr)
	}
	return cause
}

func (p *Processor) markFailed(ctx context.Context, jobID, code, message string) error {
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, err := dao.ImageJobs.Ctx(persistCtx).Where(dao.ImageJobs.Columns().Id, jobID).Data(do.ImageJobs{
		Status: "failed", Stage: "failed", ErrorCode: code, ErrorMessage: message, FinishedAt: time.Now(),
	}).Update()
	if err != nil {
		return err
	}
	return p.syncSearchJob(persistCtx, jobID, "failed", code, message)
}

func (p *Processor) refreshBatch(ctx context.Context, batchID string) error {
	if strings.TrimSpace(batchID) == "" {
		return nil
	}
	jobColumns := dao.ImageJobs.Columns()
	counts := map[string]int{}
	for _, status := range []string{"queued", "running", "completed", "failed", "cache_hit"} {
		count, err := dao.ImageJobs.Ctx(ctx).Where(jobColumns.BatchId, batchID).
			Where(jobColumns.Status, status).Count()
		if err != nil {
			return err
		}
		counts[status] = count
	}
	batchStatus := "running"
	if counts["queued"] == 0 && counts["running"] == 0 {
		batchStatus = "completed"
		if counts["failed"] > 0 {
			batchStatus = "partial_failed"
		}
	}
	_, err := dao.ImageBatches.Ctx(ctx).Where(dao.ImageBatches.Columns().Id, batchID).Data(do.ImageBatches{
		Status: batchStatus, QueuedCount: counts["queued"], RunningCount: counts["running"],
		CompletedCount: counts["completed"], FailedCount: counts["failed"], CacheHitCount: counts["cache_hit"],
	}).Update()
	return err
}

func mustJSON(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func stageRequestID(jobID, stage string, attempt int) string {
	return fmt.Sprintf("%s-%s-%d", jobID, stage, attempt)
}
