package imagetask

import (
	"context"
	"strings"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

func validateSourceAndFiles(source string, fileIDs []string) error {
	if strings.TrimSpace(source) == "" {
		return gerror.NewCode(gcode.CodeInvalidParameter, "任务来源不能为空")
	}
	if len(fileIDs) == 0 || len(fileIDs) > maxBatchFiles {
		return gerror.NewCodef(gcode.CodeInvalidParameter, "图片数量必须在 1 到 %d 之间", maxBatchFiles)
	}
	seen := make(map[string]struct{}, len(fileIDs))
	for _, value := range fileIDs {
		fileID := strings.TrimSpace(value)
		if fileID == "" {
			return gerror.NewCode(gcode.CodeInvalidParameter, "fileId 不能为空")
		}
		if _, exists := seen[fileID]; exists {
			return gerror.NewCode(gcode.CodeInvalidParameter, "同一批次不能包含重复 fileId")
		}
		seen[fileID] = struct{}{}
	}
	return nil
}

func findExistingBatch(ctx context.Context, source, externalID string) (*output.ImageBatch, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil
	}
	columns := dao.ImageBatches.Columns()
	record, err := dao.ImageBatches.Ctx(ctx).
		Where(columns.Source, strings.TrimSpace(source)).
		Where(columns.ExternalId, externalID).
		One()
	if err != nil {
		return nil, gerror.Wrap(err, "查询幂等批次失败")
	}
	if record.IsEmpty() {
		return nil, nil
	}
	return &output.ImageBatch{
		ID:      record[columns.Id].String(),
		Status:  record[columns.Status].String(),
		Total:   record[columns.TotalCount].Int(),
		Existed: true,
	}, nil
}

func findExistingJob(ctx context.Context, source, externalID string) (*output.ImageJob, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil
	}
	columns := dao.ImageJobs.Columns()
	record, err := dao.ImageJobs.Ctx(ctx).
		Where(columns.Source, strings.TrimSpace(source)).
		Where(columns.ExternalId, externalID).One()
	if err != nil {
		return nil, gerror.Wrap(err, "查询幂等图片任务失败")
	}
	if record.IsEmpty() {
		return nil, nil
	}
	return &output.ImageJob{
		ID: record[columns.Id].String(), BatchID: record[columns.BatchId].String(),
		Status: record[columns.Status].String(), CacheHit: record[columns.Status].String() == "cache_hit",
	}, nil
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
