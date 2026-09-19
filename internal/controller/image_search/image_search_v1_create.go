package image_search

import (
	"context"

	"r1rpc/api/image_search/v1"
	"r1rpc/internal/model/input"
	"r1rpc/internal/requestctx"
)

func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	requester := requestctx.RequesterFrom(ctx)
	groups := make([]input.ImageSearchGroup, 0, len(req.Groups))
	for _, group := range req.Groups {
		images := make([]input.ImageSearchItem, 0, len(group.Images))
		for _, image := range group.Images {
			images = append(images, input.ImageSearchItem{FileID: image.FileID})
		}
		groups = append(groups, input.ImageSearchGroup{
			ExternalID: group.ExternalID, SubjectUserID: group.SubjectUserID,
			Images: images, MatchPolicy: group.MatchPolicy,
		})
	}
	result, err := c.service.Create(ctx, input.CreateImageSearchRequest{
		Source: "admin", ExternalID: req.ExternalID, Priority: req.Priority,
		PipelineName: req.PipelineName, CallbackURL: req.CallbackURL,
		RequestedByUserID: requester.UserID, RequestedBySubject: requester.Subject,
		Groups: groups,
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateRes{Request: toRequestSummary(*result)}, nil
}
