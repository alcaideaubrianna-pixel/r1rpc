package taskqueue

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"r1rpc/internal/dao"
	"r1rpc/internal/imaging"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/model/input"
	"r1rpc/internal/ocr"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/guid"
	"github.com/hibiken/asynq"
)

const sourceMaterialTimeout = 100 * time.Second

func (p *Processor) ProcessSourcePrepare(ctx context.Context, task *asynq.Task) error {
	var payload sourceScanPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil || strings.TrimSpace(payload.TaskID) == "" {
		return fmt.Errorf("无效来源图片准备任务: %w", asynq.SkipRetry)
	}
	return p.prepareSourceSearch(ctx, payload.TaskID)
}

func (p *Processor) ProcessSourceItem(ctx context.Context, task *asynq.Task) error {
	var payload sourceItemPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil || payload.ScanTaskID == "" || payload.NoteID == "" {
		return fmt.Errorf("无效资料预处理 payload: %w", asynq.SkipRetry)
	}
	var scan entity.ChannelScanTasks
	if err := dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, payload.ScanTaskID).Scan(&scan); err != nil {
		return err
	}
	if scan.Status == "cancelled" || scan.Status == "paused" || scan.Mode == "sync" {
		return nil
	}
	config, _, err := loadSearchConfig(ctx, scan.SearchConfigId)
	if err != nil {
		return err
	}
	itemID, requestID, err := p.createSourceSearchItem(ctx, scan, config, payload.SearchTaskID, payload.NoteID)
	if err != nil {
		return err
	}
	if requestID != "" {
		if err = p.processSourceMaterial(ctx, itemID, requestID); err != nil {
			return err
		}
	}
	return p.refreshSourceItemProgress(ctx, scan.Id, payload.SearchTaskID)
}

// refreshSourceItemProgress 汇总资料级任务状态，避免批次任务提前宣告完成。
func (p *Processor) refreshSourceItemProgress(ctx context.Context, scanTaskID, searchTaskID string) error {
	cols := dao.SearchTaskItems.Columns()
	var items []entity.SearchTaskItems
	if err := dao.SearchTaskItems.Ctx(ctx).Where(cols.SearchTaskId, searchTaskID).Scan(&items); err != nil {
		return err
	}
	completed, failed := 0, 0
	for _, item := range items {
		switch item.Status {
		case "completed", "matched", "filtered":
			completed++
		case "failed":
			failed++
		}
	}
	status := "running"
	if completed+failed >= len(items) && len(items) > 0 {
		status = "completed"
		if failed > 0 {
			status = "partial_failed"
		}
	}
	if _, err := dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, searchTaskID).Data(do.SearchTasks{Status: status, FailedCount: failed}).Update(); err != nil {
		return err
	}
	_, err := dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, scanTaskID).Data(do.ChannelScanTasks{Status: status}).Update()
	return err
}

