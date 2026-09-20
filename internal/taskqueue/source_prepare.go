package taskqueue

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"r1rpc/internal/dao"
	"r1rpc/internal/imaging"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/model/input"

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
	_, err = dao.SearchTasks.Ctx(ctx).Data(do.SearchTasks{Id: searchTaskID, SourceType: "channel_scan", SourceTaskId: searchTaskID, Title: scan.ChannelTitle, ConfigId: config.Id, ConfigSnapshotJson: snapshot, Status: "preparing"}).InsertIgnore()
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	relations, err := loadScanRelations(ctx, scan.Id)
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
	if created == 0 {
		return p.markSourcePrepareError(ctx, scan.Id, gerror.New("扫描结果没有资料"))
	}
	_, _ = dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, searchTaskID).Data(do.SearchTasks{Status: "running", TotalCount: created, SearchCount: created}).Update()
	status := "completed"
	if scan.Mode == "continuous" {
		status = "waiting"
	}
	_, err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, scan.Id).
		Data(do.ChannelScanTasks{Status: status, SearchTaskId: searchTaskID, LastError: ""}).Update()
	return err
}

func loadScanRelations(ctx context.Context, scanID string) ([]entity.SourceScanTaskNotes, error) {
	relationColumns := dao.SourceScanTaskNotes.Columns()
	var relations []entity.SourceScanTaskNotes
	if err := dao.SourceScanTaskNotes.Ctx(ctx).Where(relationColumns.ScanTaskId, scanID).Scan(&relations); err != nil {
		return nil, err
	}
	if len(relations) == 0 {
		var legacyNotes []entity.SourceNotes
		if err := dao.SourceNotes.Ctx(ctx).Where(dao.SourceNotes.Columns().ScanTaskId, scanID).Scan(&legacyNotes); err != nil {
			return nil, err
		}
		for _, note := range legacyNotes {
			relations = append(relations, entity.SourceScanTaskNotes{ScanTaskId: scanID, NoteId: note.Id})
			_, _ = dao.SourceScanTaskNotes.Ctx(ctx).Data(do.SourceScanTaskNotes{Id: guid.S(), ScanTaskId: scanID, NoteId: note.Id}).InsertIgnore()
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
	images, err := p.prepareNoteImages(ctx, scan.DataSourceId, note.Id)
	if err != nil {
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", ErrorMessage: err.Error()}).Update()
		return nil
	}
	if len(images) == 0 {
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", ErrorMessage: "资料没有可用图片"}).Update()
		return nil
	}
	if config.OcrEnabled == 1 {
		message := "OCR 已启用，但 OCR 处理器尚未配置"
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{
			Status: "failed", PreprocessStatus: "pending", FilterStatus: "pending", ErrorMessage: message,
		}).Update()
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
			ExternalID: note.ExternalNoteId, Images: images,
			MatchPolicy: map[string]any{"scoreThreshold": config.ScoreThreshold, "maxPHashDistance": config.MaxPhashDistance, "maxDHashDistance": config.MaxDhashDistance, "maxAHashDistance": config.MaxAhashDistance, "maxCandidates": config.MaxCandidates},
		}},
	})
	if err != nil {
		_, _ = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "failed", ErrorMessage: err.Error()}).Update()
		return nil
	}
	_, err = dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().Id, itemID).Data(do.SearchTaskItems{Status: "running", PreprocessStatus: "completed", FilterStatus: "passed", ImageSearchRequestId: request.ID}).Update()
	return err
}

func sourceSearchTaskID(scan entity.ChannelScanTasks) string {
	cycle := scan.LastSuccessAt.UTC().Format("20060102T150405.000000000")
	sum := sha256.Sum256([]byte(scan.Id + ":" + cycle))
	return fmt.Sprintf("%x", sum[:16])
}

func (p *Processor) prepareNoteImages(ctx context.Context, sourceID, noteID string) ([]input.ImageSearchItem, error) {
	columns := dao.SourceNoteImages.Columns()
	var records []entity.SourceNoteImages
	if err := dao.SourceNoteImages.Ctx(ctx).Where(columns.NoteId, noteID).OrderAsc(columns.ImageIndex).Scan(&records); err != nil {
		return nil, err
	}
	items := make([]input.ImageSearchItem, 0, len(records))
	for _, record := range records {
		fileID := record.FileId
		if fileID == "" {
			resolved, err := p.sourceImageURL(ctx, sourceID, record.SourceUrl)
			if err != nil {
				p.failSourceImage(ctx, record.Id, err)
				continue
			}
			data, err := imaging.NewDownloader().Download(ctx, resolved)
			if err != nil {
				p.failSourceImage(ctx, record.Id, err)
				continue
			}
			stored, err := p.app.Files.SaveBytes(ctx, data, "source-image.jpg", 0)
			if err != nil {
				p.failSourceImage(ctx, record.Id, err)
				continue
			}
			fileID = stored.ID
			_, err = dao.SourceNoteImages.Ctx(ctx).Where(columns.Id, record.Id).Data(do.SourceNoteImages{FileId: stored.ID, Sha256: stored.SHA256, DownloadStatus: "downloaded"}).Update()
			if err != nil {
				return nil, err
			}
		}
		items = append(items, input.ImageSearchItem{FileID: fileID})
	}
	return items, nil
}

func (p *Processor) sourceImageURL(ctx context.Context, sourceID, raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() {
		return parsed.String(), nil
	}
	var source entity.DataSources
	if err = dao.DataSources.Ctx(ctx).Where(dao.DataSources.Columns().Id, sourceID).Scan(&source); err != nil {
		return "", err
	}
	base, err := url.Parse(source.BaseUrl)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
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
