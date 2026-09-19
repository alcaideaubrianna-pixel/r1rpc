// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageSearchItemsDao is the data access object for the table image_search_items.
type ImageSearchItemsDao struct {
	table    string                  // table is the underlying table name of the DAO.
	group    string                  // group is the database configuration group name of the current DAO.
	columns  ImageSearchItemsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler      // handlers for customized model modification.
}

// ImageSearchItemsColumns defines and stores column names for the table image_search_items.
type ImageSearchItemsColumns struct {
	Id            string //
	GroupId       string //
	SourceAssetId string //
	Ordinal       string //
	Status        string //
	JobId         string //
	ErrorCode     string //
	ErrorMessage  string //
	CreatedAt     string //
	UpdatedAt     string //
	FinishedAt    string //
	DeletedAt     string //
}

// imageSearchItemsColumns holds the columns for the table image_search_items.
var imageSearchItemsColumns = ImageSearchItemsColumns{
	Id:            "id",
	GroupId:       "group_id",
	SourceAssetId: "source_asset_id",
	Ordinal:       "ordinal",
	Status:        "status",
	JobId:         "job_id",
	ErrorCode:     "error_code",
	ErrorMessage:  "error_message",
	CreatedAt:     "created_at",
	UpdatedAt:     "updated_at",
	FinishedAt:    "finished_at",
	DeletedAt:     "deleted_at",
}

// NewImageSearchItemsDao creates and returns a new DAO object for table data access.
func NewImageSearchItemsDao(handlers ...gdb.ModelHandler) *ImageSearchItemsDao {
	return &ImageSearchItemsDao{
		group:    "default",
		table:    "image_search_items",
		columns:  imageSearchItemsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageSearchItemsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageSearchItemsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageSearchItemsDao) Columns() ImageSearchItemsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageSearchItemsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageSearchItemsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageSearchItemsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
