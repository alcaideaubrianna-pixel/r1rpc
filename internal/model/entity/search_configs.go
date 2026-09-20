// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// SearchConfigs is the golang structure for table search_configs.
type SearchConfigs struct {
	Id               string    `json:"id"               orm:"id"                 ` //
	Name             string    `json:"name"             orm:"name"               ` //
	OcrEnabled       int       `json:"ocrEnabled"       orm:"ocr_enabled"        ` //
	OcrKeywordsJson  string    `json:"ocrKeywordsJson"  orm:"ocr_keywords_json"  ` //
	OcrMatchMode     string    `json:"ocrMatchMode"     orm:"ocr_match_mode"     ` //
	ScoreThreshold   float64   `json:"scoreThreshold"   orm:"score_threshold"    ` //
	MaxPhashDistance int       `json:"maxPhashDistance" orm:"max_phash_distance" ` //
	MaxDhashDistance int       `json:"maxDhashDistance" orm:"max_dhash_distance" ` //
	MaxAhashDistance int       `json:"maxAhashDistance" orm:"max_ahash_distance" ` //
	MaxCandidates    int       `json:"maxCandidates"    orm:"max_candidates"     ` //
	PipelineName     string    `json:"pipelineName"     orm:"pipeline_name"      ` //
	Enabled          int       `json:"enabled"          orm:"enabled"            ` //
	Version          int       `json:"version"          orm:"version"            ` //
	CreatedAt        time.Time `json:"createdAt"        orm:"created_at"         ` //
	UpdatedAt        time.Time `json:"updatedAt"        orm:"updated_at"         ` //
}
