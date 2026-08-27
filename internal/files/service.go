package files

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"r1rpc/internal/model"
	"r1rpc/internal/store"
)

const MaxImageBytes int64 = 12 << 20

type Service struct {
	root  string
	store *store.Store
}

func New(root string, st *store.Store) (*Service, error) {
	root, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &Service{root: root, store: st}, nil
}

func (s *Service) Save(ctx context.Context, header *multipart.FileHeader, createdBy int64) (*model.File, error) {
	if header == nil {
		return nil, fmt.Errorf("请选择图片")
	}
	source, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer source.Close()

	id, err := randomID()
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(source, MaxImageBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if written == 0 || written > MaxImageBytes {
		return nil, fmt.Errorf("图片大小必须在 1 字节到 12 MiB 之间")
	}

	contentType, extension, err := detectImage(tmpName)
	if err != nil {
		return nil, err
	}
	objectKey := id + extension
	destination := filepath.Join(s.root, objectKey)
	if err := os.Rename(tmpName, destination); err != nil {
		return nil, err
	}
	item := model.File{
		ID: id, ObjectKey: objectKey, OriginalName: filepath.Base(header.Filename),
		ContentType: contentType, SizeBytes: written, SHA256: hex.EncodeToString(hash.Sum(nil)),
		Backend: "local", Status: "active", CreatedBy: createdBy,
	}
	if err := s.store.CreateFile(ctx, item); err != nil {
		_ = os.Remove(destination)
		return nil, err
	}
	return s.store.GetFile(ctx, id)
}

func (s *Service) Open(ctx context.Context, id string) (*model.File, *os.File, error) {
	item, err := s.store.GetFile(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, nil, err
	}
	path, err := s.path(item.ObjectKey)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return item, file, nil
}

func (s *Service) Read(ctx context.Context, id string) (*model.File, []byte, error) {
	item, file, err := s.Open(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxImageBytes+1))
	if err != nil || int64(len(data)) > MaxImageBytes {
		return nil, nil, fmt.Errorf("读取图片失败或图片超过 12 MiB")
	}
	return item, data, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	item, err := s.store.GetFile(ctx, id)
	if err != nil {
		return err
	}
	path, err := s.path(item.ObjectKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return s.store.SoftDeleteFile(ctx, id)
}

func (s *Service) path(objectKey string) (string, error) {
	if objectKey == "" || filepath.Base(objectKey) != objectKey {
		return "", fmt.Errorf("非法文件路径")
	}
	return filepath.Join(s.root, objectKey), nil
}

func detectImage(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	header := make([]byte, 32)
	n, _ := io.ReadFull(file, header)
	header = header[:n]
	switch {
	case len(header) >= 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff:
		return "image/jpeg", ".jpg", nil
	case len(header) >= 8 && string(header[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png", ".png", nil
	case len(header) >= 12 && string(header[4:8]) == "ftyp":
		brand := string(header[8:12])
		if strings.HasPrefix(brand, "hei") || brand == "mif1" || brand == "msf1" {
			return "image/heic", ".heic", nil
		}
	}
	return "", "", fmt.Errorf("仅支持 JPEG、PNG、HEIC 或 HEIF 图片")
}

func randomID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, os.ErrNotExist)
}