func (p *Processor) prepareSourceSearch(ctx context.Context, taskID string) error {
	var scan entity.ChannelScanTasks
	if err := dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, taskID).Scan(&scan); err != nil {
		return err
	}
	if scan.Status == "cancelled" || scan.Status == "paused" {
		return nil
	}
	config, snapshot, err := loadSearchConfig(ctx, scan.SearchConfigId)
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	searchTaskID := sourceSearchTaskID(scan)
	relations, err := loadScanRelations(ctx, scan)
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	if len(relations) == 0 {
		cause := gerror.New("扫描结果没有资料")
		_, _ = dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, searchTaskID).
			Data(do.SearchTasks{Status: "failed", ErrorMessage: cause.Error()}).Update()
		return p.markSourcePrepareError(ctx, scan.Id, cause)
	}
	fetchedCount := len(relations)
	if scan.Mode != "selected" {
		relations, err = excludeProcessedNotes(ctx, relations)
		if err != nil {
			return p.markSourcePrepareError(ctx, scan.Id, err)
		}
	}
	if len(relations) == 0 {
		status := "completed"
		nextRunAt := any(nil)
		if scan.Mode == "continuous" {
			status = "waiting"
			nextRunAt = time.Now().Add(time.Duration(scan.PollIntervalMinutes) * time.Minute)
		}
		_, err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, scan.Id).
			Data(do.ChannelScanTasks{Status: status, NextRunAt: nextRunAt, FetchedCount: fetchedCount, SkippedCount: fetchedCount, LastError: fmt.Sprintf("拉取 %d 条，新增 0 条，跳过 %d 条已扫描资料", fetchedCount, fetchedCount)}).Update()
		return err
	}
	_, err = dao.SearchTasks.Ctx(ctx).Data(do.SearchTasks{Id: searchTaskID, SourceType: "channel_scan", SourceTaskId: searchTaskID, Title: scan.ChannelTitle, ConfigId: config.Id, ConfigSnapshotJson: snapshot, Status: "preparing"}).InsertIgnore()
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	_, err = dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, searchTaskID).Data(do.SearchTasks{
		SourceTaskId: scan.Id, Status: "preparing", TotalCount: len(relations), ErrorMessage: "",
	}).Update()
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	_, err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, scan.Id).Data(do.ChannelScanTasks{
		SearchTaskId: searchTaskID, FetchedCount: fetchedCount, SkippedCount: fetchedCount - len(relations), LastError: "",
	}).Update()
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	created := 0
	for _, relation := range relations {
		if err = p.enqueuer.EnqueueSourceItem(ctx, scan.Id, searchTaskID, relation.NoteId); err != nil {
			return p.markSourcePrepareError(ctx, scan.Id, err)
		}
		created++
	}
	_, _ = dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, searchTaskID).Data(do.SearchTasks{Status: "running", TotalCount: created, SearchCount: created, ErrorMessage: ""}).Update()
	// 资料任务已经异步投递，批次本身仍处于准备中，不能提前显示完成。
	status := "preparing"
	if scan.Mode == "continuous" {
		status = "preparing"
	}
	_, err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, scan.Id).
		Data(do.ChannelScanTasks{Status: status, SearchTaskId: searchTaskID, FetchedCount: fetchedCount, SkippedCount: fetchedCount - created, LastError: ""}).Update()
	return err
}

func excludeProcessedNotes(ctx context.Context, relations []entity.SourceScanTaskNotes) ([]entity.SourceScanTaskNotes, error) {
	noteIDs := make([]string, 0, len(relations))
	for _, relation := range relations {
		noteIDs = append(noteIDs, relation.NoteId)
	}
	columns := dao.SearchTaskItems.Columns()
	var processed []entity.SearchTaskItems
	if err := dao.SearchTaskItems.Ctx(ctx).Fields(columns.SourceNoteId).
		WhereIn(columns.SourceNoteId, noteIDs).Scan(&processed); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(processed))
	for _, item := range processed {
		seen[item.SourceNoteId] = struct{}{}
	}
	result := make([]entity.SourceScanTaskNotes, 0, len(relations))
	for _, relation := range relations {
		if _, exists := seen[relation.NoteId]; !exists {
			result = append(result, relation)
		}
	}
	return result, nil
}

