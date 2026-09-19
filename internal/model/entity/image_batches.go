// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageBatches is the golang structure for table image_batches.
type ImageBatches struct {
	Id              string    `json:"id"              orm:"id"                ` //
	Source          string    `json:"source"          orm:"source"            ` //
	ExternalId      string    `json:"externalId"      orm:"external_id"       ` //
	Status          string    `json:"status"          orm:"status"            ` //
	Priority        int       `json:"priority"        orm:"priority"          ` //
	ForceRefresh    int       `json:"forceRefresh"    orm:"force_refresh"     ` //
	TotalCount      int       `json:"totalCount"      orm:"total_count"       ` //
	QueuedCount     int       `json:"queuedCount"     orm:"queued_count"      ` //
	RunningCount    int       `json:"runningCount"    orm:"running_count"     ` //
	CompletedCount  int       `json:"completedCount"  orm:"completed_count"   ` //
	FailedCount     int       `json:"failedCount"     orm:"failed_count"      ` //
	CacheHitCount   int       `json:"cacheHitCount"   orm:"cache_hit_count"   ` //
	RequestedByJson string    `json:"requestedByJson" orm:"requested_by_json" ` //
	CreatedAt       time.Time `json:"createdAt"       orm:"created_at"        ` //
	UpdatedAt       time.Time `json:"updatedAt"       orm:"updated_at"        ` //
	FinishedAt      time.Time `json:"finishedAt"      orm:"finished_at"       ` //
	DeletedAt       time.Time `json:"deletedAt"       orm:"deleted_at"        ` //
}
