// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SourceNoteImagesDao is the data access object for the table source_note_images.
type SourceNoteImagesDao struct {
	table    string                  // table is the underlying table name of the DAO.
	group    string                  // group is the database configuration group name of the current DAO.
	columns  SourceNoteImagesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler      // handlers for customized model modification.
}

// SourceNoteImagesColumns defines and stores column names for the table source_note_images.
type SourceNoteImagesColumns struct {
	Id               string //
	NoteId           string //
	ExternalAssetId  string //
	AssetType        string //
	SourceUrl        string //
	FileId           string //
	Sha256           string //
	Phash            string //
	DownloadStatus   string //
	PreprocessStatus string //
	OcrText          string //
	FilterDecision   string //
	FilterReason     string //
	RawJson          string //
	ImageIndex       string //
	CreatedAt        string //
	UpdatedAt        string //
}

// sourceNoteImagesColumns holds the columns for the table source_note_images.
var sourceNoteImagesColumns = SourceNoteImagesColumns{
	Id:               "id",
	NoteId:           "note_id",
	ExternalAssetId:  "external_asset_id",
	AssetType:        "asset_type",
	SourceUrl:        "source_url",
	FileId:           "file_id",
	Sha256:           "sha256",
	Phash:            "phash",
	DownloadStatus:   "download_status",
	PreprocessStatus: "preprocess_status",
	OcrText:          "ocr_text",
	FilterDecision:   "filter_decision",
	FilterReason:     "filter_reason",
	RawJson:          "raw_json",
	ImageIndex:       "image_index",
	CreatedAt:        "created_at",
	UpdatedAt:        "updated_at",
}

// NewSourceNoteImagesDao creates and returns a new DAO object for table data access.
func NewSourceNoteImagesDao(handlers ...gdb.ModelHandler) *SourceNoteImagesDao {
	return &SourceNoteImagesDao{
		group:    "default",
		table:    "source_note_images",
		columns:  sourceNoteImagesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SourceNoteImagesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SourceNoteImagesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SourceNoteImagesDao) Columns() SourceNoteImagesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SourceNoteImagesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SourceNoteImagesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SourceNoteImagesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
