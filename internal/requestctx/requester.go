package requestctx

import "context"

type requesterKey struct{}

type Requester struct {
	UserID  int64
	Subject string
}

func WithRequester(ctx context.Context, requester Requester) context.Context {
	return context.WithValue(ctx, requesterKey{}, requester)
}

func RequesterFrom(ctx context.Context) Requester {
	requester, _ := ctx.Value(requesterKey{}).(Requester)
	return requester
}
