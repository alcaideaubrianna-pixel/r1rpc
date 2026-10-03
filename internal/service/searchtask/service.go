package searchtask

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/guid"
)

type Service struct{}

type IndependentItem struct {
	ID                   string    `json:"id"`
	SearchTaskID         string    `json:"searchTaskId"`
	SourceNoteID         string    `json:"sourceNoteId"`
	ExternalID           string    `json:"externalId"`
	PlainText            string    `json:"plainText"`
	Status               string    `json:"status"`
	PreprocessStatus     string    `json:"preprocessStatus"`
	FilterStatus         string    `json:"filterStatus"`
	FilterReason         string    `json:"filterReason"`
	ImageSearchRequestID string    `json:"imageSearchRequestId"`
	ImageTotal           int       `json:"imageTotal"`
	ImageCompleted       int       `json:"imageCompleted"`
	ImageRunning         int       `json:"imageRunning"`
	ImageQueued          int       `json:"imageQueued"`
	ImageFailed          int       `json:"imageFailed"`
	ProgressPercent      int       `json:"progressPercent"`
	Matched              int       `json:"matched"`
	BestScore            float64   `json:"bestScore"`
	ErrorMessage         string    `json:"errorMessage"`
	DurationMS           int64     `json:"durationMs"`
	TaskTitle            string    `json:"taskTitle"`
	SourceType           string    `json:"sourceType"`
	ChannelID            int64     `json:"channelId"`
	ChannelTitle         string    `json:"channelTitle"`
	DataSourceID         string    `json:"dataSourceId"`
	DataSourceName       string    `json:"dataSourceName"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type IndependentItemPage struct {
	Items      []IndependentItem `json:"items"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	Total      int64             `json:"total"`
	TotalPages int               `json:"totalPages"`
}
type ConfigInput struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	OCREnabled       bool     `json:"ocrEnabled"`
	OCRKeywords      []string `json:"ocrKeywords"`
	OCRMatchMode     string   `json:"ocrMatchMode"`
	ScoreThreshold   float64  `json:"scoreThreshold"`
	MaxPHashDistance int      `json:"maxPHashDistance"`
	MaxDHashDistance int      `json:"maxDHashDistance"`
	MaxAHashDistance int      `json:"maxAHashDistance"`
	MaxCandidates    int      `json:"maxCandidates"`
	PipelineName     string   `json:"pipelineName"`
	Enabled          bool     `json:"enabled"`
}
type Config struct {
	entity.SearchConfigs
	OCRKeywords []string `json:"ocrKeywords"`
}
type Task struct {
	entity.SearchTasks
	CompletedCount  int `json:"completedCount"`
	ProgressPercent int `json:"progressPercent"`
}
type TaskPage struct {
	Items          []Task `json:"items"`
	Page, PageSize int
	Total          int64 `json:"total"`
	TotalPages     int   `json:"totalPages"`
}
type ItemPage struct {
	Items          []entity.SearchTaskItems `json:"items"`
	Task           entity.SearchTasks       `json:"task"`
	Page, PageSize int
	Total          int64 `json:"total"`
	TotalPages     int   `json:"totalPages"`
}

func New() *Service { return &Service{} }

func (s *Service) ListConfigs(ctx context.Context) ([]Config, error) {
	var rows []entity.SearchConfigs
	if err := dao.SearchConfigs.Ctx(ctx).OrderDesc(dao.SearchConfigs.Columns().UpdatedAt).Scan(&rows); err != nil {
		return nil, err
	}
	items := make([]Config, 0, len(rows))
	for _, row := range rows {
		var keywords []string
		_ = json.Unmarshal([]byte(row.OcrKeywordsJson), &keywords)
		items = append(items, Config{SearchConfigs: row, OCRKeywords: keywords})
	}
	return items, nil
}

