// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// SourceScanTaskNotes is the golang structure for table source_scan_task_notes.
type SourceScanTaskNotes struct {
	Id         string    `json:"id"         orm:"id"           ` //
	ScanTaskId string    `json:"scanTaskId" orm:"scan_task_id" ` //
	NoteId     string    `json:"noteId"     orm:"note_id"      ` //
	CreatedAt  time.Time `json:"createdAt"  orm:"created_at"   ` //
}
