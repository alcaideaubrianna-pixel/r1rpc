// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageSearchResponsesDao is the data access object for the table image_search_responses.
type ImageSearchResponsesDao struct {
	table    string                      // table is the underlying table name of the DAO.
	group    string                      // group is the database configuration group name of the current DAO.
	columns  ImageSearchResponsesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler          // handlers for customized model modification.
}

// ImageSearchResponsesColumns defines and stores column names for the table image_search_responses.
type ImageSearchResponsesColumns struct {
	Id             string //
	JobId          string //
	RequestId      string //
	NormalizedJson string //
	ItemCount      string //
	ParseStatus    string //
	ErrorMessage   string //
	CreatedAt      string //
	UpdatedAt      string //
	RawJson        string //
}

// imageSearchResponsesColumns holds the columns for the table image_search_responses.
var imageSearchResponsesColumns = ImageSearchResponsesColumns{
	Id:             "id",
	JobId:          "job_id",
	RequestId:      "request_id",
	NormalizedJson: "normalized_json",
	ItemCount:      "item_count",
	ParseStatus:    "parse_status",
	ErrorMessage:   "error_message",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
	RawJson:        "raw_json",
}

// NewImageSearchResponsesDao creates and returns a new DAO object for table data access.
func NewImageSearchResponsesDao(handlers ...gdb.ModelHandler) *ImageSearchResponsesDao {
	return &ImageSearchResponsesDao{
		group:    "default",
		table:    "image_search_responses",
		columns:  imageSearchResponsesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageSearchResponsesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageSearchResponsesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageSearchResponsesDao) Columns() ImageSearchResponsesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageSearchResponsesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageSearchResponsesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageSearchResponsesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
