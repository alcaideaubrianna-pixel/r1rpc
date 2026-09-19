// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// ImageJobs is the golang structure for table image_jobs.
type ImageJobs struct {
	Id                         string    `json:"id"                         orm:"id"                           ` //
	BatchId                    string    `json:"batchId"                    orm:"batch_id"                     ` //
	SourceAssetId              string    `json:"sourceAssetId"              orm:"source_asset_id"              ` //
	Source                     string    `json:"source"                     orm:"source"                       ` //
	ExternalId                 string    `json:"externalId"                 orm:"external_id"                  ` //
	Status                     string    `json:"status"                     orm:"status"                       ` //
	Stage                      string    `json:"stage"                      orm:"stage"                        ` //
	Priority                   int       `json:"priority"                   orm:"priority"                     ` //
	ForceRefresh               int       `json:"forceRefresh"               orm:"force_refresh"                ` //
	Attempt                    int       `json:"attempt"                    orm:"attempt"                      ` //
	AssignedClientId           string    `json:"assignedClientId"           orm:"assigned_client_id"           ` //
	AssignedSessionIncarnation string    `json:"assignedSessionIncarnation" orm:"assigned_session_incarnation" ` //
	UploadRequestId            string    `json:"uploadRequestId"            orm:"upload_request_id"            ` //
	SearchRequestId            string    `json:"searchRequestId"            orm:"search_request_id"            ` //
	UploadHandle               string    `json:"uploadHandle"               orm:"upload_handle"                ` //
	CacheSourceJobId           string    `json:"cacheSourceJobId"           orm:"cache_source_job_id"          ` //
	ErrorCode                  string    `json:"errorCode"                  orm:"error_code"                   ` //
	ErrorMessage               string    `json:"errorMessage"               orm:"error_message"                ` //
	InputJson                  string    `json:"inputJson"                  orm:"input_json"                   ` //
	NormalizedResponseJson     string    `json:"normalizedResponseJson"     orm:"normalized_response_json"     ` //
	CreatedAt                  time.Time `json:"createdAt"                  orm:"created_at"                   ` //
	UpdatedAt                  time.Time `json:"updatedAt"                  orm:"updated_at"                   ` //
	StartedAt                  time.Time `json:"startedAt"                  orm:"started_at"                   ` //
	FinishedAt                 time.Time `json:"finishedAt"                 orm:"finished_at"                  ` //
	DeletedAt                  time.Time `json:"deletedAt"                  orm:"deleted_at"                   ` //
}
