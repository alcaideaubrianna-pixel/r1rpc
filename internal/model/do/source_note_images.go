// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// SourceNoteImages is the golang structure of table source_note_images for DAO operations like Where/Data.
type SourceNoteImages struct {
	g.Meta           `orm:"table:source_note_images, do:true"`
	Id               any //
	NoteId           any //
	ExternalAssetId  any //
	AssetType        any //
	SourceUrl        any //
	FileId           any //
	Sha256           any //
	Phash            any //
	DownloadStatus   any //
	PreprocessStatus any //
	OcrText          any //
	FilterDecision   any //
	FilterReason     any //
	RawJson          any //
	ImageIndex       any //
	CreatedAt        any //
	UpdatedAt        any //
}
