package files

import (
	"bytes"
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
	"time"

	"r1rpc/internal/dao"
	"r1rpc/internal/model"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"
	"r1rpc/internal/storage"

	"github.com/gogf/gf/v2/errors/gerror"
)

const MaxImageBytes int64 = 12 << 20

type Service struct {
	backend storage.Backend
}

func New(backend storage.Backend) *Service {
	return &Service{backend: backend}
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
	tmp, err := os.CreateTemp("", ".r1rpc-upload-*")
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
	storedSource, err := os.Open(tmpName)
	if err != nil {
		return nil, err
	}
	defer storedSource.Close()
	if err := s.backend.Put(ctx, storage.Object{Key: objectKey, ContentType: contentType, Size: written}, storedSource); err != nil {
		return nil, err
	}
	item := model.File{
		ID: id, ObjectKey: objectKey, OriginalName: filepath.Base(header.Filename),
		ContentType: contentType, SizeBytes: written, SHA256: hex.EncodeToString(hash.Sum(nil)),
		Backend: s.backend.Name(), Status: "active", CreatedBy: createdBy,
	}
	if _, err := dao.Files.Ctx(ctx).Data(do.Files{
		Id: item.ID, ObjectKey: item.ObjectKey, OriginalName: item.OriginalName,
		ContentType: item.ContentType, SizeBytes: item.SizeBytes, Sha256: item.SHA256,
		Backend: item.Backend, Status: item.Status, CreatedBy: item.CreatedBy,
	}).Insert(); err != nil {
		_ = s.backend.Delete(ctx, objectKey)
		return nil, err
	}
	return &item, nil
}

// SaveBytes 将内部任务产生的图片写入统一文件存储。调用方只保存文件 ID，便于后续替换 COS/OSS 后端。
func (s *Service) SaveBytes(ctx context.Context, data []byte, originalName string, createdBy int64) (*model.File, error) {
	if len(data) == 0 || int64(len(data)) > MaxImageBytes {
		return nil, fmt.Errorf("图片大小必须在 1 字节到 12 MiB 之间")
	}
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	contentType, extension, err := detectImageBytes(data)
	if err != nil {
		return nil, err
	}
	objectKey := id + extension
	if err = s.backend.Put(ctx, storage.Object{Key: objectKey, ContentType: contentType, Size: int64(len(data))}, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	item := &model.File{
		ID: id, ObjectKey: objectKey, OriginalName: filepath.Base(originalName),
		ContentType: contentType, SizeBytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:]),
		Backend: s.backend.Name(), Status: "active", CreatedBy: createdBy,
	}
	if _, err = dao.Files.Ctx(ctx).Data(do.Files{
		Id: item.ID, ObjectKey: item.ObjectKey, OriginalName: item.OriginalName,
		ContentType: item.ContentType, SizeBytes: item.SizeBytes, Sha256: item.SHA256,
		Backend: item.Backend, Status: item.Status, CreatedBy: item.CreatedBy,
	}).Insert(); err != nil {
		_ = s.backend.Delete(ctx, objectKey)
		return nil, gerror.Wrap(err, "保存内部图片文件记录失败")
	}
	return item, nil
}

func (s *Service) Open(ctx context.Context, id string) (*model.File, io.ReadCloser, error) {
	item, err := s.get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	file, err := s.backend.Open(ctx, item.ObjectKey)
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
	item, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.backend.Delete(ctx, item.ObjectKey); err != nil {
		return err
	}
	_, err = dao.Files.Ctx(ctx).Where(dao.Files.Columns().Id, id).
		Data(do.Files{Status: "deleted"}).Delete()
	return err
}

func (s *Service) URL(ctx context.Context, id string) (string, error) {
	item, err := s.get(ctx, id)
	if err != nil {
		return "", err
	}
	URL, err := s.backend.URL(ctx, item.ObjectKey)
	if err != nil {
		return "", err
	}
	if URL != "" {
		return URL, nil
	}
	return "/api/files/" + item.ID + "/content", nil
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

func detectImageBytes(data []byte) (string, string, error) {
	if len(data) > 32 {
		data = data[:32]
	}
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg", ".jpg", nil
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png", ".png", nil
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		brand := string(data[8:12])
		if strings.HasPrefix(brand, "hei") || brand == "mif1" || brand == "msf1" {
			return "image/heic", ".heic", nil
		}
	}
	return "", "", fmt.Errorf("仅支持 JPEG、PNG、HEIC 或 HEIF 图片")
}

func (s *Service) get(ctx context.Context, id string) (*model.File, error) {
	var record entity.Files
	columns := dao.Files.Columns()
	if err := dao.Files.Ctx(ctx).Where(columns.Id, strings.TrimSpace(id)).Where(columns.Status, "active").Scan(&record); err != nil {
		return nil, err
	}
	if record.Id == "" {
		return nil, sql.ErrNoRows
	}
	return &model.File{
		ID: record.Id, ObjectKey: record.ObjectKey, OriginalName: record.OriginalName,
		ContentType: record.ContentType, SizeBytes: record.SizeBytes, SHA256: record.Sha256,
		Backend: record.Backend, Status: record.Status, CreatedBy: record.CreatedBy,
		CreatedAt: record.CreatedAt, ExpiresAt: timePointer(record.ExpiresAt), DeletedAt: timePointer(record.DeletedAt),
	}, nil
}

func timePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
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
