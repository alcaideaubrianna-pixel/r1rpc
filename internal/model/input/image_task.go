package input

// CreateImageJob 创建单图识别任务的内部输入，不绑定 HTTP、Telegram 或队列协议。
type CreateImageJob struct {
	Source       string
	ExternalID   string
	FileID       string
	Priority     int
	ForceRefresh bool
	RequestedBy  any
}

// CreateImageBatch 创建批量识别任务的内部输入。
type CreateImageBatch struct {
	Source       string
	ExternalID   string
	FileIDs      []string
	Priority     int
	ForceRefresh bool
	RequestedBy  any
}
