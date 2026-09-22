package v1

import "github.com/gogf/gf/v2/frame/g"

type GetOneReq struct {
	g.Meta `path:"/api/v1/image-search/requests/{id}" method:"get" tags:"ImageSearch" summary:"查询图片组检索请求"`
	ID     string `json:"id" in:"path" v:"required|length:32,32"`
}

type GetOneRes struct {
	Request      RequestSummary       `json:"request"`
	Groups       []GroupSummary       `json:"groups"`
	Items        []ItemSummary        `json:"items"`
	SourceImages []SourceImageSummary `json:"sourceImages"`
}
