// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"time"
)

// Files is the golang structure for table files.
type Files struct {
	Id           string    `json:"id"           orm:"id"            ` //
	ObjectKey    string    `json:"objectKey"    orm:"object_key"    ` //
	OriginalName string    `json:"originalName" orm:"original_name" ` //
	ContentType  string    `json:"contentType"  orm:"content_type"  ` //
	SizeBytes    int64     `json:"sizeBytes"    orm:"size_bytes"    ` //
	Sha256       string    `json:"sha256"       orm:"sha256"        ` //
	Backend      string    `json:"backend"      orm:"backend"       ` //
	Status       string    `json:"status"       orm:"status"        ` //
	CreatedBy    int64     `json:"createdBy"    orm:"created_by"    ` //
	CreatedAt    time.Time `json:"createdAt"    orm:"created_at"    ` //
	ExpiresAt    time.Time `json:"expiresAt"    orm:"expires_at"    ` //
	DeletedAt    time.Time `json:"deletedAt"    orm:"deleted_at"    ` //
}