func (s *Service) SaveConfig(ctx context.Context, in ConfigInput) (*Config, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, gerror.New("配置名称不能为空")
	}
	if in.OCRMatchMode != "contains_all" {
		in.OCRMatchMode = "contains_any"
	}
	if in.ScoreThreshold <= 0 || in.ScoreThreshold > 1 {
		in.ScoreThreshold = 0.82
	}
	if in.MaxPHashDistance < 0 || in.MaxPHashDistance > 64 {
		in.MaxPHashDistance = 12
	}
	if in.MaxDHashDistance < 0 || in.MaxDHashDistance > 64 {
		in.MaxDHashDistance = 16
	}
	if in.MaxAHashDistance < 0 || in.MaxAHashDistance > 64 {
		in.MaxAHashDistance = 16
	}
	if in.MaxCandidates < 1 || in.MaxCandidates > 100 {
		in.MaxCandidates = 20
	}
	if strings.TrimSpace(in.PipelineName) == "" {
		in.PipelineName = "hash-v1"
	}
	keywords, _ := json.Marshal(cleanKeywords(in.OCRKeywords))
	data := do.SearchConfigs{Name: strings.TrimSpace(in.Name), OcrEnabled: in.OCREnabled, OcrKeywordsJson: string(keywords), OcrMatchMode: in.OCRMatchMode, ScoreThreshold: in.ScoreThreshold, MaxPhashDistance: in.MaxPHashDistance, MaxDhashDistance: in.MaxDHashDistance, MaxAhashDistance: in.MaxAHashDistance, MaxCandidates: in.MaxCandidates, PipelineName: in.PipelineName, Enabled: in.Enabled}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = guid.S()
		data.Id = id
		data.Version = 1
		if _, err := dao.SearchConfigs.Ctx(ctx).Data(data).Insert(); err != nil {
			return nil, err
		}
	} else {
		data.Version = gdb.Raw("version + 1")
		if _, err := dao.SearchConfigs.Ctx(ctx).Where(dao.SearchConfigs.Columns().Id, id).Data(data).Update(); err != nil {
			return nil, err
		}
	}
	items, err := s.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Id == id {
			return &item, nil
		}
	}
	return nil, gerror.New("搜索配置保存后读取失败")
}

func (s *Service) ListTasks(ctx context.Context, page, pageSize int) (*TaskPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	model := dao.SearchTasks.Ctx(ctx)
	total, err := model.Count()
	if err != nil {
		return nil, err
	}
	var rows []entity.SearchTasks
	if err = model.OrderDesc(dao.SearchTasks.Columns().CreatedAt).Page(page, pageSize).Scan(&rows); err != nil {
		return nil, err
	}
	for i := range rows {
		if err = s.refreshTask(ctx, rows[i].Id); err != nil {
			return nil, err
		}
	}
	if len(rows) > 0 {
		_ = model.OrderDesc(dao.SearchTasks.Columns().CreatedAt).Page(page, pageSize).Scan(&rows)
	}
	items := make([]Task, 0, len(rows))
	for _, row := range rows {
		completed, countErr := s.completedCount(ctx, row.Id)
		if countErr != nil {
			return nil, countErr
		}
		percent := 0
		if row.TotalCount > 0 {
			percent = completed * 100 / row.TotalCount
		}
		items = append(items, Task{SearchTasks: row, CompletedCount: completed, ProgressPercent: percent})
	}
	return &TaskPage{Items: items, Page: page, PageSize: pageSize, Total: int64(total), TotalPages: (total + pageSize - 1) / pageSize}, nil
}
func (s *Service) ListItems(ctx context.Context, taskID string, page, pageSize int) (*ItemPage, error) {
	if err := s.refreshTask(ctx, taskID); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	model := dao.SearchTaskItems.Ctx(ctx).Where(dao.SearchTaskItems.Columns().SearchTaskId, taskID)
	total, err := model.Count()
	if err != nil {
		return nil, err
	}
	var rows []entity.SearchTaskItems
	if err = model.OrderAsc(dao.SearchTaskItems.Columns().CreatedAt).Page(page, pageSize).Scan(&rows); err != nil {
		return nil, err
	}
	var task entity.SearchTasks
	if err = dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, taskID).Scan(&task); err != nil {
		return nil, err
	}
	return &ItemPage{Items: rows, Task: task, Page: page, PageSize: pageSize, Total: int64(total), TotalPages: (total + pageSize - 1) / pageSize}, nil
}

