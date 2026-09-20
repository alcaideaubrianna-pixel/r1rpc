// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// DataSources is the golang structure for table data_sources.
type DataSources struct {
	Id                    string    `json:"id"                    orm:"id"                      ` //
	Name                  string    `json:"name"                  orm:"name"                    ` //
	BaseUrl               string    `json:"baseUrl"               orm:"base_url"                ` //
	ImageBaseUrl          string    `json:"imageBaseUrl"          orm:"image_base_url"          ` //
	AppId                 string    `json:"appId"                 orm:"app_id"                  ` //
	AccessKey             string    `json:"accessKey"             orm:"access_key"              ` //
	SecretKeyEncrypted    string    `json:"secretKeyEncrypted"    orm:"secret_key_encrypted"    ` //
	Status                string    `json:"status"                orm:"status"                  ` //
	RequestTimeoutSeconds int       `json:"requestTimeoutSeconds" orm:"request_timeout_seconds" ` //
	CreatedAt             time.Time `json:"createdAt"             orm:"created_at"              ` //
	UpdatedAt             time.Time `json:"updatedAt"             orm:"updated_at"              ` //
}
