// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageJobsDao is the data access object for the table image_jobs.
type ImageJobsDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  ImageJobsColumns   // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// ImageJobsColumns defines and stores column names for the table image_jobs.
type ImageJobsColumns struct {
	Id                         string //
	BatchId                    string //
	SourceAssetId              string //
	Source                     string //
	ExternalId                 string //
	Status                     string //
	Stage                      string //
	Priority                   string //
	ForceRefresh               string //
	Attempt                    string //
	AssignedClientId           string //
	AssignedSessionIncarnation string //
	UploadRequestId            string //
	SearchRequestId            string //
	UploadHandle               string //
	CacheSourceJobId           string //
	ErrorCode                  string //
	ErrorMessage               string //
	InputJson                  string //
	NormalizedResponseJson     string //
	CreatedAt                  string //
	UpdatedAt                  string //
	StartedAt                  string //
	FinishedAt                 string //
	DeletedAt                  string //
}

// imageJobsColumns holds the columns for the table image_jobs.
var imageJobsColumns = ImageJobsColumns{
	Id:                         "id",
	BatchId:                    "batch_id",
	SourceAssetId:              "source_asset_id",
	Source:                     "source",
	ExternalId:                 "external_id",
	Status:                     "status",
	Stage:                      "stage",
	Priority:                   "priority",
	ForceRefresh:               "force_refresh",
	Attempt:                    "attempt",
	AssignedClientId:           "assigned_client_id",
	AssignedSessionIncarnation: "assigned_session_incarnation",
	UploadRequestId:            "upload_request_id",
	SearchRequestId:            "search_request_id",
	UploadHandle:               "upload_handle",
	CacheSourceJobId:           "cache_source_job_id",
	ErrorCode:                  "error_code",
	ErrorMessage:               "error_message",
	InputJson:                  "input_json",
	NormalizedResponseJson:     "normalized_response_json",
	CreatedAt:                  "created_at",
	UpdatedAt:                  "updated_at",
	StartedAt:                  "started_at",
	FinishedAt:                 "finished_at",
	DeletedAt:                  "deleted_at",
}

// NewImageJobsDao creates and returns a new DAO object for table data access.
func NewImageJobsDao(handlers ...gdb.ModelHandler) *ImageJobsDao {
	return &ImageJobsDao{
		group:    "default",
		table:    "image_jobs",
		columns:  imageJobsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageJobsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageJobsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageJobsDao) Columns() ImageJobsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageJobsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageJobsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageJobsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