func (s *Service) ListIndependentItems(ctx context.Context, page, pageSize int, query string) (*IndependentItemPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	columns := dao.SearchTaskItems.Columns()
	model := dao.SearchTaskItems.Ctx(ctx)
	query = strings.TrimSpace(query)
	if query != "" {
		noteIDs, queryErr := findNoteIDs(ctx, query)
		if queryErr != nil {
			return nil, queryErr
		}
		model = model.WhereLike(columns.ExternalId, "%"+query+"%")
		if len(noteIDs) > 0 {
			model = model.WhereOrIn(columns.SourceNoteId, noteIDs)
		}
	}
	total, err := model.Count()
	if err != nil {
		return nil, err
	}
	var rows []entity.SearchTaskItems
	if err = model.OrderDesc(columns.CreatedAt).Page(page, pageSize).Scan(&rows); err != nil {
		return nil, err
	}
	for _, taskID := range uniqueTaskIDs(rows) {
		if err = s.refreshTask(ctx, taskID); err != nil {
			return nil, err
		}
	}
	if len(rows) > 0 {
		if err = model.OrderDesc(columns.CreatedAt).Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
	}
	notes, err := loadNotes(ctx, rows)
	if err != nil {
		return nil, err
	}
	tasks, err := loadTasks(ctx, rows)
	if err != nil {
		return nil, err
	}
	channels, dataSources, err := loadSources(ctx, notes)
	if err != nil {
		return nil, err
	}
	items := make([]IndependentItem, 0, len(rows))
	for _, row := range rows {
		item := IndependentItem{
			ID: row.Id, SearchTaskID: row.SearchTaskId, SourceNoteID: row.SourceNoteId,
			ExternalID: row.ExternalId, Status: row.Status, PreprocessStatus: row.PreprocessStatus,
			FilterStatus: row.FilterStatus, FilterReason: row.FilterReason,
			ImageSearchRequestID: row.ImageSearchRequestId, Matched: row.Matched,
			BestScore: row.BestScore, ErrorMessage: row.ErrorMessage,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
		if note, ok := notes[row.SourceNoteId]; ok {
			item.PlainText = note.PlainText
			item.ChannelID = note.ChannelId
			item.DataSourceID = note.DataSourceId
			if item.ExternalID == "" {
				item.ExternalID = note.ExternalNoteId
			}
			if channel, exists := channels[sourceChannelKey(note.DataSourceId, note.ChannelId)]; exists {
				item.ChannelTitle = channel.Title
			}
			if source, exists := dataSources[note.DataSourceId]; exists {
				item.DataSourceName = source.Name
			}
		}
		if task, ok := tasks[row.SearchTaskId]; ok {
			item.TaskTitle = task.Title
			item.SourceType = task.SourceType
		}
		if row.ImageSearchRequestId != "" {
			if err = fillImageProgress(ctx, &item); err != nil {
				return nil, err
			}
		}
		finishedAt := item.UpdatedAt
		if !isTerminalSearchStatus(item.Status) {
			finishedAt = time.Now()
		}
		item.DurationMS = finishedAt.Sub(item.CreatedAt).Milliseconds()
		items = append(items, item)
	}
	return &IndependentItemPage{Items: items, Page: page, PageSize: pageSize, Total: int64(total), TotalPages: (total + pageSize - 1) / pageSize}, nil
}

func findNoteIDs(ctx context.Context, query string) ([]string, error) {
	columns := dao.SourceNotes.Columns()
	var notes []entity.SourceNotes
	model := dao.SourceNotes.Ctx(ctx).Fields(columns.Id).WhereLike(columns.ExternalNoteId, "%"+query+"%").
		WhereOrLike(columns.NoteCode, "%"+query+"%").WhereOrLike(columns.Id, "%"+query+"%")
	if err := model.Scan(&notes); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(notes))
	for _, note := range notes {
		ids = append(ids, note.Id)
	}
	return ids, nil
}

