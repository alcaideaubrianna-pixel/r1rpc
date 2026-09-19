// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// StorageSettingsDao is the data access object for the table storage_settings.
type StorageSettingsDao struct {
	table    string                 // table is the underlying table name of the DAO.
	group    string                 // group is the database configuration group name of the current DAO.
	columns  StorageSettingsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler     // handlers for customized model modification.
}

// StorageSettingsColumns defines and stores column names for the table storage_settings.
type StorageSettingsColumns struct {
	Id                 string //
	Backend            string //
	LocalPath          string //
	Endpoint           string //
	Region             string //
	Bucket             string //
	PathStyle          string //
	AccessKeyEncrypted string //
	SecretKeyEncrypted string //
	UpdatedAt          string //
}

// storageSettingsColumns holds the columns for the table storage_settings.
var storageSettingsColumns = StorageSettingsColumns{
	Id:                 "id",
	Backend:            "backend",
	LocalPath:          "local_path",
	Endpoint:           "endpoint",
	Region:             "region",
	Bucket:             "bucket",
	PathStyle:          "path_style",
	AccessKeyEncrypted: "access_key_encrypted",
	SecretKeyEncrypted: "secret_key_encrypted",
	UpdatedAt:          "updated_at",
}

// NewStorageSettingsDao creates and returns a new DAO object for table data access.
func NewStorageSettingsDao(handlers ...gdb.ModelHandler) *StorageSettingsDao {
	return &StorageSettingsDao{
		group:    "default",
		table:    "storage_settings",
		columns:  storageSettingsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *StorageSettingsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *StorageSettingsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *StorageSettingsDao) Columns() StorageSettingsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *StorageSettingsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *StorageSettingsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *StorageSettingsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
