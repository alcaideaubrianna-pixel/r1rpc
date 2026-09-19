package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"r1rpc/internal/storage"
)

type Backend struct {
	root string
}

func New(root string) (*Backend, error) {
	root, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &Backend{root: root}, nil
}

func (b *Backend) Name() string { return "local" }

func (b *Backend) Put(_ context.Context, object storage.Object, source io.Reader) error {
	path, err := b.path(object.Key)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(b.root, ".object-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err = io.Copy(temporary, source); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func (b *Backend) Open(_ context.Context, objectKey string) (io.ReadCloser, error) {
	path, err := b.path(objectKey)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (b *Backend) Delete(_ context.Context, objectKey string) error {
	path, err := b.path(objectKey)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (b *Backend) URL(context.Context, string) (string, error) { return "", nil }

func (b *Backend) path(objectKey string) (string, error) {
	if objectKey == "" || filepath.Base(objectKey) != objectKey {
		return "", fmt.Errorf("非法对象路径")
	}
	return filepath.Join(b.root, objectKey), nil
}
