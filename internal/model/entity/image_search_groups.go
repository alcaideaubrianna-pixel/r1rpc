// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageSearchGroups is the golang structure for table image_search_groups.
type ImageSearchGroups struct {
	Id              string    `json:"id"              orm:"id"                ` //
	RequestId       string    `json:"requestId"       orm:"request_id"        ` //
	ExternalId      string    `json:"externalId"      orm:"external_id"       ` //
	SubjectUserId   string    `json:"subjectUserId"   orm:"subject_user_id"   ` //
	Status          string    `json:"status"          orm:"status"            ` //
	MatchPolicyJson string    `json:"matchPolicyJson" orm:"match_policy_json" ` //
	ItemCount       int       `json:"itemCount"       orm:"item_count"        ` //
	QueuedCount     int       `json:"queuedCount"     orm:"queued_count"      ` //
	RunningCount    int       `json:"runningCount"    orm:"running_count"     ` //
	CompletedCount  int       `json:"completedCount"  orm:"completed_count"   ` //
	FailedCount     int       `json:"failedCount"     orm:"failed_count"      ` //
	CancelledCount  int       `json:"cancelledCount"  orm:"cancelled_count"   ` //
	BestMatchId     string    `json:"bestMatchId"     orm:"best_match_id"     ` //
	BestScore       float64   `json:"bestScore"       orm:"best_score"        ` //
	MatchedAt       time.Time `json:"matchedAt"       orm:"matched_at"        ` //
	CreatedAt       time.Time `json:"createdAt"       orm:"created_at"        ` //
	UpdatedAt       time.Time `json:"updatedAt"       orm:"updated_at"        ` //
	FinishedAt      time.Time `json:"finishedAt"      orm:"finished_at"       ` //
	DeletedAt       time.Time `json:"deletedAt"       orm:"deleted_at"        ` //
}
