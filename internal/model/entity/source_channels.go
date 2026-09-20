// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// SourceChannels is the golang structure for table source_channels.
type SourceChannels struct {
	Id           string    `json:"id"           orm:"id"             ` //
	DataSourceId string    `json:"dataSourceId" orm:"data_source_id" ` //
	ChannelId    int64     `json:"channelId"    orm:"channel_id"     ` //
	Title        string    `json:"title"        orm:"title"          ` //
	Username     string    `json:"username"     orm:"username"       ` //
	ChatType     string    `json:"chatType"     orm:"chat_type"      ` //
	RawJson      string    `json:"rawJson"      orm:"raw_json"       ` //
	LastSyncedAt time.Time `json:"lastSyncedAt" orm:"last_synced_at" ` //
	CreatedAt    time.Time `json:"createdAt"    orm:"created_at"     ` //
	UpdatedAt    time.Time `json:"updatedAt"    orm:"updated_at"     ` //
}
