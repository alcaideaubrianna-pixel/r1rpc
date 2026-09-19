package output

import "time"

// ImageJob 是调用方共享的任务创建结果。
type ImageJob struct {
	ID       string `json:"id"`
	BatchID  string `json:"batchId,omitempty"`
	Status   string `json:"status"`
	CacheHit bool   `json:"cacheHit"`
}

// ImageBatch 是批量任务创建结果。
type ImageBatch struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	Total     int        `json:"total"`
	Jobs      []ImageJob `json:"jobs"`
	Existed   bool       `json:"existed"`
	Queued    int        `json:"queued"`
	Running   int        `json:"running"`
	Completed int        `json:"completed"`
	Failed    int        `json:"failed"`
	CacheHit  int        `json:"cacheHit"`
	CreatedAt time.Time  `json:"createdAt"`
}

type ImageBatchPage struct {
	Items    []ImageBatch `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	Total    int64        `json:"total"`
}
