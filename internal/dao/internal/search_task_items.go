// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SearchTaskItemsDao is the data access object for the table search_task_items.
type SearchTaskItemsDao struct {
	table    string                 // table is the underlying table name of the DAO.
	group    string                 // group is the database configuration group name of the current DAO.
	columns  SearchTaskItemsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler     // handlers for customized model modification.
}

// SearchTaskItemsColumns defines and stores column names for the table search_task_items.
type SearchTaskItemsColumns struct {
	Id                   string //
	SearchTaskId         string //
	SourceNoteId         string //
	ExternalId           string //
	Title                string //
	Status               string //
	PreprocessStatus     string //
	FilterStatus         string //
	FilterReason         string //
	ImageSearchRequestId string //
	Matched              string //
	BestScore            string //
	ErrorMessage         string //
	CreatedAt            string //
	UpdatedAt            string //
}

// searchTaskItemsColumns holds the columns for the table search_task_items.
var searchTaskItemsColumns = SearchTaskItemsColumns{
	Id:                   "id",
	SearchTaskId:         "search_task_id",
	SourceNoteId:         "source_note_id",
	ExternalId:           "external_id",
	Title:                "title",
	Status:               "status",
	PreprocessStatus:     "preprocess_status",
	FilterStatus:         "filter_status",
	FilterReason:         "filter_reason",
	ImageSearchRequestId: "image_search_request_id",
	Matched:              "matched",
	BestScore:            "best_score",
	ErrorMessage:         "error_message",
	CreatedAt:            "created_at",
	UpdatedAt:            "updated_at",
}

// NewSearchTaskItemsDao creates and returns a new DAO object for table data access.
func NewSearchTaskItemsDao(handlers ...gdb.ModelHandler) *SearchTaskItemsDao {
	return &SearchTaskItemsDao{
		group:    "default",
		table:    "search_task_items",
		columns:  searchTaskItemsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SearchTaskItemsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SearchTaskItemsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SearchTaskItemsDao) Columns() SearchTaskItemsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SearchTaskItemsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SearchTaskItemsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SearchTaskItemsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