func loadScanRelations(ctx context.Context, scan entity.ChannelScanTasks) ([]entity.SourceScanTaskNotes, error) {
	relationColumns := dao.SourceScanTaskNotes.Columns()
	var relations []entity.SourceScanTaskNotes
	if err := dao.SourceScanTaskNotes.Ctx(ctx).Where(relationColumns.ScanTaskId, scan.Id).
		OrderAsc(relationColumns.CreatedAt).Limit(scan.InitialLimit).Scan(&relations); err != nil {
		return nil, err
	}
	if len(relations) == 0 {
		var legacyNotes []entity.SourceNotes
		noteColumns := dao.SourceNotes.Columns()
		model := dao.SourceNotes.Ctx(ctx).Where(noteColumns.ScanTaskId, scan.Id)
		if scan.Mode == "once" {
			model = dao.SourceNotes.Ctx(ctx).
				Where(noteColumns.DataSourceId, scan.DataSourceId).
				Where(noteColumns.ChannelId, scan.ChannelId).
				OrderDesc(noteColumns.CreatedAt).
				Limit(scan.InitialLimit)
		}
		if err := model.Scan(&legacyNotes); err != nil {
			return nil, err
		}
		for _, note := range legacyNotes {
			relations = append(relations, entity.SourceScanTaskNotes{ScanTaskId: scan.Id, NoteId: note.Id})
			_, _ = dao.SourceScanTaskNotes.Ctx(ctx).Data(do.SourceScanTaskNotes{Id: guid.S(), ScanTaskId: scan.Id, NoteId: note.Id}).InsertIgnore()
		}
	}
	return relations, nil
}

func loadSearchConfig(ctx context.Context, id string) (entity.SearchConfigs, string, error) {
	var config entity.SearchConfigs
	model := dao.SearchConfigs.Ctx(ctx).Where(dao.SearchConfigs.Columns().Enabled, 1)
	if id != "" {
		model = model.Where(dao.SearchConfigs.Columns().Id, id)
	}
	if err := model.OrderDesc(dao.SearchConfigs.Columns().UpdatedAt).Limit(1).Scan(&config); err != nil {
		return config, "", err
	}
	if config.Id == "" {
		if id != "" {
			return config, "", gerror.New("指定的搜索配置不存在或已停用")
		}
		config = entity.SearchConfigs{Id: "", Name: "默认配置", ScoreThreshold: 0.82, MaxPhashDistance: 12, MaxDhashDistance: 16, MaxAhashDistance: 16, MaxCandidates: 20, PipelineName: "hash-v1", Enabled: 1, Version: 1}
	}
	raw, _ := json.Marshal(config)
	return config, string(raw), nil
}

func (p *Processor) createSourceSearchItem(ctx context.Context, scan entity.ChannelScanTasks, config entity.SearchConfigs, searchTaskID, noteID string) (string, string, error) {
	var note entity.SourceNotes
	if err := dao.SourceNotes.Ctx(ctx).Where(dao.SourceNotes.Columns().Id, noteID).Scan(&note); err != nil {
		return "", "", err
	}
	itemID := guid.S()
	_, err := dao.SearchTaskItems.Ctx(ctx).Data(do.SearchTaskItems{Id: itemID, SearchTaskId: searchTaskID, SourceNoteId: note.Id, ExternalId: note.ExternalNoteId, Title: note.Title, Status: "preparing", PreprocessStatus: "running", FilterStatus: "pending"}).InsertIgnore()
	if err != nil {
		return "", "", err
	}
	var item entity.SearchTaskItems
	itemColumns := dao.SearchTaskItems.Columns()
	if err = dao.SearchTaskItems.Ctx(ctx).Where(itemColumns.SearchTaskId, searchTaskID).Where(itemColumns.SourceNoteId, note.Id).Scan(&item); err != nil {
		return "", "", err
	}
	itemID = item.Id
	if item.ImageSearchRequestId != "" {
		return itemID, item.ImageSearchRequestId, nil
	}
	prepared, err := p.prepareNoteImages(ctx, scan.DataSourceId, note.Id, config)
	if err != nil {
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", PreprocessStatus: "failed", ErrorMessage: err.Error()}).Update()
		return itemID, "", nil
	}
	if len(prepared.Items) == 0 {
		if prepared.Blocked > 0 && prepared.Failed == 0 {
			_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{
				Status: "filtered", PreprocessStatus: "completed", FilterStatus: "blocked", FilterReason: "全部图片命中 OCR 屏蔽关键字",
			}).Update()
			return itemID, "", nil
		}
		message := "资料没有可用图片"
		if prepared.Failed > 0 {
			message = fmt.Sprintf("%d 张图片下载或 OCR 失败", prepared.Failed)
		}
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", PreprocessStatus: "failed", ErrorMessage: message}).Update()
		return itemID, "", nil
	}
	priority := scan.Priority
	if priority < -10 {
		priority = -10
	}
	if priority > 10 {
		priority = 10
	}
	request, err := p.app.ImageSearch.Create(ctx, input.CreateImageSearchRequest{
		Source: "channel_scan_item", ExternalID: "source-note:" + searchTaskID + ":" + note.Id,
		Priority: priority, PipelineName: config.PipelineName, RequestedBySubject: note.Title,
		Groups: []input.ImageSearchGroup{{
			ExternalID: note.ExternalNoteId, Images: prepared.Items,
			MatchPolicy: map[string]any{"scoreThreshold": config.ScoreThreshold, "maxPHashDistance": config.MaxPhashDistance, "maxDHashDistance": config.MaxDhashDistance, "maxAHashDistance": config.MaxAhashDistance, "maxCandidates": config.MaxCandidates},
		}},
		DeferQueue: true, InitialJobStage: "material_queued",
	})
	if err != nil {
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", ErrorMessage: err.Error()}).Update()
		return itemID, "", nil
	}
	filterReason := ""
	if prepared.Blocked > 0 {
		filterReason = fmt.Sprintf("OCR 已屏蔽 %d 张图片", prepared.Blocked)
	}
	_, err = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "queued", PreprocessStatus: "completed", FilterStatus: "passed", FilterReason: filterReason, ImageSearchRequestId: request.ID, ErrorMessage: ""}).Update()
	return itemID, request.ID, err
}

