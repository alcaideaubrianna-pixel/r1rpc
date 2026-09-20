// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// SourceChannels is the golang structure of table source_channels for DAO operations like Where/Data.
type SourceChannels struct {
	g.Meta       `orm:"table:source_channels, do:true"`
	Id           any //
	DataSourceId any //
	ChannelId    any //
	Title        any //
	Username     any //
	ChatType     any //
	RawJson      any //
	LastSyncedAt any //
	CreatedAt    any //
	UpdatedAt    any //
}
