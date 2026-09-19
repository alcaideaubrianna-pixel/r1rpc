// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageCandidates is the golang structure for table image_candidates.
type ImageCandidates struct {
	Id               string    `json:"id"               orm:"id"                ` //
	ResponseId       string    `json:"responseId"       orm:"response_id"       ` //
	JobId            string    `json:"jobId"            orm:"job_id"            ` //
	SearchItemId     string    `json:"searchItemId"     orm:"search_item_id"    ` //
	RankNo           int       `json:"rankNo"           orm:"rank_no"           ` //
	ContentId        string    `json:"contentId"        orm:"content_id"        ` //
	Title            string    `json:"title"            orm:"title"             ` //
	AuthorId         string    `json:"authorId"         orm:"author_id"         ` //
	AuthorName       string    `json:"authorName"       orm:"author_name"       ` //
	CoverUrl         string    `json:"coverUrl"         orm:"cover_url"         ` //
	RawItemJson      string    `json:"rawItemJson"      orm:"raw_item_json"     ` //
	DownloadStatus   string    `json:"downloadStatus"   orm:"download_status"   ` //
	ImageSha256      string    `json:"imageSha256"      orm:"image_sha256"      ` //
	Phash            string    `json:"phash"            orm:"phash"             ` //
	Dhash            string    `json:"dhash"            orm:"dhash"             ` //
	Ahash            string    `json:"ahash"            orm:"ahash"             ` //
	AlgorithmVersion string    `json:"algorithmVersion" orm:"algorithm_version" ` //
	Score            float64   `json:"score"            orm:"score"             ` //
	Matched          int       `json:"matched"          orm:"matched"           ` //
	ErrorMessage     string    `json:"errorMessage"     orm:"error_message"     ` //
	CreatedAt        time.Time `json:"createdAt"        orm:"created_at"        ` //
	UpdatedAt        time.Time `json:"updatedAt"        orm:"updated_at"        ` //
	ImageFileId      string    `json:"imageFileId"      orm:"image_file_id"     ` //
}
