// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageAssetsDao is the data access object for the table image_assets.
type ImageAssetsDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  ImageAssetsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// ImageAssetsColumns defines and stores column names for the table image_assets.
type ImageAssetsColumns struct {
	Id               string //
	FileId           string //
	Sha256           string //
	MimeType         string //
	SizeBytes        string //
	Width            string //
	Height           string //
	ObjectKey        string //
	StorageStatus    string //
	Phash            string //
	Dhash            string //
	Ahash            string //
	AlgorithmVersion string //
	CreatedAt        string //
	UpdatedAt        string //
	DeletedAt        string //
}

// imageAssetsColumns holds the columns for the table image_assets.
var imageAssetsColumns = ImageAssetsColumns{
	Id:               "id",
	FileId:           "file_id",
	Sha256:           "sha256",
	MimeType:         "mime_type",
	SizeBytes:        "size_bytes",
	Width:            "width",
	Height:           "height",
	ObjectKey:        "object_key",
	StorageStatus:    "storage_status",
	Phash:            "phash",
	Dhash:            "dhash",
	Ahash:            "ahash",
	AlgorithmVersion: "algorithm_version",
	CreatedAt:        "created_at",
	UpdatedAt:        "updated_at",
	DeletedAt:        "deleted_at",
}

// NewImageAssetsDao creates and returns a new DAO object for table data access.
func NewImageAssetsDao(handlers ...gdb.ModelHandler) *ImageAssetsDao {
	return &ImageAssetsDao{
		group:    "default",
		table:    "image_assets",
		columns:  imageAssetsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageAssetsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageAssetsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageAssetsDao) Columns() ImageAssetsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageAssetsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageAssetsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageAssetsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
