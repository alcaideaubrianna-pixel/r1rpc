// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// StorageSettings is the golang structure for table storage_settings.
type StorageSettings struct {
	Id                 int       `json:"id"                 orm:"id"                   ` //
	Backend            string    `json:"backend"            orm:"backend"              ` //
	LocalPath          string    `json:"localPath"          orm:"local_path"           ` //
	Endpoint           string    `json:"endpoint"           orm:"endpoint"             ` //
	Region             string    `json:"region"             orm:"region"               ` //
	Bucket             string    `json:"bucket"             orm:"bucket"               ` //
	PathStyle          int       `json:"pathStyle"          orm:"path_style"           ` //
	AccessKeyEncrypted string    `json:"accessKeyEncrypted" orm:"access_key_encrypted" ` //
	SecretKeyEncrypted string    `json:"secretKeyEncrypted" orm:"secret_key_encrypted" ` //
	UpdatedAt          time.Time `json:"updatedAt"          orm:"updated_at"           ` //
}
