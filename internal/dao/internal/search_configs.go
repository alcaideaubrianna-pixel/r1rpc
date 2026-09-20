// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SearchConfigsDao is the data access object for the table search_configs.
type SearchConfigsDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  SearchConfigsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// SearchConfigsColumns defines and stores column names for the table search_configs.
type SearchConfigsColumns struct {
	Id               string //
	Name             string //
	OcrEnabled       string //
	OcrKeywordsJson  string //
	OcrMatchMode     string //
	ScoreThreshold   string //
	MaxPhashDistance string //
	MaxDhashDistance string //
	MaxAhashDistance string //
	MaxCandidates    string //
	PipelineName     string //
	Enabled          string //
	Version          string //
	CreatedAt        string //
	UpdatedAt        string //
}

// searchConfigsColumns holds the columns for the table search_configs.
var searchConfigsColumns = SearchConfigsColumns{
	Id:               "id",
	Name:             "name",
	OcrEnabled:       "ocr_enabled",
	OcrKeywordsJson:  "ocr_keywords_json",
	OcrMatchMode:     "ocr_match_mode",
	ScoreThreshold:   "score_threshold",
	MaxPhashDistance: "max_phash_distance",
	MaxDhashDistance: "max_dhash_distance",
	MaxAhashDistance: "max_ahash_distance",
	MaxCandidates:    "max_candidates",
	PipelineName:     "pipeline_name",
	Enabled:          "enabled",
	Version:          "version",
	CreatedAt:        "created_at",
	UpdatedAt:        "updated_at",
}

// NewSearchConfigsDao creates and returns a new DAO object for table data access.
func NewSearchConfigsDao(handlers ...gdb.ModelHandler) *SearchConfigsDao {
	return &SearchConfigsDao{
		group:    "default",
		table:    "search_configs",
		columns:  searchConfigsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SearchConfigsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SearchConfigsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SearchConfigsDao) Columns() SearchConfigsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SearchConfigsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SearchConfigsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SearchConfigsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
