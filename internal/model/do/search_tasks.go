// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// SearchTasks is the golang structure of table search_tasks for DAO operations like Where/Data.
type SearchTasks struct {
	g.Meta             `orm:"table:search_tasks, do:true"`
	Id                 any //
	SourceType         any //
	SourceTaskId       any //
	Title              any //
	ConfigId           any //
	ConfigSnapshotJson any //
	Status             any //
	TotalCount         any //
	FilteredCount      any //
	SearchCount        any //
	MatchedCount       any //
	FailedCount        any //
	CreatedAt          any //
	UpdatedAt          any //
}
