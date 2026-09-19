// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageSearchItems is the golang structure for table image_search_items.
type ImageSearchItems struct {
	Id            string    `json:"id"            orm:"id"              ` //
	GroupId       string    `json:"groupId"       orm:"group_id"        ` //
	SourceAssetId string    `json:"sourceAssetId" orm:"source_asset_id" ` //
	Ordinal       int       `json:"ordinal"       orm:"ordinal"         ` //
	Status        string    `json:"status"        orm:"status"          ` //
	JobId         string    `json:"jobId"         orm:"job_id"          ` //
	ErrorCode     string    `json:"errorCode"     orm:"error_code"      ` //
	ErrorMessage  string    `json:"errorMessage"  orm:"error_message"   ` //
	CreatedAt     time.Time `json:"createdAt"     orm:"created_at"      ` //
	UpdatedAt     time.Time `json:"updatedAt"     orm:"updated_at"      ` //
	FinishedAt    time.Time `json:"finishedAt"    orm:"finished_at"     ` //
	DeletedAt     time.Time `json:"deletedAt"     orm:"deleted_at"      ` //
}
