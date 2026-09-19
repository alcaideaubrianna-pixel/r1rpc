package storage

import (
	"context"
	"io"
)

type Object struct {
	Key         string
	ContentType string
	Size        int64
}

// Backend 只负责对象读写和地址解析，业务层不感知本地磁盘、COS 或 OSS。
type Backend interface {
	Name() string
	Put(ctx context.Context, object Object, source io.Reader) error
	Open(ctx context.Context, objectKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectKey string) error
	URL(ctx context.Context, objectKey string) (string, error)
}
