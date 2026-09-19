package output

import "time"

type ImageSearchRequest struct {
	ID              string             `json:"id"`
	ExternalID      string             `json:"externalId,omitempty"`
	Status          string             `json:"status"`
	PipelineName    string             `json:"pipelineName"`
	PipelineVersion string             `json:"pipelineVersion"`
	GroupCount      int                `json:"groupCount"`
	MatchedCount    int                `json:"matchedCount"`
	NotMatchedCount int                `json:"notMatchedCount"`
	RunningCount    int                `json:"runningCount"`
	FailedCount     int                `json:"failedCount"`
	CreatedAt       time.Time          `json:"createdAt"`
	Groups          []ImageSearchGroup `json:"groups,omitempty"`
	Items           []ImageSearchItem  `json:"items,omitempty"`
	Existed         bool               `json:"-"`
}

type ImageSearchGroup struct {
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

type ImageSearchItem struct {
	ID           string
	GroupID      string
	FileID       string
	FileURL      string
	Ordinal      int
	Status       string
	ErrorCode    string
	ErrorMessage string
}

type ImageSearchRequestPage struct {
	Items    []ImageSearchRequest `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
	Total    int                  `json:"total"`
}

type ImageCandidate struct {
	ID               string
	SearchItemID     string
	Rank             int
	ContentID        string
	Title            string
	AuthorID         string
	AuthorName       string
	CoverURL         string
	ImageFileID      string
	ImageURL         string
	DownloadStatus   string
	ErrorMessage     string
	AlgorithmVersion string
	PHashDistance    *int
	DHashDistance    *int
	AHashDistance    *int
	Score            float64
	Matched          bool
	ImageCount       int
	AnalyzedImages   int
	FailedImages     int
	Images           []CandidateImage
}

type CandidateImage struct {
	ImageIndex     int
	ImageFileID    string
	ImageURL       string
	DownloadStatus string
	ErrorMessage   string
	Score          float64
	Matched        bool
	PHashDistance  *int
	DHashDistance  *int
	AHashDistance  *int
}

type ImageCandidatePage struct {
	Items    []ImageCandidate
	Page     int
	PageSize int
	Total    int
}

type ImageSearchResponse struct {
	ID             string
	JobID          string
	RequestID      string
	ItemCount      int
	ParseStatus    string
	ErrorMessage   string
	NormalizedJSON string
	RawJSON        string
	CreatedAt      time.Time
}
