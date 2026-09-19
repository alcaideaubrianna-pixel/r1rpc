package storagesetting

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
)

type Service struct{ key [32]byte }

type Setting struct {
	Backend             string `json:"backend"`
	LocalPath           string `json:"localPath"`
	Endpoint            string `json:"endpoint"`
	Region              string `json:"region"`
	Bucket              string `json:"bucket"`
	PathStyle           bool   `json:"pathStyle"`
	AccessKeyConfigured bool   `json:"accessKeyConfigured"`
	SecretKeyConfigured bool   `json:"secretKeyConfigured"`
	RestartRequired     bool   `json:"restartRequired"`
	DriverAvailable     bool   `json:"driverAvailable"`
}

type SaveInput struct {
	Backend   string `json:"backend"`
	LocalPath string `json:"localPath"`
	Endpoint  string `json:"endpoint"`
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
	PathStyle bool   `json:"pathStyle"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

func New(secret string) *Service { return &Service{key: sha256.Sum256([]byte(secret))} }

func (s *Service) Get(ctx context.Context) (*Setting, error) {
	var record entity.StorageSettings
	if err := dao.StorageSettings.Ctx(ctx).Where(dao.StorageSettings.Columns().Id, 1).Scan(&record); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &Setting{Backend: "local", LocalPath: "./data/files", PathStyle: true, DriverAvailable: true}, nil
		}
		return nil, gerror.Wrap(err, "读取存储配置失败")
	}
	if record.Id == 0 {
		return &Setting{Backend: "local", LocalPath: "./data/files", PathStyle: true, DriverAvailable: true}, nil
	}
	return toSetting(record), nil
}

func (s *Service) Save(ctx context.Context, input SaveInput) (*Setting, error) {
	backend := strings.ToLower(strings.TrimSpace(input.Backend))
	if backend != "local" && backend != "cos" && backend != "oss" {
		return nil, gerror.New("存储类型必须是 local、cos 或 oss")
	}
	if backend == "local" && strings.TrimSpace(input.LocalPath) == "" {
		return nil, gerror.New("Local 存储目录不能为空")
	}
	if backend != "local" && (strings.TrimSpace(input.Endpoint) == "" || strings.TrimSpace(input.Bucket) == "") {
		return nil, gerror.New("云存储 endpoint 和 bucket 不能为空")
	}
	current, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	accessKey, secretKey := input.AccessKey, input.SecretKey
	if accessKey == "" && current.AccessKeyConfigured {
		accessKey = "__KEEP__"
	}
	if secretKey == "" && current.SecretKeyConfigured {
		secretKey = "__KEEP__"
	}
	data := do.StorageSettings{Id: 1, Backend: backend, LocalPath: strings.TrimSpace(input.LocalPath),
		Endpoint: strings.TrimSpace(input.Endpoint), Region: strings.TrimSpace(input.Region),
		Bucket: strings.TrimSpace(input.Bucket), PathStyle: input.PathStyle}
	if accessKey != "__KEEP__" {
		value, encryptErr := s.encrypt(accessKey)
		if encryptErr != nil {
			return nil, encryptErr
		}
		data.AccessKeyEncrypted = value
	}
	if secretKey != "__KEEP__" {
		value, encryptErr := s.encrypt(secretKey)
		if encryptErr != nil {
			return nil, encryptErr
		}
		data.SecretKeyEncrypted = value
	}
	_, err = dao.StorageSettings.Ctx(ctx).Data(data).OnConflict(dao.StorageSettings.Columns().Id).Save()
	if err != nil {
		return nil, gerror.Wrap(err, "保存存储配置失败")
	}
	return s.Get(ctx)
}

func (s *Service) encrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func toSetting(record entity.StorageSettings) *Setting {
	return &Setting{Backend: record.Backend, LocalPath: record.LocalPath, Endpoint: record.Endpoint,
		Region: record.Region, Bucket: record.Bucket, PathStyle: record.PathStyle != 0,
		AccessKeyConfigured: record.AccessKeyEncrypted != "", SecretKeyConfigured: record.SecretKeyEncrypted != "",
		RestartRequired: true, DriverAvailable: record.Backend == "local"}
}
