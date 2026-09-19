package imagetask

import (
	"context"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/errors/gerror"
)

// ListBatches 分页查询批量任务，按创建时间和 ID 稳定倒序。
func (s *Service) ListBatches(ctx context.Context, page, pageSize int) (*output.ImageBatchPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	columns := dao.ImageBatches.Columns()
	model := dao.ImageBatches.Ctx(ctx)
	total, err := model.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "统计图片批次失败")
	}
	var entities []entity.ImageBatches
	if err := model.OrderDesc(columns.CreatedAt).OrderDesc(columns.Id).
		Page(page, pageSize).Scan(&entities); err != nil {
		return nil, gerror.Wrap(err, "查询图片批次失败")
	}
	result := &output.ImageBatchPage{Page: page, PageSize: pageSize, Total: int64(total)}
	for _, item := range entities {
		result.Items = append(result.Items, output.ImageBatch{
			ID: item.Id, Status: item.Status, Total: item.TotalCount,
			Queued: item.QueuedCount, Running: item.RunningCount,
			Completed: item.CompletedCount, Failed: item.FailedCount,
			CacheHit: item.CacheHitCount, CreatedAt: item.CreatedAt,
		})
	}
	return result, nil
}
