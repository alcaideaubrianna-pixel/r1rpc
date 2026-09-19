// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageMatches is the golang structure of table image_matches for DAO operations like Where/Data.
type ImageMatches struct {
	g.Meta           `orm:"table:image_matches, do:true"`
	Id               any //
	SearchItemId     any //
	CandidateId      any //
	Algorithm        any //
	AlgorithmVersion any //
	PhashDistance    any //
	DhashDistance    any //
	AhashDistance    any //
	Score            any //
	Decision         any //
	CreatedAt        any //
}