func uniqueTaskIDs(rows []entity.SearchTaskItems) []string {
	seen := make(map[string]struct{}, len(rows))
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.SearchTaskId]; ok {
			continue
		}
		seen[row.SearchTaskId] = struct{}{}
		result = append(result, row.SearchTaskId)
	}
	return result
}

func loadNotes(ctx context.Context, rows []entity.SearchTaskItems) (map[string]entity.SourceNotes, error) {
	result := make(map[string]entity.SourceNotes, len(rows))
	if len(rows) == 0 {
		return result, nil
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.SourceNoteId)
	}
	var notes []entity.SourceNotes
	if err := dao.SourceNotes.Ctx(ctx).WhereIn(dao.SourceNotes.Columns().Id, ids).Scan(&notes); err != nil {
		return nil, err
	}
	for _, note := range notes {
		result[note.Id] = note
	}
	return result, nil
}

func loadTasks(ctx context.Context, rows []entity.SearchTaskItems) (map[string]entity.SearchTasks, error) {
	result := make(map[string]entity.SearchTasks)
	ids := uniqueTaskIDs(rows)
	if len(ids) == 0 {
		return result, nil
	}
	var tasks []entity.SearchTasks
	if err := dao.SearchTasks.Ctx(ctx).WhereIn(dao.SearchTasks.Columns().Id, ids).Scan(&tasks); err != nil {
		return nil, err
	}
	for _, task := range tasks {
		result[task.Id] = task
	}
	return result, nil
}

func loadSources(ctx context.Context, notes map[string]entity.SourceNotes) (map[string]entity.SourceChannels, map[string]entity.DataSources, error) {
	channels := make(map[string]entity.SourceChannels)
	dataSources := make(map[string]entity.DataSources)
	if len(notes) == 0 {
		return channels, dataSources, nil
	}
	sourceIDs := make([]string, 0, len(notes))
	channelIDs := make([]int64, 0, len(notes))
	seenSources := make(map[string]struct{})
	seenChannels := make(map[int64]struct{})
	for _, note := range notes {
		if _, ok := seenSources[note.DataSourceId]; !ok {
			seenSources[note.DataSourceId] = struct{}{}
			sourceIDs = append(sourceIDs, note.DataSourceId)
		}
		if _, ok := seenChannels[note.ChannelId]; !ok {
			seenChannels[note.ChannelId] = struct{}{}
			channelIDs = append(channelIDs, note.ChannelId)
		}
	}
	var channelRows []entity.SourceChannels
	if err := dao.SourceChannels.Ctx(ctx).WhereIn(dao.SourceChannels.Columns().DataSourceId, sourceIDs).
		WhereIn(dao.SourceChannels.Columns().ChannelId, channelIDs).Scan(&channelRows); err != nil {
		return nil, nil, err
	}
	for _, channel := range channelRows {
		channels[sourceChannelKey(channel.DataSourceId, channel.ChannelId)] = channel
	}
	var sourceRows []entity.DataSources
	if err := dao.DataSources.Ctx(ctx).WhereIn(dao.DataSources.Columns().Id, sourceIDs).Scan(&sourceRows); err != nil {
		return nil, nil, err
	}
	for _, source := range sourceRows {
		dataSources[source.Id] = source
	}
	return channels, dataSources, nil
}

func sourceChannelKey(dataSourceID string, channelID int64) string {
	return dataSourceID + ":" + strconv.FormatInt(channelID, 10)
}

func fillImageProgress(ctx context.Context, item *IndependentItem) error {
	groupColumns := dao.ImageSearchGroups.Columns()
	var groups []entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Fields(groupColumns.Id).
		Where(groupColumns.RequestId, item.ImageSearchRequestID).Scan(&groups); err != nil {
		return err
	}
	if len(groups) == 0 {
		return nil
	}
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.Id)
	}
	var images []entity.ImageSearchItems
	imageColumns := dao.ImageSearchItems.Columns()
	if err := dao.ImageSearchItems.Ctx(ctx).WhereIn(imageColumns.GroupId, groupIDs).Scan(&images); err != nil {
		return err
	}
	item.ImageTotal = len(images)
	for _, image := range images {
		switch image.Status {
		case "matched", "not_matched", "search_completed", "completed", "cancelled":
			item.ImageCompleted++
		case "failed":
			item.ImageFailed++
			if item.ErrorMessage == "" {
				item.ErrorMessage = image.ErrorMessage
			}
		case "running", "analyzing":
			item.ImageRunning++
		default:
			item.ImageQueued++
		}
	}
	if item.ImageTotal > 0 {
		item.ProgressPercent = (item.ImageCompleted + item.ImageFailed) * 100 / item.ImageTotal
	}
	return nil
}

