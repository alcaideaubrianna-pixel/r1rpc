// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ChannelScanTasks is the golang structure for table channel_scan_tasks.
type ChannelScanTasks struct {
	Id                  string    `json:"id"                  orm:"id"                    ` //
	DataSourceId        string    `json:"dataSourceId"        orm:"data_source_id"        ` //
	ChannelId           int64     `json:"channelId"           orm:"channel_id"            ` //
	ChannelTitle        string    `json:"channelTitle"        orm:"channel_title"         ` //
	Mode                string    `json:"mode"                orm:"mode"                  ` //
	InitialLimit        int       `json:"initialLimit"        orm:"initial_limit"         ` //
	PollIntervalMinutes int       `json:"pollIntervalMinutes" orm:"poll_interval_minutes" ` //
	Priority            int       `json:"priority"            orm:"priority"              ` //
	Status              string    `json:"status"              orm:"status"                ` //
	CursorValue         string    `json:"cursorValue"         orm:"cursor_value"          ` //
	Watermark           time.Time `json:"watermark"           orm:"watermark"             ` //
	NextRunAt           time.Time `json:"nextRunAt"           orm:"next_run_at"           ` //
	LastSuccessAt       time.Time `json:"lastSuccessAt"       orm:"last_success_at"       ` //
	LastError           string    `json:"lastError"           orm:"last_error"            ` //
	CreatedAt           time.Time `json:"createdAt"           orm:"created_at"            ` //
	UpdatedAt           time.Time `json:"updatedAt"           orm:"updated_at"            ` //
}
