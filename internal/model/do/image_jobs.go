// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ImageJobs is the golang structure of table image_jobs for DAO operations like Where/Data.
type ImageJobs struct {
	g.Meta                     `orm:"table:image_jobs, do:true"`
	Id                         any //
	BatchId                    any //
	SourceAssetId              any //
	Source                     any //
	ExternalId                 any //
	Status                     any //
	Stage                      any //
	Priority                   any //
	ForceRefresh               any //
	Attempt                    any //
	AssignedClientId           any //
	AssignedSessionIncarnation any //
	UploadRequestId            any //
	SearchRequestId            any //
	UploadHandle               any //
	CacheSourceJobId           any //
	ErrorCode                  any //
	ErrorMessage               any //
	InputJson                  any //
	NormalizedResponseJson     any //
	CreatedAt                  any //
	UpdatedAt                  any //
	StartedAt                  any //
	FinishedAt                 any //
	DeletedAt                  any //
}
