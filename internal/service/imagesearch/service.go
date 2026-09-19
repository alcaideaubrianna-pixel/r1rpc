package imagesearch

import (
	"sync"

	"r1rpc/internal/files"
)

const (
	defaultPipelineName    = "xhs-image-match"
	defaultPipelineVersion = "v1"
	statusCreated          = "created"
)

type Service struct {
	queueMu  sync.RWMutex
	enqueuer Enqueuer
	files    *files.Service
}

func New(fileServices ...*files.Service) *Service {
	service := &Service{}
	if len(fileServices) > 0 {
		service.files = fileServices[0]
	}
	return service
}