func (p *Processor) processSourceMaterial(ctx context.Context, itemID, requestID string) error {
	workflow, err := p.acquireDeviceWorkflow(ctx)
	if err != nil {
		return p.markSourceMaterialRetry(ctx, itemID, err)
	}
	defer p.app.Hub.ReleaseWorkflowLease(workflow)

	materialCtx, cancel := context.WithTimeout(ctx, sourceMaterialTimeout)
	defer cancel()
	itemColumns := dao.SearchTaskItems.Columns()
	if _, err = dao.SearchTaskItems.Ctx(materialCtx).Where(itemColumns.Id, itemID).
		Data(do.SearchTaskItems{Status: "running", ErrorMessage: ""}).Update(); err != nil {
		return err
	}
	jobIDs, err := sourceMaterialJobIDs(materialCtx, requestID)
	if err != nil {
		return err
	}
	for _, jobID := range jobIDs {
		if err = p.processImageJobWithWorkflow(materialCtx, jobID, workflow, true); err != nil {
			return p.markSourceMaterialRetry(ctx, itemID, err)
		}
	}
	return nil
}

func sourceMaterialJobIDs(ctx context.Context, requestID string) ([]string, error) {
	groupColumns := dao.ImageSearchGroups.Columns()
	var groups []entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Fields(groupColumns.Id).
		Where(groupColumns.RequestId, requestID).Scan(&groups); err != nil {
		return nil, err
	}
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.Id)
	}
	if len(groupIDs) == 0 {
		return nil, nil
	}
	itemColumns := dao.ImageSearchItems.Columns()
	var items []entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Fields(itemColumns.JobId).
		WhereIn(itemColumns.GroupId, groupIDs).OrderAsc(itemColumns.Ordinal).Scan(&items); err != nil {
		return nil, err
	}
	jobIDs := make([]string, 0, len(items))
	for _, item := range items {
		jobIDs = append(jobIDs, item.JobId)
	}
	return jobIDs, nil
}

