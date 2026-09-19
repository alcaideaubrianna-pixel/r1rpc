package taskqueue

import (
	"context"
	"time"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
)

func (p *Processor) finalizeAnalysis(ctx context.Context, job entity.ImageJobs, item entity.ImageSearchItems, best bestCandidate) error {
	itemStatus := "not_matched"
	if best.Matched {
		itemStatus = "matched"
	}
	if _, err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, job.Id).Data(do.ImageJobs{
		Status: "completed", Stage: "completed", ErrorCode: "", ErrorMessage: "", FinishedAt: time.Now(),
	}).Update(); err != nil {
		return gerror.Wrap(err, "完成图片分析任务失败")
	}
	if _, err := dao.ImageSearchItems.Ctx(ctx).Where(dao.ImageSearchItems.Columns().Id, item.Id).Data(do.ImageSearchItems{
		Status: itemStatus, ErrorCode: "", ErrorMessage: "", FinishedAt: time.Now(),
	}).Update(); err != nil {
		return gerror.Wrap(err, "完成图片检索项失败")
	}
	if best.Matched {
		if err := p.claimGroupMatch(ctx, item, best); err != nil {
			return err
		}
	}
	return p.refreshDecisionGroup(ctx, item.GroupId)
}

func (p *Processor) claimGroupMatch(ctx context.Context, item entity.ImageSearchItems, best bestCandidate) error {
	return dao.ImageSearchGroups.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		columns := dao.ImageSearchGroups.Columns()
		result, err := tx.Model(dao.ImageSearchGroups.Table()).Ctx(ctx).
			Where(columns.Id, item.GroupId).
			WhereNotIn(columns.Status, []string{"matched", "cancelled"}).
			Data(do.ImageSearchGroups{
				Status: "matched", BestMatchId: best.ID, BestScore: best.Score,
				MatchedAt: time.Now(), FinishedAt: time.Now(),
			}).Update()
		if err != nil {
			return gerror.Wrap(err, "确认图片组命中失败")
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return nil
		}
		itemColumns := dao.ImageSearchItems.Columns()
		var siblings []entity.ImageSearchItems
		if err := tx.Model(dao.ImageSearchItems.Table()).Ctx(ctx).
			Where(itemColumns.GroupId, item.GroupId).WhereNot(itemColumns.Id, item.Id).
			WhereIn(itemColumns.Status, []string{"created", "queued", "retry_wait", "running", "search_completed", "analyzing"}).
			Scan(&siblings); err != nil {
			return gerror.Wrap(err, "读取待取消同组任务失败")
		}
		for _, sibling := range siblings {
			if _, err := tx.Model(dao.ImageSearchItems.Table()).Ctx(ctx).
				Where(itemColumns.Id, sibling.Id).Data(do.ImageSearchItems{Status: "cancelled"}).Update(); err != nil {
				return err
			}
			if sibling.JobId != "" {
				jobColumns := dao.ImageJobs.Columns()
				if _, err := tx.Model(dao.ImageJobs.Table()).Ctx(ctx).Where(jobColumns.Id, sibling.JobId).
					WhereNotIn(jobColumns.Status, []string{"completed", "failed", "cancelled"}).
					Data(do.ImageJobs{Status: "cancelled", Stage: "cancelled"}).Update(); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (p *Processor) failAnalysis(ctx context.Context, jobID, code string, cause error) error {
	if _, err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, jobID).Data(do.ImageJobs{
		Status: "failed", Stage: "failed", ErrorCode: code, ErrorMessage: cause.Error(), FinishedAt: time.Now(),
	}).Update(); err != nil {
		return err
	}
	if _, err := dao.ImageSearchResponses.Ctx(ctx).Where(dao.ImageSearchResponses.Columns().JobId, jobID).
		Data(do.ImageSearchResponses{ParseStatus: "failed", ErrorMessage: cause.Error()}).Update(); err != nil {
		return err
	}
	itemColumns := dao.ImageSearchItems.Columns()
	var item entity.ImageSearchItems
	if err := dao.ImageSearchItems.Ctx(ctx).Where(itemColumns.JobId, jobID).Scan(&item); err != nil {
		return err
	}
	if _, err := dao.ImageSearchItems.Ctx(ctx).Where(itemColumns.Id, item.Id).Data(do.ImageSearchItems{
		Status: "failed", ErrorCode: code, ErrorMessage: cause.Error(), FinishedAt: time.Now(),
	}).Update(); err != nil {
		return err
	}
	return p.refreshDecisionGroup(ctx, item.GroupId)
}

func (p *Processor) refreshDecisionGroup(ctx context.Context, groupID string) error {
	itemColumns := dao.ImageSearchItems.Columns()
	counts := make(map[string]int, 5)
	statusSets := map[string][]string{
		"active":    {"created", "queued", "retry_wait", "running", "search_completed", "analyzing"},
		"matched":   {"matched"},
		"completed": {"matched", "not_matched"},
		"failed":    {"failed"},
		"cancelled": {"cancelled"},
	}
	for name, statuses := range statusSets {
		count, err := dao.ImageSearchItems.Ctx(ctx).Where(itemColumns.GroupId, groupID).
			WhereIn(itemColumns.Status, statuses).Count()
		if err != nil {
			return err
		}
		counts[name] = count
	}
	groupColumns := dao.ImageSearchGroups.Columns()
	var group entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Where(groupColumns.Id, groupID).Scan(&group); err != nil {
		return err
	}
	status := group.Status
	if counts["matched"] > 0 {
		status = "matched"
	} else if counts["active"] == 0 {
		status = "not_matched"
		if counts["completed"] == 0 && counts["failed"] > 0 {
			status = "failed"
		}
	}
	_, err := dao.ImageSearchGroups.Ctx(ctx).Where(groupColumns.Id, groupID).Data(do.ImageSearchGroups{
		Status: status, QueuedCount: 0, RunningCount: counts["active"],
		CompletedCount: counts["completed"], FailedCount: counts["failed"], CancelledCount: counts["cancelled"],
	}).Update()
	if err != nil {
		return err
	}
	return p.refreshDecisionRequest(ctx, group.RequestId)
}

func (p *Processor) refreshDecisionRequest(ctx context.Context, requestID string) error {
	columns := dao.ImageSearchGroups.Columns()
	var groups []entity.ImageSearchGroups
	if err := dao.ImageSearchGroups.Ctx(ctx).Where(columns.RequestId, requestID).Scan(&groups); err != nil {
		return err
	}
	var matched, notMatched, failed, active int
	for _, group := range groups {
		switch group.Status {
		case "matched":
			matched++
		case "not_matched":
			notMatched++
		case "failed":
			failed++
		default:
			active++
		}
	}
	status := "running"
	if active == 0 && len(groups) > 0 {
		status = "completed"
		if failed > 0 {
			status = "partial_failed"
		}
	}
	_, err := dao.ImageSearchRequests.Ctx(ctx).Where(dao.ImageSearchRequests.Columns().Id, requestID).
		Data(do.ImageSearchRequests{
			Status: status, MatchedCount: matched, NotMatchedCount: notMatched,
			RunningCount: active, FailedCount: failed,
		}).Update()
	return err
}
