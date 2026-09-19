// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageSearchRequests is the golang structure for table image_search_requests.
type ImageSearchRequests struct {
	Id                 string    `json:"id"                 orm:"id"                   ` //
	Source             string    `json:"source"             orm:"source"               ` //
	ExternalId         string    `json:"externalId"         orm:"external_id"          ` //
	Status             string    `json:"status"             orm:"status"               ` //
	PipelineName       string    `json:"pipelineName"       orm:"pipeline_name"        ` //
	PipelineVersion    string    `json:"pipelineVersion"    orm:"pipeline_version"     ` //
	Priority           int       `json:"priority"           orm:"priority"             ` //
	GroupCount         int       `json:"groupCount"         orm:"group_count"          ` //
	MatchedCount       int       `json:"matchedCount"       orm:"matched_count"        ` //
	NotMatchedCount    int       `json:"notMatchedCount"    orm:"not_matched_count"    ` //
	RunningCount       int       `json:"runningCount"       orm:"running_count"        ` //
	FailedCount        int       `json:"failedCount"        orm:"failed_count"         ` //
	RequestedByUserId  int64     `json:"requestedByUserId"  orm:"requested_by_user_id" ` //
	RequestedBySubject string    `json:"requestedBySubject" orm:"requested_by_subject" ` //
	CallbackUrl        string    `json:"callbackUrl"        orm:"callback_url"         ` //
	CallbackStatus     string    `json:"callbackStatus"     orm:"callback_status"      ` //
	CreatedAt          time.Time `json:"createdAt"          orm:"created_at"           ` //
	UpdatedAt          time.Time `json:"updatedAt"          orm:"updated_at"           ` //
	FinishedAt         time.Time `json:"finishedAt"         orm:"finished_at"          ` //
	DeletedAt          time.Time `json:"deletedAt"          orm:"deleted_at"           ` //
}
