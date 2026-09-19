// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageSearchRequestsDao is the data access object for the table image_search_requests.
type ImageSearchRequestsDao struct {
	table    string                     // table is the underlying table name of the DAO.
	group    string                     // group is the database configuration group name of the current DAO.
	columns  ImageSearchRequestsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler         // handlers for customized model modification.
}

// ImageSearchRequestsColumns defines and stores column names for the table image_search_requests.
type ImageSearchRequestsColumns struct {
	Id                 string //
	Source             string //
	ExternalId         string //
	Status             string //
	PipelineName       string //
	PipelineVersion    string //
	Priority           string //
	GroupCount         string //
	MatchedCount       string //
	NotMatchedCount    string //
	RunningCount       string //
	FailedCount        string //
	RequestedByUserId  string //
	RequestedBySubject string //
	CallbackUrl        string //
	CallbackStatus     string //
	CreatedAt          string //
	UpdatedAt          string //
	FinishedAt         string //
	DeletedAt          string //
}

// imageSearchRequestsColumns holds the columns for the table image_search_requests.
var imageSearchRequestsColumns = ImageSearchRequestsColumns{
	Id:                 "id",
	Source:             "source",
	ExternalId:         "external_id",
	Status:             "status",
	PipelineName:       "pipeline_name",
	PipelineVersion:    "pipeline_version",
	Priority:           "priority",
	GroupCount:         "group_count",
	MatchedCount:       "matched_count",
	NotMatchedCount:    "not_matched_count",
	RunningCount:       "running_count",
	FailedCount:        "failed_count",
	RequestedByUserId:  "requested_by_user_id",
	RequestedBySubject: "requested_by_subject",
	CallbackUrl:        "callback_url",
	CallbackStatus:     "callback_status",
	CreatedAt:          "created_at",
	UpdatedAt:          "updated_at",
	FinishedAt:         "finished_at",
	DeletedAt:          "deleted_at",
}

// NewImageSearchRequestsDao creates and returns a new DAO object for table data access.
func NewImageSearchRequestsDao(handlers ...gdb.ModelHandler) *ImageSearchRequestsDao {
	return &ImageSearchRequestsDao{
		group:    "default",
		table:    "image_search_requests",
		columns:  imageSearchRequestsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageSearchRequestsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageSearchRequestsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageSearchRequestsDao) Columns() ImageSearchRequestsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageSearchRequestsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageSearchRequestsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageSearchRequestsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
