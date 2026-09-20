package taskqueue

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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
	"github.com/gogf/gf/v2/util/guid"
	"github.com/hibiken/asynq"
)

func (p *Processor) ProcessSourcePrepare(ctx context.Context, task *asynq.Task) error {
	var payload sourceScanPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil || strings.TrimSpace(payload.TaskID) == "" {
		return fmt.Errorf("无效来源图片准备任务: %w", asynq.SkipRetry)
	}
	return p.prepareSourceSearch(ctx, payload.TaskID)
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
	_, err = dao.SearchTasks.Ctx(ctx).Data(do.SearchTasks{Id: searchTaskID, SourceType: "channel_scan", SourceTaskId: searchTaskID, Title: scan.ChannelTitle, ConfigId: config.Id, ConfigSnapshotJson: snapshot, Status: "preparing"}).InsertIgnore()
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	created := 0
	for _, relation := range relations {
		if err = p.createSourceSearchItem(ctx, scan, config, searchTaskID, relation.NoteId); err != nil {
			return p.markSourcePrepareError(ctx, scan.Id, err)
		}
		created++
	}
	_, _ = dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, searchTaskID).Data(do.SearchTasks{Status: "running", TotalCount: created, SearchCount: created, ErrorMessage: ""}).Update()
	status := "completed"
	if scan.Mode == "continuous" {
		status = "waiting"
	}
	_, err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, scan.Id).
		Data(do.ChannelScanTasks{Status: status, SearchTaskId: searchTaskID, LastError: ""}).Update()
	return err
}

func loadScanRelations(ctx context.Context, scan entity.ChannelScanTasks) ([]entity.SourceScanTaskNotes, error) {
	relationColumns := dao.SourceScanTaskNotes.Columns()
	var relations []entity.SourceScanTaskNotes
	if err := dao.SourceScanTaskNotes.Ctx(ctx).Where(relationColumns.ScanTaskId, scan.Id).Scan(&relations); err != nil {
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

func (p *Processor) createSourceSearchItem(ctx context.Context, scan entity.ChannelScanTasks, config entity.SearchConfigs, searchTaskID, noteID string) error {
	var note entity.SourceNotes
	if err := dao.SourceNotes.Ctx(ctx).Where(dao.SourceNotes.Columns().Id, noteID).Scan(&note); err != nil {
		return err
	}
	itemID := guid.S()
	_, err := dao.SearchTaskItems.Ctx(ctx).Data(do.SearchTaskItems{Id: itemID, SearchTaskId: searchTaskID, SourceNoteId: note.Id, ExternalId: note.ExternalNoteId, Title: note.Title, Status: "preparing", PreprocessStatus: "running", FilterStatus: "pending"}).InsertIgnore()
	if err != nil {
		return err
	}
	var item entity.SearchTaskItems
	itemColumns := dao.SearchTaskItems.Columns()
	if err = dao.SearchTaskItems.Ctx(ctx).Where(itemColumns.SearchTaskId, searchTaskID).Where(itemColumns.SourceNoteId, note.Id).Scan(&item); err != nil {
		return err
	}
	itemID = item.Id
	if item.ImageSearchRequestId != "" {
		return nil
	}
	prepared, err := p.prepareNoteImages(ctx, scan.DataSourceId, note.Id, config)
	if err != nil {
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", PreprocessStatus: "failed", ErrorMessage: err.Error()}).Update()
		return nil
	}
	if len(prepared.Items) == 0 {
		if prepared.Blocked > 0 && prepared.Failed == 0 {
			_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{
				Status: "filtered", PreprocessStatus: "completed", FilterStatus: "blocked", FilterReason: "全部图片命中 OCR 屏蔽关键字",
			}).Update()
			return nil
		}
		message := "资料没有可用图片"
		if prepared.Failed > 0 {
			message = fmt.Sprintf("%d 张图片下载或 OCR 失败", prepared.Failed)
		}
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", PreprocessStatus: "failed", ErrorMessage: message}).Update()
		return nil
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
	})
	if err != nil {
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", ErrorMessage: err.Error()}).Update()
		return nil
	}
	filterReason := ""
	if prepared.Blocked > 0 {
		filterReason = fmt.Sprintf("OCR 已屏蔽 %d 张图片", prepared.Blocked)
	}
	_, err = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "running", PreprocessStatus: "completed", FilterStatus: "passed", FilterReason: filterReason, ImageSearchRequestId: request.ID, ErrorMessage: ""}).Update()
	return err
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
	if err := dao.SourceNoteImages.Ctx(ctx).Where(columns.NoteId, noteID).OrderAsc(columns.ImageIndex).Scan(&records); err != nil {
		return preparedImages{}, err
	}
	result := preparedImages{Items: make([]input.ImageSearchItem, 0, len(records))}
	keywords, err := searchOCRKeywords(config.OcrKeywordsJson)
	if err != nil {
		return result, err
	}
	processor := ocr.NewTesseract()
	for _, record := range records {
		fileID := record.FileId
		if fileID == "" {
			_, _ = dao.SourceNoteImages.Ctx(ctx).Where(columns.Id, record.Id).
				Data(do.SourceNoteImages{DownloadStatus: "downloading", FilterReason: ""}).Update()
			assetID, parseErr := strconv.ParseInt(record.ExternalAssetId, 10, 64)
			var data []byte
			var err error
			parsedURL, _ := url.Parse(strings.TrimSpace(record.SourceUrl))
			if parsedURL != nil && parsedURL.IsAbs() && parsedURL.Scheme == "https" {
				data, err = imaging.NewDownloader().Download(ctx, record.SourceUrl)
			} else if parseErr == nil && assetID > 0 {
				data, err = p.app.DataSources.DownloadAsset(ctx, sourceID, assetID)
			} else {
				data, err = p.downloadSourceImage(ctx, sourceID, record.SourceUrl)
			}
			if err != nil {
				p.failSourceImage(ctx, record.Id, err)
				result.Failed++
				continue
			}
			stored, err := p.app.Files.SaveBytes(ctx, data, "source-image.jpg", 0)
			if err != nil {
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
		if config.OcrEnabled == 1 {
			blocked, recognizeErr := p.preprocessSourceImage(ctx, processor, record, fileID, keywords, config.OcrMatchMode)
			if recognizeErr != nil {
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

func (p *Processor) downloadSourceImage(ctx context.Context, sourceID, raw string) ([]byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if parsed.IsAbs() {
		return imaging.NewDownloader().Download(ctx, parsed.String())
	}
	var source entity.DataSources
	if err = dao.DataSources.Ctx(ctx).Where(dao.DataSources.Columns().Id, sourceID).Scan(&source); err != nil {
		return nil, err
	}
	base, err := url.Parse(source.BaseUrl)
	if err != nil {
		return nil, err
	}
	downloader, err := imaging.NewSourceDownloader(source.BaseUrl)
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
