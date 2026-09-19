package image_search

import (
	"time"

	"r1rpc/api/image_search/v1"
	"r1rpc/internal/model/output"
)

func toRequestSummary(item output.ImageSearchRequest) v1.RequestSummary {
	createdAt := ""
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.Format(time.RFC3339)
	}
	return v1.RequestSummary{
		ID: item.ID, ExternalID: item.ExternalID, Status: item.Status,
		PipelineName: item.PipelineName, PipelineVersion: item.PipelineVersion,
		GroupCount: item.GroupCount, MatchedCount: item.MatchedCount,
		NotMatchedCount: item.NotMatchedCount, RunningCount: item.RunningCount,
		FailedCount: item.FailedCount, CreatedAt: createdAt,
	}
}
