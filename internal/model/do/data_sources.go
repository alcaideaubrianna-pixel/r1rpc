// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// DataSources is the golang structure of table data_sources for DAO operations like Where/Data.
type DataSources struct {
	g.Meta                `orm:"table:data_sources, do:true"`
	Id                    any //
	Name                  any //
	BaseUrl               any //
	AppId                 any //
	AccessKey             any //
	SecretKeyEncrypted    any //
	Status                any //
	RequestTimeoutSeconds any //
	CreatedAt             any //
	UpdatedAt             any //
}