func isTerminalSearchStatus(status string) bool {
	switch status {
	case "completed", "partial_failed", "failed", "cancelled", "filtered", "matched", "not_matched":
		return true
	default:
		return false
	}
}
func (s *Service) refreshTask(ctx context.Context, taskID string) error {
	var items []entity.SearchTaskItems
	cols := dao.SearchTaskItems.Columns()
	if err := dao.SearchTaskItems.Ctx(ctx).Where(cols.SearchTaskId, taskID).Scan(&items); err != nil {
		return err
	}
	if len(items) == 0 {
		var task entity.SearchTasks
		if err := dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, taskID).Scan(&task); err != nil {
			return err
		}
		if task.Id != "" && task.Status != "failed" && time.Since(task.CreatedAt) > time.Minute {
			_, err := dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, taskID).Data(do.SearchTasks{
				Status: "failed", ErrorMessage: "准备阶段未生成任何资料，请检查扫描任务和数据源响应",
			}).Update()
			return err
		}
		return nil
	}
	counts := map[string]int{}
	for _, item := range items {
		status := item.Status
		matched := 0
		bestScore := any(gdb.Raw("NULL"))
		if item.ImageSearchRequestId != "" {
			var request entity.ImageSearchRequests
			if err := dao.ImageSearchRequests.Ctx(ctx).Where(dao.ImageSearchRequests.Columns().Id, item.ImageSearchRequestId).Scan(&request); err != nil {
				return err
			}
			status = request.Status
			if request.MatchedCount > 0 {
				matched = 1
				var group entity.ImageSearchGroups
				if err := dao.ImageSearchGroups.Ctx(ctx).Where(dao.ImageSearchGroups.Columns().RequestId, request.Id).
					Where(dao.ImageSearchGroups.Columns().Status, "matched").OrderDesc(dao.ImageSearchGroups.Columns().BestScore).Limit(1).Scan(&group); err != nil {
					return err
				}
				bestScore = group.BestScore
			}
			_, _ = dao.SearchTaskItems.Ctx(ctx).Where(cols.Id, item.Id).Data(do.SearchTaskItems{Status: status, Matched: matched, BestScore: bestScore}).Update()
		}
		counts[status]++
		if item.FilterStatus == "blocked" {
			counts["filtered_items"]++
		}
		if matched == 1 {
			counts["matched"]++
		}
	}
	failed := counts["failed"] + counts["partial_failed"]
	finished := counts["completed"] + failed + counts["cancelled"] + counts["filtered"]
	status := "running"
	if len(items) > 0 && finished == len(items) {
		status = "completed"
		if failed == len(items) {
			status = "failed"
		} else if failed > 0 {
			status = "partial_failed"
		}
	}
	_, err := dao.SearchTasks.Ctx(ctx).Where(dao.SearchTasks.Columns().Id, taskID).Data(do.SearchTasks{Status: status, TotalCount: len(items), SearchCount: len(items) - counts["filtered_items"], FilteredCount: counts["filtered_items"], MatchedCount: counts["matched"], FailedCount: failed}).Update()
	return err
}

func (s *Service) completedCount(ctx context.Context, taskID string) (int, error) {
	columns := dao.SearchTaskItems.Columns()
	return dao.SearchTaskItems.Ctx(ctx).Where(columns.SearchTaskId, taskID).
		WhereIn(columns.Status, []string{"completed", "partial_failed", "failed", "cancelled", "filtered"}).Count()
}
func cleanKeywords(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
