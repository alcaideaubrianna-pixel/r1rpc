// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// OcrFilterRules is the golang structure for table ocr_filter_rules.
type OcrFilterRules struct {
	Id          string    `json:"id"          orm:"id"           ` //
	Name        string    `json:"name"        orm:"name"         ` //
	Mode        string    `json:"mode"        orm:"mode"         ` //
	PatternJson string    `json:"patternJson" orm:"pattern_json" ` //
	Enabled     int       `json:"enabled"     orm:"enabled"      ` //
	Version     int       `json:"version"     orm:"version"      ` //
	CreatedAt   time.Time `json:"createdAt"   orm:"created_at"   ` //
	UpdatedAt   time.Time `json:"updatedAt"   orm:"updated_at"   ` //
}
