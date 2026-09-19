// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageMatches is the golang structure for table image_matches.
type ImageMatches struct {
	Id               string    `json:"id"               orm:"id"                ` //
	SearchItemId     string    `json:"searchItemId"     orm:"search_item_id"    ` //
	CandidateId      string    `json:"candidateId"      orm:"candidate_id"      ` //
	Algorithm        string    `json:"algorithm"        orm:"algorithm"         ` //
	AlgorithmVersion string    `json:"algorithmVersion" orm:"algorithm_version" ` //
	PhashDistance    int       `json:"phashDistance"    orm:"phash_distance"    ` //
	DhashDistance    int       `json:"dhashDistance"    orm:"dhash_distance"    ` //
	AhashDistance    int       `json:"ahashDistance"    orm:"ahash_distance"    ` //
	Score            float64   `json:"score"            orm:"score"             ` //
	Decision         string    `json:"decision"         orm:"decision"          ` //
	CreatedAt        time.Time `json:"createdAt"        orm:"created_at"        ` //
}