func (p *Processor) markSourceMaterialRetry(ctx context.Context, itemID string, cause error) error {
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	status := "retry_wait"
	retried, hasRetried := asynq.GetRetryCount(ctx)
	maxRetry, hasMaxRetry := asynq.GetMaxRetry(ctx)
	if hasRetried && hasMaxRetry && retried >= maxRetry {
		status = "failed"
	}
	_, updateErr := dao.SearchTaskItems.Ctx(persistCtx).Where(dao.SearchTaskItems.Columns().Id, itemID).
		Data(do.SearchTaskItems{Status: status, ErrorMessage: cause.Error()}).Update()
	if updateErr != nil {
		return errors.Join(cause, updateErr)
	}
	return cause
}

func sourceSearchTaskID(scan entity.ChannelScanTasks) string {
	cycle := scan.LastSuccessAt.UTC().Format("20060102T150405.000000000")
	sum := sha256.Sum256([]byte(scan.Id + ":" + cycle))
	return fmt.Sprintf("%x", sum[:16])
}

type preparedImages struct {
	Items   []input.ImageSearchItem
	Blocked int
	Failed  int
}

func (p *Processor) prepareNoteImages(ctx context.Context, sourceID, noteID string, config entity.SearchConfigs) (preparedImages, error) {
	columns := dao.SourceNoteImages.Columns()
	var records []entity.SourceNoteImages
	if err := dao.SourceNoteImages.Ctx(ctx).Where(columns.NoteId, noteID).
		WhereNot(columns.DownloadStatus, "unavailable").OrderAsc(columns.ImageIndex).Scan(&records); err != nil {
		return preparedImages{}, err
	}
	result := preparedImages{Items: make([]input.ImageSearchItem, 0, len(records))}
	var source entity.DataSources
	if err := dao.DataSources.Ctx(ctx).Where(dao.DataSources.Columns().Id, sourceID).Scan(&source); err != nil {
		return result, err
	}
	imageBase := strings.TrimSpace(source.ImageBaseUrl)
	if imageBase == "" {
		imageBase = source.BaseUrl
	}
	downloader, err := imaging.NewSourceDownloader(imageBase)
	if err != nil {
		return result, err
	}
	keywords, err := searchOCRKeywords(config.OcrKeywordsJson)
	if err != nil {
		return result, err
	}
	processor := ocr.NewTesseract()
	for _, record := range records {
		fileID := record.FileId
		if fileID != "" {
			if _, _, readErr := p.app.Files.Read(ctx, fileID); readErr != nil {
				fileID = ""
			}
		}
		if fileID == "" {
			_, _ = dao.SourceNoteImages.Ctx(ctx).Where(columns.Id, record.Id).
				Data(do.SourceNoteImages{DownloadStatus: "downloading", FilterReason: ""}).Update()
			assetID, parseErr := strconv.ParseInt(record.ExternalAssetId, 10, 64)
			var data []byte
			var err error
			parsedURL, _ := url.Parse(strings.TrimSpace(record.SourceUrl))
			if parsedURL != nil && parsedURL.IsAbs() {
				data, err = downloader.Download(ctx, record.SourceUrl)
			} else if parseErr == nil && assetID > 0 {
				data, err = p.app.DataSources.DownloadAsset(ctx, sourceID, assetID)
			} else {
				data, err = p.downloadSourceImage(ctx, source, downloader, record.SourceUrl)
			}
			if err != nil {
				g.Log().Errorf(ctx, "资料图片下载失败 noteID=%s imageID=%s index=%d url=%s err=%v", noteID, record.Id, record.ImageIndex, record.SourceUrl, err)
				p.failSourceImage(ctx, record.Id, err)
				result.Failed++
				continue
			}
			stored, err := p.app.Files.SaveBytes(ctx, data, "source-image.jpg", 0)
			if err != nil {
				g.Log().Errorf(ctx, "资料图片存储失败 noteID=%s imageID=%s index=%d err=%v", noteID, record.Id, record.ImageIndex, err)
				p.failSourceImage(ctx, record.Id, err)
				result.Failed++
				continue
			}
			fileID = stored.ID
			_, err = dao.SourceNoteImages.Ctx(ctx).Where(columns.Id, record.Id).Data(do.SourceNoteImages{FileId: stored.ID, Sha256: stored.SHA256, DownloadStatus: "downloaded", FilterReason: ""}).Update()
			if err != nil {
				return result, err
			}
		}
		// 没有配置屏蔽词时无需启动 OCR；否则 OCR 环境问题会把本可搜索的图片全部判为失败。
		if config.OcrEnabled == 1 && len(keywords) > 0 {
			blocked, recognizeErr := p.preprocessSourceImage(ctx, processor, record, fileID, keywords, config.OcrMatchMode)
			if recognizeErr != nil {
				g.Log().Errorf(ctx, "资料图片 OCR 失败 noteID=%s imageID=%s index=%d err=%v", noteID, record.Id, record.ImageIndex, recognizeErr)
				result.Failed++
				continue
			}
			if blocked {
				result.Blocked++
				continue
			}
		}
		result.Items = append(result.Items, input.ImageSearchItem{FileID: fileID})
	}
	return result, nil
}

