// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// XhsUserProfiles is the golang structure for table xhs_user_profiles.
type XhsUserProfiles struct {
	UserId         string    `json:"userId"         orm:"user_id"         ` //
	RedId          string    `json:"redId"          orm:"red_id"          ` //
	Nickname       string    `json:"nickname"       orm:"nickname"        ` //
	AvatarUrl      string    `json:"avatarUrl"      orm:"avatar_url"      ` //
	AvatarFileId   string    `json:"avatarFileId"   orm:"avatar_file_id"  ` //
	Description    string    `json:"description"    orm:"description"     ` //
	Gender         int       `json:"gender"         orm:"gender"          ` //
	IpLocation     string    `json:"ipLocation"     orm:"ip_location"     ` //
	FansCount      int64     `json:"fansCount"      orm:"fans_count"      ` //
	LikedCount     int64     `json:"likedCount"     orm:"liked_count"     ` //
	CollectedCount int64     `json:"collectedCount" orm:"collected_count" ` //
	NoteCount      int64     `json:"noteCount"      orm:"note_count"      ` //
	ShareLink      string    `json:"shareLink"      orm:"share_link"      ` //
	RawJson        string    `json:"rawJson"        orm:"raw_json"        ` //
	FetchedAt      time.Time `json:"fetchedAt"      orm:"fetched_at"      ` //
	CreatedAt      time.Time `json:"createdAt"      orm:"created_at"      ` //
	UpdatedAt      time.Time `json:"updatedAt"      orm:"updated_at"      ` //
}
