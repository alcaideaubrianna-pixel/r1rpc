package input

type ImageSearchItem struct {
	FileID string
}

type ImageSearchGroup struct {
	ExternalID    string
	SubjectUserID string
	Images        []ImageSearchItem
	MatchPolicy   map[string]any
}

type CreateImageSearchRequest struct {
	Source             string
	ExternalID         string
	Priority           int
	PipelineName       string
	CallbackURL        string
	RequestedByUserID  int64
	RequestedBySubject string
	Groups             []ImageSearchGroup
	DeferQueue         bool
	InitialJobStage    string
}
