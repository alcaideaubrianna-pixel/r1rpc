package v1

import "github.com/gogf/gf/v2/frame/g"

type RetryAnalysisReq struct {
	g.Meta `path:"/api/v1/image-search/requests/{id}/retry-analysis" method:"post" tags:"ImageSearch" summary:"使用已保存响应重新分析候选"`
	ID     string `json:"id" in:"path" v:"required|length:32,32"`
}

type RetryAnalysisRes struct {
	OK bool `json:"ok"`
}
