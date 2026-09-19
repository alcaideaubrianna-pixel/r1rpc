// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageAssets is the golang structure for table image_assets.
type ImageAssets struct {
	Id               string    `json:"id"               orm:"id"                ` //
	FileId           string    `json:"fileId"           orm:"file_id"           ` //
	Sha256           string    `json:"sha256"           orm:"sha256"            ` //
	MimeType         string    `json:"mimeType"         orm:"mime_type"         ` //
	SizeBytes        int64     `json:"sizeBytes"        orm:"size_bytes"        ` //
	Width            int       `json:"width"            orm:"width"             ` //
	Height           int       `json:"height"           orm:"height"            ` //
	ObjectKey        string    `json:"objectKey"        orm:"object_key"        ` //
	StorageStatus    string    `json:"storageStatus"    orm:"storage_status"    ` //
	Phash            string    `json:"phash"            orm:"phash"             ` //
	Dhash            string    `json:"dhash"            orm:"dhash"             ` //
	Ahash            string    `json:"ahash"            orm:"ahash"             ` //
	AlgorithmVersion string    `json:"algorithmVersion" orm:"algorithm_version" ` //
	CreatedAt        time.Time `json:"createdAt"        orm:"created_at"        ` //
	UpdatedAt        time.Time `json:"updatedAt"        orm:"updated_at"        ` //
	DeletedAt        time.Time `json:"deletedAt"        orm:"deleted_at"        ` //
}
