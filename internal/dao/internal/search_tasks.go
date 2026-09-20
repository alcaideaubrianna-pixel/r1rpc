// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SearchTasksDao is the data access object for the table search_tasks.
type SearchTasksDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SearchTasksColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SearchTasksColumns defines and stores column names for the table search_tasks.
type SearchTasksColumns struct {
	Id                 string //
	SourceType         string //
	SourceTaskId       string //
	Title              string //
	ConfigId           string //
	ConfigSnapshotJson string //
	Status             string //
	TotalCount         string //
	FilteredCount      string //
	SearchCount        string //
	MatchedCount       string //
	FailedCount        string //
	CreatedAt          string //
	UpdatedAt          string //
}

// searchTasksColumns holds the columns for the table search_tasks.
var searchTasksColumns = SearchTasksColumns{
	Id:                 "id",
	SourceType:         "source_type",
	SourceTaskId:       "source_task_id",
	Title:              "title",
	ConfigId:           "config_id",
	ConfigSnapshotJson: "config_snapshot_json",
	Status:             "status",
	TotalCount:         "total_count",
	FilteredCount:      "filtered_count",
	SearchCount:        "search_count",
	MatchedCount:       "matched_count",
	FailedCount:        "failed_count",
	CreatedAt:          "created_at",
	UpdatedAt:          "updated_at",
}

// NewSearchTasksDao creates and returns a new DAO object for table data access.
func NewSearchTasksDao(handlers ...gdb.ModelHandler) *SearchTasksDao {
	return &SearchTasksDao{
		group:    "default",
		table:    "search_tasks",
		columns:  searchTasksColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SearchTasksDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SearchTasksDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SearchTasksDao) Columns() SearchTasksColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SearchTasksDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SearchTasksDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SearchTasksDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
