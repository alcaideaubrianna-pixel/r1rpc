// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageAssets is the golang structure of table image_assets for DAO operations like Where/Data.
type ImageAssets struct {
	g.Meta           `orm:"table:image_assets, do:true"`
	Id               any //
	FileId           any //
	Sha256           any //
	MimeType         any //
	SizeBytes        any //
	Width            any //
	Height           any //
	ObjectKey        any //
	StorageStatus    any //
	Phash            any //
	Dhash            any //
	Ahash            any //
	AlgorithmVersion any //
	CreatedAt        any //
	UpdatedAt        any //
	DeletedAt        any //
}
