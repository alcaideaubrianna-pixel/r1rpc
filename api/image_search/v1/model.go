package v1

type ImageInput struct {
	FileID string `json:"fileId" v:"required|length:32,32" dc:"已上传的图片文件 ID"`
}

type GroupInput struct {
	ExternalID    string         `json:"externalId" v:"max-length:128" dc:"调用方图片组 ID"`
	SubjectUserID string         `json:"subjectUserId" v:"max-length:128" dc:"图片组关联的业务用户 ID"`
	Images        []ImageInput   `json:"images" v:"required|length:1,100" dc:"组内图片；任一图片命中即组命中"`
	MatchPolicy   map[string]any `json:"matchPolicy" dc:"匹配策略覆盖项"`
}

type RequestSummary struct {
	ID              string `json:"id"`
	ExternalID      string `json:"externalId,omitempty"`
	Status          string `json:"status"`
	PipelineName    string `json:"pipelineName"`
	PipelineVersion string `json:"pipelineVersion"`
	GroupCount      int    `json:"groupCount"`
	MatchedCount    int    `json:"matchedCount"`
	NotMatchedCount int    `json:"notMatchedCount"`
	RunningCount    int    `json:"runningCount"`
	FailedCount     int    `json:"failedCount"`
	CreatedAt       string `json:"createdAt"`
}

type GroupSummary struct {
	ID             string   `json:"id"`
	ExternalID     string   `json:"externalId,omitempty"`
	SubjectUserID  string   `json:"subjectUserId,omitempty"`
	Status         string   `json:"status"`
	ItemCount      int      `json:"itemCount"`
	QueuedCount    int      `json:"queuedCount"`
	RunningCount   int      `json:"runningCount"`
	CompletedCount int      `json:"completedCount"`
	FailedCount    int      `json:"failedCount"`
	BestScore      *float64 `json:"bestScore,omitempty"`
	BestMatchID    string   `json:"bestMatchId,omitempty"`
}

type ItemSummary struct {
	ID           string `json:"id"`
	GroupID      string `json:"groupId"`
	FileID       string `json:"fileId"`
	FileURL      string `json:"fileUrl"`
	Ordinal      int    `json:"ordinal"`
	Status       string `json:"status"`
	JobID        string `json:"jobId,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type SourceImageSummary struct {
	ID               string `json:"id"`
	GroupID          string `json:"groupId"`
	SearchItemID     string `json:"searchItemId,omitempty"`
	FileID           string `json:"fileId,omitempty"`
	FileURL          string `json:"fileUrl,omitempty"`
	ImageIndex       int    `json:"imageIndex"`
	DownloadStatus   string `json:"downloadStatus"`
	PreprocessStatus string `json:"preprocessStatus,omitempty"`
	FilterDecision   string `json:"filterDecision,omitempty"`
	FilterReason     string `json:"filterReason,omitempty"`
}

type CandidateSummary struct {
	ID               string                  `json:"id"`
	SearchItemID     string                  `json:"searchItemId"`
	Rank             int                     `json:"rank"`
	ContentID        string                  `json:"contentId"`
	Title            string                  `json:"title,omitempty"`
	AuthorID         string                  `json:"authorId,omitempty"`
	AuthorName       string                  `json:"authorName,omitempty"`
	CoverURL         string                  `json:"coverUrl,omitempty"`
	ImageFileID      string                  `json:"imageFileId,omitempty"`
	ImageURL         string                  `json:"imageUrl,omitempty"`
	DownloadStatus   string                  `json:"downloadStatus"`
	ErrorMessage     string                  `json:"errorMessage,omitempty"`
	AlgorithmVersion string                  `json:"algorithmVersion,omitempty"`
	PHashDistance    *int                    `json:"phashDistance,omitempty"`
	DHashDistance    *int                    `json:"dhashDistance,omitempty"`
	AHashDistance    *int                    `json:"ahashDistance,omitempty"`
	Score            float64                 `json:"score"`
	Matched          bool                    `json:"matched"`
	ImageCount       int                     `json:"imageCount"`
	AnalyzedImages   int                     `json:"analyzedImages"`
	FailedImages     int                     `json:"failedImages"`
	Images           []CandidateImageSummary `json:"images"`
}

type CandidateImageSummary struct {
	ImageIndex     int     `json:"imageIndex"`
	ImageFileID    string  `json:"imageFileId,omitempty"`
	ImageURL       string  `json:"imageUrl,omitempty"`
	DownloadStatus string  `json:"downloadStatus"`
	ErrorMessage   string  `json:"errorMessage,omitempty"`
	Score          float64 `json:"score"`
	Matched        bool    `json:"matched"`
	PHashDistance  *int    `json:"phashDistance,omitempty"`
	DHashDistance  *int    `json:"dhashDistance,omitempty"`
	AHashDistance  *int    `json:"ahashDistance,omitempty"`
}

type ResponseSummary struct {
	ID             string `json:"id"`
	JobID          string `json:"jobId"`
	RequestID      string `json:"requestId"`
	ItemCount      int    `json:"itemCount"`
	ParseStatus    string `json:"parseStatus"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	NormalizedJSON string `json:"normalizedJson"`
	RawJSON        string `json:"rawJson,omitempty"`
	CreatedAt      string `json:"createdAt"`
}
