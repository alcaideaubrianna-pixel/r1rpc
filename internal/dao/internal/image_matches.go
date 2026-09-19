// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageMatchesDao is the data access object for the table image_matches.
type ImageMatchesDao struct {
	table    string              // table is the underlying table name of the DAO.
	group    string              // group is the database configuration group name of the current DAO.
	columns  ImageMatchesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler  // handlers for customized model modification.
}

// ImageMatchesColumns defines and stores column names for the table image_matches.
type ImageMatchesColumns struct {
	Id               string //
	SearchItemId     string //
	CandidateId      string //
	Algorithm        string //
	AlgorithmVersion string //
	PhashDistance    string //
	DhashDistance    string //
	AhashDistance    string //
	Score            string //
	Decision         string //
	CreatedAt        string //
}

// imageMatchesColumns holds the columns for the table image_matches.
var imageMatchesColumns = ImageMatchesColumns{
	Id:               "id",
	SearchItemId:     "search_item_id",
	CandidateId:      "candidate_id",
	Algorithm:        "algorithm",
	AlgorithmVersion: "algorithm_version",
	PhashDistance:    "phash_distance",
	DhashDistance:    "dhash_distance",
	AhashDistance:    "ahash_distance",
	Score:            "score",
	Decision:         "decision",
	CreatedAt:        "created_at",
}

// NewImageMatchesDao creates and returns a new DAO object for table data access.
func NewImageMatchesDao(handlers ...gdb.ModelHandler) *ImageMatchesDao {
	return &ImageMatchesDao{
		group:    "default",
		table:    "image_matches",
		columns:  imageMatchesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageMatchesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageMatchesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageMatchesDao) Columns() ImageMatchesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageMatchesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageMatchesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageMatchesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
