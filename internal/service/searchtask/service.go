package searchtask

import (
	"context"
	"encoding/json"
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
