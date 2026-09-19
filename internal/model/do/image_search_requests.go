// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageSearchRequests is the golang structure of table image_search_requests for DAO operations like Where/Data.
type ImageSearchRequests struct {
	g.Meta             `orm:"table:image_search_requests, do:true"`
	Id                 any //
	Source             any //
	ExternalId         any //
	Status             any //
	PipelineName       any //
	PipelineVersion    any //
	Priority           any //
	GroupCount         any //
	MatchedCount       any //
	NotMatchedCount    any //
	RunningCount       any //
	FailedCount        any //
	RequestedByUserId  any //
	RequestedBySubject any //
	CallbackUrl        any //
	CallbackStatus     any //
	CreatedAt          any //
	UpdatedAt          any //
	FinishedAt         any //
	DeletedAt          any //
}
