// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ChannelScanTasksDao is the data access object for the table channel_scan_tasks.
type ChannelScanTasksDao struct {
	table    string                  // table is the underlying table name of the DAO.
	group    string                  // group is the database configuration group name of the current DAO.
	columns  ChannelScanTasksColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler      // handlers for customized model modification.
}

// ChannelScanTasksColumns defines and stores column names for the table channel_scan_tasks.
type ChannelScanTasksColumns struct {
	Id                   string //
	DataSourceId         string //
	ChannelId            string //
	ChannelTitle         string //
	Mode                 string //
	InitialLimit         string //
	PollIntervalMinutes  string //
	Priority             string //
	Status               string //
	CursorValue          string //
	Watermark            string //
	NextRunAt            string //
	LastSuccessAt        string //
	LastError            string //
	ImageSearchRequestId string //
	CreatedAt            string //
	UpdatedAt            string //
}

// channelScanTasksColumns holds the columns for the table channel_scan_tasks.
var channelScanTasksColumns = ChannelScanTasksColumns{
	Id:                   "id",
	DataSourceId:         "data_source_id",
	ChannelId:            "channel_id",
	ChannelTitle:         "channel_title",
	Mode:                 "mode",
	InitialLimit:         "initial_limit",
	PollIntervalMinutes:  "poll_interval_minutes",
	Priority:             "priority",
	Status:               "status",
	CursorValue:          "cursor_value",
	Watermark:            "watermark",
	NextRunAt:            "next_run_at",
	LastSuccessAt:        "last_success_at",
	LastError:            "last_error",
	ImageSearchRequestId: "image_search_request_id",
	CreatedAt:            "created_at",
	UpdatedAt:            "updated_at",
}

// NewChannelScanTasksDao creates and returns a new DAO object for table data access.
func NewChannelScanTasksDao(handlers ...gdb.ModelHandler) *ChannelScanTasksDao {
	return &ChannelScanTasksDao{
		group:    "default",
		table:    "channel_scan_tasks",
		columns:  channelScanTasksColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ChannelScanTasksDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ChannelScanTasksDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ChannelScanTasksDao) Columns() ChannelScanTasksColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ChannelScanTasksDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ChannelScanTasksDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ChannelScanTasksDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
