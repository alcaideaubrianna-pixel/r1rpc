// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SourceScanTaskNotesDao is the data access object for the table source_scan_task_notes.
type SourceScanTaskNotesDao struct {
	table    string                     // table is the underlying table name of the DAO.
	group    string                     // group is the database configuration group name of the current DAO.
	columns  SourceScanTaskNotesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler         // handlers for customized model modification.
}

// SourceScanTaskNotesColumns defines and stores column names for the table source_scan_task_notes.
type SourceScanTaskNotesColumns struct {
	Id         string //
	ScanTaskId string //
	NoteId     string //
	CreatedAt  string //
}

// sourceScanTaskNotesColumns holds the columns for the table source_scan_task_notes.
var sourceScanTaskNotesColumns = SourceScanTaskNotesColumns{
	Id:         "id",
	ScanTaskId: "scan_task_id",
	NoteId:     "note_id",
	CreatedAt:  "created_at",
}

// NewSourceScanTaskNotesDao creates and returns a new DAO object for table data access.
func NewSourceScanTaskNotesDao(handlers ...gdb.ModelHandler) *SourceScanTaskNotesDao {
	return &SourceScanTaskNotesDao{
		group:    "default",
		table:    "source_scan_task_notes",
		columns:  sourceScanTaskNotesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SourceScanTaskNotesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SourceScanTaskNotesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SourceScanTaskNotesDao) Columns() SourceScanTaskNotesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SourceScanTaskNotesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SourceScanTaskNotesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SourceScanTaskNotesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
