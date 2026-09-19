package v1

import "github.com/gogf/gf/v2/frame/g"

type GetCandidatesReq struct {
	g.Meta   `path:"/api/v1/image-search/requests/{id}/candidates" method:"get" tags:"ImageSearch" summary:"分页查询图片匹配候选"`
	ID       string `json:"id" in:"path" v:"required|length:32,32"`
	Page     int    `json:"page" d:"1" v:"min:1"`
	PageSize int    `json:"pageSize" d:"20" v:"between:1,100"`
	Matched  *bool  `json:"matched" dc:"仅查询命中或未命中候选"`
}

type GetCandidatesRes struct {
	Items    []CandidateSummary `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	Total    int                `json:"total"`
}
