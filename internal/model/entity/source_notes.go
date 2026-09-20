// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// SourceNotes is the golang structure for table source_notes.
type SourceNotes struct {
	Id             string    `json:"id"             orm:"id"               ` //
	DataSourceId   string    `json:"dataSourceId"   orm:"data_source_id"   ` //
	ScanTaskId     string    `json:"scanTaskId"     orm:"scan_task_id"     ` //
	ChannelId      int64     `json:"channelId"      orm:"channel_id"       ` //
	ExternalNoteId string    `json:"externalNoteId" orm:"external_note_id" ` //
	NoteCode       string    `json:"noteCode"       orm:"note_code"        ` //
	Title          string    `json:"title"          orm:"title"            ` //
	PlainText      string    `json:"plainText"      orm:"plain_text"       ` //
	AttributesJson string    `json:"attributesJson" orm:"attributes_json"  ` //
	RawJson        string    `json:"rawJson"        orm:"raw_json"         ` //
	Status         string    `json:"status"         orm:"status"           ` //
	CreatedAt      time.Time `json:"createdAt"      orm:"created_at"       ` //
	UpdatedAt      time.Time `json:"updatedAt"      orm:"updated_at"       ` //
}
