// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// XhsUserProfilesDao is the data access object for the table xhs_user_profiles.
type XhsUserProfilesDao struct {
	table    string                 // table is the underlying table name of the DAO.
	group    string                 // group is the database configuration group name of the current DAO.
	columns  XhsUserProfilesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler     // handlers for customized model modification.
}

// XhsUserProfilesColumns defines and stores column names for the table xhs_user_profiles.
type XhsUserProfilesColumns struct {
	UserId         string //
	RedId          string //
	Nickname       string //
	AvatarUrl      string //
	AvatarFileId   string //
	Description    string //
	Gender         string //
	IpLocation     string //
	FansCount      string //
	LikedCount     string //
	CollectedCount string //
	NoteCount      string //
	ShareLink      string //
	RawJson        string //
	FetchedAt      string //
	CreatedAt      string //
	UpdatedAt      string //
}

// xhsUserProfilesColumns holds the columns for the table xhs_user_profiles.
var xhsUserProfilesColumns = XhsUserProfilesColumns{
	UserId:         "user_id",
	RedId:          "red_id",
	Nickname:       "nickname",
	AvatarUrl:      "avatar_url",
	AvatarFileId:   "avatar_file_id",
	Description:    "description",
	Gender:         "gender",
	IpLocation:     "ip_location",
	FansCount:      "fans_count",
	LikedCount:     "liked_count",
	CollectedCount: "collected_count",
	NoteCount:      "note_count",
	ShareLink:      "share_link",
	RawJson:        "raw_json",
	FetchedAt:      "fetched_at",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
}

// NewXhsUserProfilesDao creates and returns a new DAO object for table data access.
func NewXhsUserProfilesDao(handlers ...gdb.ModelHandler) *XhsUserProfilesDao {
	return &XhsUserProfilesDao{
		group:    "default",
		table:    "xhs_user_profiles",
		columns:  xhsUserProfilesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *XhsUserProfilesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *XhsUserProfilesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *XhsUserProfilesDao) Columns() XhsUserProfilesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *XhsUserProfilesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *XhsUserProfilesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *XhsUserProfilesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
