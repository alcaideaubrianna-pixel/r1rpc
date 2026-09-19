// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageSearchResponses is the golang structure of table image_search_responses for DAO operations like Where/Data.
type ImageSearchResponses struct {
	g.Meta         `orm:"table:image_search_responses, do:true"`
	Id             any //
	JobId          any //
	RequestId      any //
	NormalizedJson any //
	ItemCount      any //
	ParseStatus    any //
	ErrorMessage   any //
	CreatedAt      any //
	UpdatedAt      any //
	RawJson        any //
}
