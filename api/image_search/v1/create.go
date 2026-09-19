package v1

import "github.com/gogf/gf/v2/frame/g"

type CreateReq struct {
	g.Meta       `path:"/api/v1/image-search/requests" method:"post" tags:"ImageSearch" summary:"创建图片组检索请求"`
	ExternalID   string       `json:"externalId" v:"max-length:128" dc:"调用方幂等 ID"`
	Priority     int          `json:"priority" v:"between:-10,10" dc:"任务优先级"`
	PipelineName string       `json:"pipelineName" v:"max-length:64" dc:"服务端已注册的流水线名称"`
	CallbackURL  string       `json:"callbackUrl" v:"url|max-length:1024" dc:"异步结果回调地址"`
	Groups       []GroupInput `json:"groups" v:"required|length:1,1000" dc:"待检索图片组"`
}

type CreateRes struct {
	Request RequestSummary `json:"request"`
}
