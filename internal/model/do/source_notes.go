// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// SourceNotes is the golang structure of table source_notes for DAO operations like Where/Data.
type SourceNotes struct {
	g.Meta         `orm:"table:source_notes, do:true"`
	Id             any //
	DataSourceId   any //
	ScanTaskId     any //
	ChannelId      any //
	ExternalNoteId any //
	NoteCode       any //
	Title          any //
	PlainText      any //
	AttributesJson any //
	RawJson        any //
	Status         any //
	CreatedAt      any //
	UpdatedAt      any //
}
