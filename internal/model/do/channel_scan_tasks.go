// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ChannelScanTasks is the golang structure of table channel_scan_tasks for DAO operations like Where/Data.
type ChannelScanTasks struct {
	g.Meta              `orm:"table:channel_scan_tasks, do:true"`
	Id                  any //
	DataSourceId        any //
	ChannelId           any //
	ChannelTitle        any //
	Mode                any //
	InitialLimit        any //
	PollIntervalMinutes any //
	Priority            any //
	Status              any //
	CursorValue         any //
	Watermark           any //
	NextRunAt           any //
	LastSuccessAt       any //
	LastError           any //
	CreatedAt           any //
	UpdatedAt           any //
}
