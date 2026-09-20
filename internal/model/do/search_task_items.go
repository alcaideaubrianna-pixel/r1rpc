// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// SearchTaskItems is the golang structure of table search_task_items for DAO operations like Where/Data.
type SearchTaskItems struct {
	g.Meta               `orm:"table:search_task_items, do:true"`
	Id                   any //
	SearchTaskId         any //
	SourceNoteId         any //
	ExternalId           any //
	Title                any //
	Status               any //
	PreprocessStatus     any //
	FilterStatus         any //
	FilterReason         any //
	ImageSearchRequestId any //
	Matched              any //
	BestScore            any //
	ErrorMessage         any //
	CreatedAt            any //
	UpdatedAt            any //
}
