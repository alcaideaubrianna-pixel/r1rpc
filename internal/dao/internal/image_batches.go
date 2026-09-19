// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageBatchesDao is the data access object for the table image_batches.
type ImageBatchesDao struct {
	table    string              // table is the underlying table name of the DAO.
	group    string              // group is the database configuration group name of the current DAO.
	columns  ImageBatchesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler  // handlers for customized model modification.
}

// ImageBatchesColumns defines and stores column names for the table image_batches.
type ImageBatchesColumns struct {
	Id              string //
	Source          string //
	ExternalId      string //
	Status          string //
	Priority        string //
	ForceRefresh    string //
	TotalCount      string //
	QueuedCount     string //
	RunningCount    string //
	CompletedCount  string //
	FailedCount     string //
	CacheHitCount   string //
	RequestedByJson string //
	CreatedAt       string //
	UpdatedAt       string //
	FinishedAt      string //
	DeletedAt       string //
}

// imageBatchesColumns holds the columns for the table image_batches.
var imageBatchesColumns = ImageBatchesColumns{
	Id:              "id",
	Source:          "source",
	ExternalId:      "external_id",
	Status:          "status",
	Priority:        "priority",
	ForceRefresh:    "force_refresh",
	TotalCount:      "total_count",
	QueuedCount:     "queued_count",
	RunningCount:    "running_count",
	CompletedCount:  "completed_count",
	FailedCount:     "failed_count",
	CacheHitCount:   "cache_hit_count",
	RequestedByJson: "requested_by_json",
	CreatedAt:       "created_at",
	UpdatedAt:       "updated_at",
	FinishedAt:      "finished_at",
	DeletedAt:       "deleted_at",
}

// NewImageBatchesDao creates and returns a new DAO object for table data access.
func NewImageBatchesDao(handlers ...gdb.ModelHandler) *ImageBatchesDao {
	return &ImageBatchesDao{
		group:    "default",
		table:    "image_batches",
		columns:  imageBatchesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageBatchesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageBatchesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageBatchesDao) Columns() ImageBatchesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageBatchesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageBatchesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageBatchesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
