// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ImageCandidateImagesDao is the data access object for the table image_candidate_images.
type ImageCandidateImagesDao struct {
	table    string                      // table is the underlying table name of the DAO.
	group    string                      // group is the database configuration group name of the current DAO.
	columns  ImageCandidateImagesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler          // handlers for customized model modification.
}

// ImageCandidateImagesColumns defines and stores column names for the table image_candidate_images.
type ImageCandidateImagesColumns struct {
	Id               string //
	CandidateId      string //
	ImageIndex       string //
	SourceUrl        string //
	ImageFileId      string //
	DownloadStatus   string //
	ImageSha256      string //
	Phash            string //
	Dhash            string //
	Ahash            string //
	AlgorithmVersion string //
	Score            string //
	PhashDistance    string //
	DhashDistance    string //
	AhashDistance    string //
	Matched          string //
	ErrorMessage     string //
	CreatedAt        string //
	UpdatedAt        string //
}

// imageCandidateImagesColumns holds the columns for the table image_candidate_images.
var imageCandidateImagesColumns = ImageCandidateImagesColumns{
	Id:               "id",
	CandidateId:      "candidate_id",
	ImageIndex:       "image_index",
	SourceUrl:        "source_url",
	ImageFileId:      "image_file_id",
	DownloadStatus:   "download_status",
	ImageSha256:      "image_sha256",
	Phash:            "phash",
	Dhash:            "dhash",
	Ahash:            "ahash",
	AlgorithmVersion: "algorithm_version",
	Score:            "score",
	PhashDistance:    "phash_distance",
	DhashDistance:    "dhash_distance",
	AhashDistance:    "ahash_distance",
	Matched:          "matched",
	ErrorMessage:     "error_message",
	CreatedAt:        "created_at",
	UpdatedAt:        "updated_at",
}

// NewImageCandidateImagesDao creates and returns a new DAO object for table data access.
func NewImageCandidateImagesDao(handlers ...gdb.ModelHandler) *ImageCandidateImagesDao {
	return &ImageCandidateImagesDao{
		group:    "default",
		table:    "image_candidate_images",
		columns:  imageCandidateImagesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ImageCandidateImagesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ImageCandidateImagesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ImageCandidateImagesDao) Columns() ImageCandidateImagesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ImageCandidateImagesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ImageCandidateImagesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ImageCandidateImagesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
