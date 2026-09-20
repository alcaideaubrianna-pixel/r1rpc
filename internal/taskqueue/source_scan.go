package taskqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"r1rpc/internal/service/datasource"

	"github.com/hibiken/asynq"
)

func (p *Processor) ProcessSourceScan(ctx context.Context, task *asynq.Task) error {
	var payload sourceScanPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil || strings.TrimSpace(payload.TaskID) == "" {
		return fmt.Errorf("无效数据源扫描任务: %w", asynq.SkipRetry)
	}
	return p.app.DataSources.RunScan(ctx, payload.TaskID)
}

var _ datasource.Enqueuer = (*Enqueuer)(nil)
