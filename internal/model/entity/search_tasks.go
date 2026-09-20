// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// SearchTasks is the golang structure for table search_tasks.
type SearchTasks struct {
	Id                 string    `json:"id"                 orm:"id"                   ` //
	SourceType         string    `json:"sourceType"         orm:"source_type"          ` //
	SourceTaskId       string    `json:"sourceTaskId"       orm:"source_task_id"       ` //
	Title              string    `json:"title"              orm:"title"                ` //
	ConfigId           string    `json:"configId"           orm:"config_id"            ` //
	ConfigSnapshotJson string    `json:"configSnapshotJson" orm:"config_snapshot_json" ` //
	Status             string    `json:"status"             orm:"status"               ` //
	TotalCount         int       `json:"totalCount"         orm:"total_count"          ` //
	FilteredCount      int       `json:"filteredCount"      orm:"filtered_count"       ` //
	SearchCount        int       `json:"searchCount"        orm:"search_count"         ` //
	MatchedCount       int       `json:"matchedCount"       orm:"matched_count"        ` //
	FailedCount        int       `json:"failedCount"        orm:"failed_count"         ` //
	ErrorMessage       string    `json:"errorMessage"       orm:"error_message"        ` //
	CreatedAt          time.Time `json:"createdAt"          orm:"created_at"           ` //
	UpdatedAt          time.Time `json:"updatedAt"          orm:"updated_at"           ` //
}
