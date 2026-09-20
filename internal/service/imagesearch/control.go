package imagesearch

import (
	"context"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *Service) SetRequestState(ctx context.Context, requestID, state string) error {
	if _, err := s.GetOne(ctx, requestID); err != nil {
		return err
	}
	if state != "cancelled" && state != "paused" && state != "running" {
		return gerror.New("不支持的任务状态")
	}
	jobStatus := state
	itemStatus := state
	if state == "running" {
		jobStatus, itemStatus = "queued", "queued"
	}
	if _, err := dao.ImageSearchRequests.Ctx(ctx).Where(dao.ImageSearchRequests.Columns().Id, requestID).
		Data(do.ImageSearchRequests{Status: state}).Update(); err != nil {
		return gerror.Wrap(err, "更新图片搜索任务状态失败")
	}
	groupCols := dao.ImageSearchGroups.Columns()
	var groups []struct{ Id string }
	if err := dao.ImageSearchGroups.Ctx(ctx).Fields(groupCols.Id).Where(groupCols.RequestId, requestID).Scan(&groups); err != nil {
		return err
	}
	for _, group := range groups {
		if _, err := dao.ImageSearchGroups.Ctx(ctx).Where(groupCols.Id, group.Id).Data(do.ImageSearchGroups{Status: state}).Update(); err != nil {
			return err
		}
		itemCols := dao.ImageSearchItems.Columns()
		if _, err := dao.ImageSearchItems.Ctx(ctx).Where(itemCols.GroupId, group.Id).
			WhereIn(itemCols.Status, []string{"created", "queued", "retry_wait", "running", "analyzing", "search_completed"}).
			Data(do.ImageSearchItems{Status: itemStatus}).Update(); err != nil {
			return err
		}
	}
	jobCols := dao.ImageJobs.Columns()
	_, err := dao.ImageJobs.Ctx(ctx).WhereIn(jobCols.Id, jobIDsForRequest(ctx, requestID)).
		WhereIn(jobCols.Status, []string{"created", "queued", "retry_wait", "running"}).Data(do.ImageJobs{Status: jobStatus, Stage: state}).Update()
	return err
}

func jobIDsForRequest(ctx context.Context, requestID string) []string {
	var ids []string
	itemCols := dao.ImageSearchItems.Columns()
	_ = dao.ImageSearchItems.Ctx(ctx).Fields(itemCols.JobId).WhereIn(itemCols.GroupId,
		dao.ImageSearchGroups.Ctx(ctx).Fields(dao.ImageSearchGroups.Columns().Id).Where(dao.ImageSearchGroups.Columns().RequestId, requestID)).Scan(&ids)
	return ids
}

func (s *Service) SetPriority(ctx context.Context, requestID string, priority int) error {
	if priority < -100 || priority > 100 {
		return gerror.New("优先级范围必须是 -100 到 100")
	}
	if _, err := s.GetOne(ctx, requestID); err != nil {
		return err
	}
	ids := jobIDsForRequest(ctx, requestID)
	if len(ids) == 0 {
		return nil
	}
	_, err := dao.ImageJobs.Ctx(ctx).WhereIn(dao.ImageJobs.Columns().Id, ids).Data(do.ImageJobs{Priority: priority}).Update()
	return err
}
