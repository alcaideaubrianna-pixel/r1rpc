// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageSearchGroups is the golang structure of table image_search_groups for DAO operations like Where/Data.
type ImageSearchGroups struct {
	g.Meta          `orm:"table:image_search_groups, do:true"`
	Id              any //
	RequestId       any //
	ExternalId      any //
	SubjectUserId   any //
	Status          any //
	MatchPolicyJson any //
	ItemCount       any //
	QueuedCount     any //
	RunningCount    any //
	CompletedCount  any //
	FailedCount     any //
	CancelledCount  any //
	BestMatchId     any //
	BestScore       any //
	MatchedAt       any //
	CreatedAt       any //
	UpdatedAt       any //
	FinishedAt      any //
	DeletedAt       any //
}
