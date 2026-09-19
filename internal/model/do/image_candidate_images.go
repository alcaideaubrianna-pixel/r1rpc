// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageCandidateImages is the golang structure of table image_candidate_images for DAO operations like Where/Data.
type ImageCandidateImages struct {
	g.Meta           `orm:"table:image_candidate_images, do:true"`
	Id               any //
	CandidateId      any //
	ImageIndex       any //
	SourceUrl        any //
	ImageFileId      any //
	DownloadStatus   any //
	ImageSha256      any //
	Phash            any //
	Dhash            any //
	Ahash            any //
	AlgorithmVersion any //
	Score            any //
	PhashDistance    any //
	DhashDistance    any //
	AhashDistance    any //
	Matched          any //
	ErrorMessage     any //
	CreatedAt        any //
	UpdatedAt        any //
}
