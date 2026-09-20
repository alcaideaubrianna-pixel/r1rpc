// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// SearchConfigs is the golang structure of table search_configs for DAO operations like Where/Data.
type SearchConfigs struct {
	g.Meta           `orm:"table:search_configs, do:true"`
	Id               any //
	Name             any //
	OcrEnabled       any //
	OcrKeywordsJson  any //
	OcrMatchMode     any //
	ScoreThreshold   any //
	MaxPhashDistance any //
	MaxDhashDistance any //
	MaxAhashDistance any //
	MaxCandidates    any //
	PipelineName     any //
	Enabled          any //
	Version          any //
	CreatedAt        any //
	UpdatedAt        any //
}
