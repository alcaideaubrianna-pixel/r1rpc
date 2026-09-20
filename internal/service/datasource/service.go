package datasource

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"r1rpc/internal/dao"
	"r1rpc/internal/datasource/feiniu"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/guid"
)

type Service struct{ key [32]byte }
type Source struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	BaseURL          string    `json:"baseUrl"`
	AppID            string    `json:"appId"`
	AccessKey        string    `json:"accessKey"`
	Status           string    `json:"status"`
	SecretConfigured bool      `json:"secretConfigured"`
	CreatedAt        time.Time `json:"createdAt"`
}
type SaveInput struct {
	Name      string `json:"name"`
	BaseURL   string `json:"baseUrl"`
	AppID     string `json:"appId"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Status    string `json:"status"`
}

func New(secret string) *Service { return &Service{key: sha256.Sum256([]byte(secret))} }
func (s *Service) List(ctx context.Context) ([]Source, error) {
	var rows []entity.DataSources
	if err := dao.DataSources.Ctx(ctx).OrderDesc(dao.DataSources.Columns().CreatedAt).Scan(&rows); err != nil {
		return nil, err
	}
	out := make([]Source, 0, len(rows))
	for _, r := range rows {
		out = append(out, Source{ID: r.Id, Name: r.Name, BaseURL: r.BaseUrl, AppID: r.AppId, AccessKey: r.AccessKey, Status: r.Status, SecretConfigured: r.SecretKeyEncrypted != "", CreatedAt: r.CreatedAt})
	}
	return out, nil
}
func (s *Service) Save(ctx context.Context, in SaveInput) (*Source, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.BaseURL) == "" || strings.TrimSpace(in.AppID) == "" || strings.TrimSpace(in.AccessKey) == "" || strings.TrimSpace(in.SecretKey) == "" {
		return nil, gerror.New("名称、Base URL、APP ID、AK、SK 均不能为空")
	}
	encrypted, err := s.encrypt(in.SecretKey)
	if err != nil {
		return nil, err
	}
	id := guid.S()
	status := in.Status
	if status == "" {
		status = "enabled"
	}
	_, err = dao.DataSources.Ctx(ctx).Data(do.DataSources{Id: id, Name: strings.TrimSpace(in.Name), BaseUrl: strings.TrimRight(strings.TrimSpace(in.BaseURL), "/"), AppId: strings.TrimSpace(in.AppID), AccessKey: strings.TrimSpace(in.AccessKey), SecretKeyEncrypted: encrypted, Status: status, RequestTimeoutSeconds: 30}).Insert()
	if err != nil {
		return nil, gerror.Wrap(err, "保存数据源失败")
	}
	rows, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.ID == id {
			return &item, nil
		}
	}
	return nil, gerror.New("数据源保存后读取失败")
}
func (s *Service) Channels(ctx context.Context, id string) ([]feiniu.Channel, error) {
	client, err := s.client(ctx, id)
	if err != nil {
		return nil, err
	}
	page, err := client.Channels(ctx, 100, "", "")
	return page.Items, err
}
func (s *Service) client(ctx context.Context, id string) (*feiniu.Client, error) {
	var row entity.DataSources
	if err := dao.DataSources.Ctx(ctx).Where(dao.DataSources.Columns().Id, id).Scan(&row); err != nil {
		return nil, err
	}
	if row.Id == "" {
		return nil, gerror.New("数据源不存在")
	}
	secret, err := s.decrypt(row.SecretKeyEncrypted)
	if err != nil {
		return nil, err
	}
	return &feiniu.Client{BaseURL: row.BaseUrl, AppID: row.AppId, AccessKey: row.AccessKey, SecretKey: secret, HTTPClient: &http.Client{Timeout: time.Duration(row.RequestTimeoutSeconds) * time.Second}}, nil
}
func (s *Service) encrypt(v string) (string, error) {
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(g.Seal(nonce, nonce, []byte(v), nil)), nil
}
func (s *Service) decrypt(v string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < g.NonceSize() {
		return "", gerror.New("数据源密钥密文无效")
	}
	plain, err := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], nil)
	return string(plain), err
}
