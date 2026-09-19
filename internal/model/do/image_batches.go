// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageBatches is the golang structure of table image_batches for DAO operations like Where/Data.
type ImageBatches struct {
	g.Meta          `orm:"table:image_batches, do:true"`
	Id              any //
	Source          any //
	ExternalId      any //
	Status          any //
	Priority        any //
	ForceRefresh    any //
	TotalCount      any //
	QueuedCount     any //
	RunningCount    any //
	CompletedCount  any //
	FailedCount     any //
	CacheHitCount   any //
	RequestedByJson any //
	CreatedAt       any //
	UpdatedAt       any //
	FinishedAt      any //
	DeletedAt       any //
}
