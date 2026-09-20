package taskqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
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
	groups, err := p.buildSourceGroups(ctx, scan)
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	if len(groups) == 0 {
		return p.markSourcePrepareError(ctx, scan.Id, gerror.New("扫描结果没有可用图片"))
	}
	priority := scan.Priority
	if priority < -10 {
		priority = -10
	}
	if priority > 10 {
		priority = 10
	}
	request, err := p.app.ImageSearch.Create(ctx, input.CreateImageSearchRequest{
		Source: "channel_scan", ExternalID: "channel-scan:" + scan.Id + ":" + strconv.FormatInt(scan.LastSuccessAt.UnixNano(), 10),
		Priority: priority, RequestedBySubject: scan.ChannelTitle, Groups: groups,
	})
	if err != nil {
		return p.markSourcePrepareError(ctx, scan.Id, err)
	}
	status := "completed"
	if scan.Mode == "continuous" {
		status = "waiting"
	}
	_, err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, scan.Id).
		Data(do.ChannelScanTasks{Status: status, ImageSearchRequestId: request.ID, LastError: ""}).Update()
	return err
}

func (p *Processor) buildSourceGroups(ctx context.Context, scan entity.ChannelScanTasks) ([]input.ImageSearchGroup, error) {
	relationColumns := dao.SourceScanTaskNotes.Columns()
	var relations []entity.SourceScanTaskNotes
	if err := dao.SourceScanTaskNotes.Ctx(ctx).Where(relationColumns.ScanTaskId, scan.Id).Scan(&relations); err != nil {
		return nil, err
	}
	if len(relations) == 0 {
		var legacyNotes []entity.SourceNotes
		if err := dao.SourceNotes.Ctx(ctx).Where(dao.SourceNotes.Columns().ScanTaskId, scan.Id).Scan(&legacyNotes); err != nil {
			return nil, err
		}
		for _, note := range legacyNotes {
			relations = append(relations, entity.SourceScanTaskNotes{ScanTaskId: scan.Id, NoteId: note.Id})
			_, _ = dao.SourceScanTaskNotes.Ctx(ctx).Data(do.SourceScanTaskNotes{Id: guid.S(), ScanTaskId: scan.Id, NoteId: note.Id}).InsertIgnore()
		}
	}
	groups := make([]input.ImageSearchGroup, 0, len(relations))
	for _, relation := range relations {
		var note entity.SourceNotes
		if err := dao.SourceNotes.Ctx(ctx).Where(dao.SourceNotes.Columns().Id, relation.NoteId).Scan(&note); err != nil {
			return nil, err
		}
		images, err := p.prepareNoteImages(ctx, scan.DataSourceId, note.Id)
		if err != nil {
			return nil, err
		}
		if len(images) > 0 {
			groups = append(groups, input.ImageSearchGroup{ExternalID: note.ExternalNoteId, Images: images, MatchPolicy: map[string]any{}})
		}
	}
	return groups, nil
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
