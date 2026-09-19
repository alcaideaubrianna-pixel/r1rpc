// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// StorageSettings is the golang structure of table storage_settings for DAO operations like Where/Data.
type StorageSettings struct {
	g.Meta             `orm:"table:storage_settings, do:true"`
	Id                 any //
	Backend            any //
	LocalPath          any //
	Endpoint           any //
	Region             any //
	Bucket             any //
	PathStyle          any //
	AccessKeyEncrypted any //
	SecretKeyEncrypted any //
	UpdatedAt          any //
}
