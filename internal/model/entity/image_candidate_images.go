// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageCandidateImages is the golang structure for table image_candidate_images.
type ImageCandidateImages struct {
	Id               string    `json:"id"               orm:"id"                ` //
	CandidateId      string    `json:"candidateId"      orm:"candidate_id"      ` //
	ImageIndex       int       `json:"imageIndex"       orm:"image_index"       ` //
	SourceUrl        string    `json:"sourceUrl"        orm:"source_url"        ` //
	ImageFileId      string    `json:"imageFileId"      orm:"image_file_id"     ` //
	DownloadStatus   string    `json:"downloadStatus"   orm:"download_status"   ` //
	ImageSha256      string    `json:"imageSha256"      orm:"image_sha256"      ` //
	Phash            string    `json:"phash"            orm:"phash"             ` //
	Dhash            string    `json:"dhash"            orm:"dhash"             ` //
	Ahash            string    `json:"ahash"            orm:"ahash"             ` //
	AlgorithmVersion string    `json:"algorithmVersion" orm:"algorithm_version" ` //
	Score            float64   `json:"score"            orm:"score"             ` //
	PhashDistance    int       `json:"phashDistance"    orm:"phash_distance"    ` //
	DhashDistance    int       `json:"dhashDistance"    orm:"dhash_distance"    ` //
	AhashDistance    int       `json:"ahashDistance"    orm:"ahash_distance"    ` //
	Matched          int       `json:"matched"          orm:"matched"           ` //
	ErrorMessage     string    `json:"errorMessage"     orm:"error_message"     ` //
	CreatedAt        time.Time `json:"createdAt"        orm:"created_at"        ` //
	UpdatedAt        time.Time `json:"updatedAt"        orm:"updated_at"        ` //
}