func (p *Processor) preprocessSourceImage(ctx context.Context, processor ocr.Processor, record entity.SourceNoteImages, fileID string, keywords []string, mode string) (bool, error) {
	columns := dao.SourceNoteImages.Columns()
	text := record.OcrText
	if record.PreprocessStatus != "completed" {
		_, data, err := p.app.Files.Read(ctx, fileID)
		if err != nil {
			return false, err
		}
		ocrCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		text, err = processor.Recognize(ocrCtx, data)
		if err != nil {
			_, _ = dao.SourceNoteImages.Ctx(ctx).Where(columns.Id, record.Id).Data(do.SourceNoteImages{
				PreprocessStatus: "failed", FilterDecision: "error", FilterReason: err.Error(),
			}).Update()
			return false, err
		}
	}
	blocked := ocr.Matches(text, keywords, mode)
	decision := "passed"
	reason := ""
	if blocked {
		decision = "blocked"
		reason = "命中 OCR 屏蔽关键字"
	}
	_, err := dao.SourceNoteImages.Ctx(ctx).Where(columns.Id, record.Id).Data(do.SourceNoteImages{
		PreprocessStatus: "completed", OcrText: text, FilterDecision: decision, FilterReason: reason,
	}).Update()
	return blocked, err
}

func searchOCRKeywords(raw string) ([]string, error) {
	var keywords []string
	if strings.TrimSpace(raw) == "" {
		return keywords, nil
	}
	if err := json.Unmarshal([]byte(raw), &keywords); err != nil {
		return nil, gerror.Wrap(err, "解析 OCR 屏蔽关键字失败")
	}
	return keywords, nil
}

func (p *Processor) downloadSourceImage(ctx context.Context, source entity.DataSources, downloader *imaging.Downloader, raw string) ([]byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if parsed.IsAbs() {
		return downloader.Download(ctx, parsed.String())
	}
	base, err := url.Parse(source.ImageBaseUrl)
	if strings.TrimSpace(source.ImageBaseUrl) == "" {
		base, err = url.Parse(source.BaseUrl)
	}
	if err != nil {
		return nil, err
	}
	return downloader.Download(ctx, base.ResolveReference(parsed).String())
}

func (p *Processor) failSourceImage(ctx context.Context, id string, cause error) {
	_, _ = dao.SourceNoteImages.Ctx(ctx).Where(dao.SourceNoteImages.Columns().Id, id).
		Data(do.SourceNoteImages{DownloadStatus: "failed", FilterReason: cause.Error()}).Update()
}

func (p *Processor) markSourcePrepareError(ctx context.Context, taskID string, cause error) error {
	_, _ = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, taskID).
		Data(do.ChannelScanTasks{Status: "failed", LastError: cause.Error()}).Update()
	return cause
}
