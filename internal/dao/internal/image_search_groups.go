// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageSearchGroupsDao is the data access object for the table image_search_groups.
type ImageSearchGroupsDao struct {
	table    string                   // table is the underlying table name of the DAO.
	group    string                   // group is the database configuration group name of the current DAO.
	columns  ImageSearchGroupsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler       // handlers for customized model modification.
}

// ImageSearchGroupsColumns defines and stores column names for the table image_search_groups.
type ImageSearchGroupsColumns struct {
	Id              string //
	RequestId       string //
	ExternalId      string //
	SubjectUserId   string //
	Status          string //
	MatchPolicyJson string //
	ItemCount       string //
	QueuedCount     string //
	RunningCount    string //
	CompletedCount  string //
	FailedCount     string //
	CancelledCount  string //
	BestMatchId     string //
	BestScore       string //
	MatchedAt       string //
	CreatedAt       string //
	UpdatedAt       string //
	FinishedAt      string //
	DeletedAt       string //
}

// imageSearchGroupsColumns holds the columns for the table image_search_groups.
var imageSearchGroupsColumns = ImageSearchGroupsColumns{
	Id:              "id",
	RequestId:       "request_id",
	ExternalId:      "external_id",
	SubjectUserId:   "subject_user_id",
	Status:          "status",
	MatchPolicyJson: "match_policy_json",
	ItemCount:       "item_count",
	QueuedCount:     "queued_count",
	RunningCount:    "running_count",
	CompletedCount:  "completed_count",
	FailedCount:     "failed_count",
	CancelledCount:  "cancelled_count",
	BestMatchId:     "best_match_id",
	BestScore:       "best_score",
	MatchedAt:       "matched_at",
	CreatedAt:       "created_at",
	UpdatedAt:       "updated_at",
	FinishedAt:      "finished_at",
	DeletedAt:       "deleted_at",
}

// NewImageSearchGroupsDao creates and returns a new DAO object for table data access.
func NewImageSearchGroupsDao(handlers ...gdb.ModelHandler) *ImageSearchGroupsDao {
	return &ImageSearchGroupsDao{
		group:    "default",
		table:    "image_search_groups",
		columns:  imageSearchGroupsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageSearchGroupsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageSearchGroupsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageSearchGroupsDao) Columns() ImageSearchGroupsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageSearchGroupsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageSearchGroupsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageSearchGroupsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
