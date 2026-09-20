// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// DataSourcesDao is the data access object for the table data_sources.
type DataSourcesDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  DataSourcesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// DataSourcesColumns defines and stores column names for the table data_sources.
type DataSourcesColumns struct {
	Id                    string //
	Name                  string //
	BaseUrl               string //
	ImageBaseUrl          string //
	AppId                 string //
	AccessKey             string //
	SecretKeyEncrypted    string //
	Status                string //
	RequestTimeoutSeconds string //
	CreatedAt             string //
	UpdatedAt             string //
}

// dataSourcesColumns holds the columns for the table data_sources.
var dataSourcesColumns = DataSourcesColumns{
	Id:                    "id",
	Name:                  "name",
	BaseUrl:               "base_url",
	ImageBaseUrl:          "image_base_url",
	AppId:                 "app_id",
	AccessKey:             "access_key",
	SecretKeyEncrypted:    "secret_key_encrypted",
	Status:                "status",
	RequestTimeoutSeconds: "request_timeout_seconds",
	CreatedAt:             "created_at",
	UpdatedAt:             "updated_at",
}

// NewDataSourcesDao creates and returns a new DAO object for table data access.
func NewDataSourcesDao(handlers ...gdb.ModelHandler) *DataSourcesDao {
	return &DataSourcesDao{
		group:    "default",
		table:    "data_sources",
		columns:  dataSourcesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *DataSourcesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *DataSourcesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *DataSourcesDao) Columns() DataSourcesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *DataSourcesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *DataSourcesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *DataSourcesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
