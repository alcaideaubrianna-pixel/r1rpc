// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SourceChannelsDao is the data access object for the table source_channels.
type SourceChannelsDao struct {
	table    string                // table is the underlying table name of the DAO.
	group    string                // group is the database configuration group name of the current DAO.
	columns  SourceChannelsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler    // handlers for customized model modification.
}

// SourceChannelsColumns defines and stores column names for the table source_channels.
type SourceChannelsColumns struct {
	Id           string //
	DataSourceId string //
	ChannelId    string //
	Title        string //
	Username     string //
	ChatType     string //
	RawJson      string //
	LastSyncedAt string //
	CreatedAt    string //
	UpdatedAt    string //
}

// sourceChannelsColumns holds the columns for the table source_channels.
var sourceChannelsColumns = SourceChannelsColumns{
	Id:           "id",
	DataSourceId: "data_source_id",
	ChannelId:    "channel_id",
	Title:        "title",
	Username:     "username",
	ChatType:     "chat_type",
	RawJson:      "raw_json",
	LastSyncedAt: "last_synced_at",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
}

// NewSourceChannelsDao creates and returns a new DAO object for table data access.
func NewSourceChannelsDao(handlers ...gdb.ModelHandler) *SourceChannelsDao {
	return &SourceChannelsDao{
		group:    "default",
		table:    "source_channels",
		columns:  sourceChannelsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SourceChannelsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SourceChannelsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SourceChannelsDao) Columns() SourceChannelsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SourceChannelsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SourceChannelsDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *SourceChannelsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
