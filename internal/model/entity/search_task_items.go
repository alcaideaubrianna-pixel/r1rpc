// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// SearchTaskItems is the golang structure for table search_task_items.
type SearchTaskItems struct {
	Id                   string    `json:"id"                   orm:"id"                      ` //
	SearchTaskId         string    `json:"searchTaskId"         orm:"search_task_id"          ` //
	SourceNoteId         string    `json:"sourceNoteId"         orm:"source_note_id"          ` //
	ExternalId           string    `json:"externalId"           orm:"external_id"             ` //
	Title                string    `json:"title"                orm:"title"                   ` //
	Status               string    `json:"status"               orm:"status"                  ` //
	PreprocessStatus     string    `json:"preprocessStatus"     orm:"preprocess_status"       ` //
	FilterStatus         string    `json:"filterStatus"         orm:"filter_status"           ` //
	FilterReason         string    `json:"filterReason"         orm:"filter_reason"           ` //
	ImageSearchRequestId string    `json:"imageSearchRequestId" orm:"image_search_request_id" ` //
	Matched              int       `json:"matched"              orm:"matched"                 ` //
	BestScore            float64   `json:"bestScore"            orm:"best_score"              ` //
	ErrorMessage         string    `json:"errorMessage"         orm:"error_message"           ` //
	CreatedAt            time.Time `json:"createdAt"            orm:"created_at"              ` //
	UpdatedAt            time.Time `json:"updatedAt"            orm:"updated_at"              ` //
}
