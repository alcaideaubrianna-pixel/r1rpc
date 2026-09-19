package taskqueue

import (
	"context"
	"database/sql"
	"errors"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (p *Processor) syncSearchJob(ctx context.Context, jobID, status, errorCode, errorMessage string) error {
	itemColumns := dao.ImageSearchItems.Columns()
	var item entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Where(itemColumns.JobId, jobID).Scan(&item); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return gerror.Wrap(err, "查询图片检索项失败")
	}
	if item.Id == "" {
		return nil
	}
	data := do.ImageSearchItems{Status: status, ErrorCode: errorCode, ErrorMessage: errorMessage}
	if _, err := dao.ImageSearchItems.Ctx(ctx).Where(itemColumns.Id, item.Id).Data(data).Update(); err != nil {
		return gerror.Wrap(err, "更新图片检索项状态失败")
	}
	return p.refreshSearchGroup(ctx, item.GroupId)
}

func (p *Processor) refreshSearchGroup(ctx context.Context, groupID string) error {
	columns := dao.ImageSearchItems.Columns()
	counts := make(map[string]int, 4)
	statusSets := map[string][]string{
		"queued":    {"created", "queued", "retry_wait"},
		"running":   {"running", "analyzing"},
		"completed": {"search_completed"},
		"failed":    {"failed"},
	}
	for name, statuses := range statusSets {
		count, err := dao.ImageSearchItems.Ctx(ctx).Where(columns.GroupId, groupID).
			WhereIn(columns.Status, statuses).Count()
		if err != nil {
			return gerror.Wrap(err, "统计图片检索项状态失败")
		}
		counts[name] = count
	}
	groupStatus := searchGroupStatus(counts)
	groupColumns := dao.ImageSearchGroups.Columns()
	if _, err := dao.ImageSearchGroups.Ctx(ctx).Where(groupColumns.Id, groupID).Data(do.ImageSearchGroups{
		Status: groupStatus, QueuedCount: counts["queued"], RunningCount: counts["running"],
		CompletedCount: counts["completed"], FailedCount: counts["failed"],
	}).Update(); err != nil {
		return gerror.Wrap(err, "更新图片检索组进度失败")
	}
	var group entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Where(groupColumns.Id, groupID).Scan(&group); err != nil {
		return gerror.Wrap(err, "读取图片检索组失败")
	}
	return p.refreshSearchRequest(ctx, group.RequestId)
}

func (p *Processor) refreshSearchRequest(ctx context.Context, requestID string) error {
	columns := dao.ImageSearchGroups.Columns()
	var groups []entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Where(columns.RequestId, requestID).Scan(&groups); err != nil {
		return gerror.Wrap(err, "读取图片检索请求进度失败")
	}
	var (
		active int
		failed int
	)
	for _, group := range groups {
		switch group.Status {
		case "search_completed":
		case "failed", "partial_failed":
			failed++
		default:
			active++
		}
	}
	status := "running"
	if active == 0 && len(groups) > 0 {
		status = "search_completed"
		if failed > 0 {
			status = "partial_failed"
		}
	}
	_, err := dao.ImageSearchRequests.Ctx(ctx).Where(dao.ImageSearchRequests.Columns().Id, requestID).
		Data(do.ImageSearchRequests{Status: status, RunningCount: active, FailedCount: failed}).Update()
	return gerror.Wrap(err, "更新图片检索请求进度失败")
}

func searchGroupStatus(counts map[string]int) string {
	total := counts["queued"] + counts["running"] + counts["completed"] + counts["failed"]
	terminal := counts["completed"] + counts["failed"]
	switch {
	case total > 0 && terminal == total && counts["failed"] == total:
		return "failed"
	case total > 0 && terminal == total && counts["failed"] > 0:
		return "partial_failed"
	case total > 0 && terminal == total:
		return "search_completed"
	case counts["running"] > 0 || counts["completed"] > 0:
		return "running"
	default:
		return "queued"
	}
}
