// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SourceNotesDao is the data access object for the table source_notes.
type SourceNotesDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SourceNotesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SourceNotesColumns defines and stores column names for the table source_notes.
type SourceNotesColumns struct {
	Id             string //
	DataSourceId   string //
	ScanTaskId     string //
	ChannelId      string //
	ExternalNoteId string //
	NoteCode       string //
	Title          string //
	PlainText      string //
	AttributesJson string //
	RawJson        string //
	Status         string //
	CreatedAt      string //
	UpdatedAt      string //
}

// sourceNotesColumns holds the columns for the table source_notes.
var sourceNotesColumns = SourceNotesColumns{
	Id:             "id",
	DataSourceId:   "data_source_id",
	ScanTaskId:     "scan_task_id",
	ChannelId:      "channel_id",
	ExternalNoteId: "external_note_id",
	NoteCode:       "note_code",
	Title:          "title",
	PlainText:      "plain_text",
	AttributesJson: "attributes_json",
	RawJson:        "raw_json",
	Status:         "status",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
}

// NewSourceNotesDao creates and returns a new DAO object for table data access.
func NewSourceNotesDao(handlers ...gdb.ModelHandler) *SourceNotesDao {
	return &SourceNotesDao{
		group:    "default",
		table:    "source_notes",
		columns:  sourceNotesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SourceNotesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SourceNotesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SourceNotesDao) Columns() SourceNotesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SourceNotesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SourceNotesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SourceNotesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
