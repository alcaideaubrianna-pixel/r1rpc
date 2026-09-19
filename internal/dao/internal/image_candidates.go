// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageCandidatesDao is the data access object for the table image_candidates.
type ImageCandidatesDao struct {
	table    string                 // table is the underlying table name of the DAO.
	group    string                 // group is the database configuration group name of the current DAO.
	columns  ImageCandidatesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler     // handlers for customized model modification.
}

// ImageCandidatesColumns defines and stores column names for the table image_candidates.
type ImageCandidatesColumns struct {
	Id               string //
	ResponseId       string //
	JobId            string //
	SearchItemId     string //
	RankNo           string //
	ContentId        string //
	Title            string //
	AuthorId         string //
	AuthorName       string //
	CoverUrl         string //
	RawItemJson      string //
	DownloadStatus   string //
	ImageSha256      string //
	Phash            string //
	Dhash            string //
	Ahash            string //
	AlgorithmVersion string //
	Score            string //
	Matched          string //
	ErrorMessage     string //
	CreatedAt        string //
	UpdatedAt        string //
	ImageFileId      string //
}

// imageCandidatesColumns holds the columns for the table image_candidates.
var imageCandidatesColumns = ImageCandidatesColumns{
	Id:               "id",
	ResponseId:       "response_id",
	JobId:            "job_id",
	SearchItemId:     "search_item_id",
	RankNo:           "rank_no",
	ContentId:        "content_id",
	Title:            "title",
	AuthorId:         "author_id",
	AuthorName:       "author_name",
	CoverUrl:         "cover_url",
	RawItemJson:      "raw_item_json",
	DownloadStatus:   "download_status",
	ImageSha256:      "image_sha256",
	Phash:            "phash",
	Dhash:            "dhash",
	Ahash:            "ahash",
	AlgorithmVersion: "algorithm_version",
	Score:            "score",
	Matched:          "matched",
	ErrorMessage:     "error_message",
	CreatedAt:        "created_at",
	UpdatedAt:        "updated_at",
	ImageFileId:      "image_file_id",
}

// NewImageCandidatesDao creates and returns a new DAO object for table data access.
func NewImageCandidatesDao(handlers ...gdb.ModelHandler) *ImageCandidatesDao {
	return &ImageCandidatesDao{
		group:    "default",
		table:    "image_candidates",
		columns:  imageCandidatesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageCandidatesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageCandidatesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageCandidatesDao) Columns() ImageCandidatesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageCandidatesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageCandidatesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageCandidatesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
