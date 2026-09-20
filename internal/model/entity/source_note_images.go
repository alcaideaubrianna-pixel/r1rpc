// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// SourceNoteImages is the golang structure for table source_note_images.
type SourceNoteImages struct {
	Id               string    `json:"id"               orm:"id"                ` //
	NoteId           string    `json:"noteId"           orm:"note_id"           ` //
	ExternalAssetId  string    `json:"externalAssetId"  orm:"external_asset_id" ` //
	AssetType        string    `json:"assetType"        orm:"asset_type"        ` //
	SourceUrl        string    `json:"sourceUrl"        orm:"source_url"        ` //
	FileId           string    `json:"fileId"           orm:"file_id"           ` //
	Sha256           string    `json:"sha256"           orm:"sha256"            ` //
	Phash            string    `json:"phash"            orm:"phash"             ` //
	DownloadStatus   string    `json:"downloadStatus"   orm:"download_status"   ` //
	PreprocessStatus string    `json:"preprocessStatus" orm:"preprocess_status" ` //
	OcrText          string    `json:"ocrText"          orm:"ocr_text"          ` //
	FilterDecision   string    `json:"filterDecision"   orm:"filter_decision"   ` //
	FilterReason     string    `json:"filterReason"     orm:"filter_reason"     ` //
	RawJson          string    `json:"rawJson"          orm:"raw_json"          ` //
	ImageIndex       int       `json:"imageIndex"       orm:"image_index"       ` //
	CreatedAt        time.Time `json:"createdAt"        orm:"created_at"        ` //
	UpdatedAt        time.Time `json:"updatedAt"        orm:"updated_at"        ` //
}
