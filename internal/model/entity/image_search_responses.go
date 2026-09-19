// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageSearchResponses is the golang structure for table image_search_responses.
type ImageSearchResponses struct {
	Id             string    `json:"id"             orm:"id"              ` //
	JobId          string    `json:"jobId"          orm:"job_id"          ` //
	RequestId      string    `json:"requestId"      orm:"request_id"      ` //
	NormalizedJson string    `json:"normalizedJson" orm:"normalized_json" ` //
	ItemCount      int       `json:"itemCount"      orm:"item_count"      ` //
	ParseStatus    string    `json:"parseStatus"    orm:"parse_status"    ` //
	ErrorMessage   string    `json:"errorMessage"   orm:"error_message"   ` //
	CreatedAt      time.Time `json:"createdAt"      orm:"created_at"      ` //
	UpdatedAt      time.Time `json:"updatedAt"      orm:"updated_at"      ` //
	RawJson        string    `json:"rawJson"        orm:"raw_json"        ` //
}
