package v1

import "github.com/gogf/gf/v2/frame/g"

type GetResponsesReq struct {
	g.Meta `path:"/api/v1/image-search/requests/{id}/responses" method:"get" tags:"ImageSearch" summary:"查询设备图片搜索完整响应"`
	ID     string `json:"id" in:"path" v:"required|length:32,32"`
}

type GetResponsesRes struct {
	Items []ResponseSummary `json:"items"`
}
