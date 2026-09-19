// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// Files is the golang structure of table files for DAO operations like Where/Data.
type Files struct {
	g.Meta       `orm:"table:files, do:true"`
	Id           any //
	ObjectKey    any //
	OriginalName any //
	ContentType  any //
	SizeBytes    any //
	Sha256       any //
	Backend      any //
	Status       any //
	CreatedBy    any //
	CreatedAt    any //
	ExpiresAt    any //
	DeletedAt    any //
}
