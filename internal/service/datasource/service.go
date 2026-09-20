package datasource

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

type Enqueuer interface {
	EnqueueSourceScan(context.Context, string, int) error
	EnqueueSourcePrepare(context.Context, string) error
}
type Service struct {
	key      [32]byte
	enqueuer Enqueuer
}
type SourceChannelPage struct {
	Items      []entity.SourceChannels `json:"items"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"pageSize"`
	Total      int64                   `json:"total"`
	TotalPages int                     `json:"totalPages"`
}
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
type ScanTaskPage struct {
	Items      []entity.ChannelScanTasks `json:"items"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"pageSize"`
	Total      int64                     `json:"total"`
	TotalPages int                       `json:"totalPages"`
}
type ScanTaskInput struct {
	DataSourceID        string `json:"dataSourceId"`
	ChannelID           int64  `json:"channelId"`
	ChannelTitle        string `json:"channelTitle"`
	Mode                string `json:"mode"`
	InitialLimit        int    `json:"initialLimit"`
	PollIntervalMinutes int    `json:"pollIntervalMinutes"`
	Priority            int    `json:"priority"`
	SearchConfigID      string `json:"searchConfigId"`
}

func (s *Service) SetEnqueuer(value Enqueuer) { s.enqueuer = value }
func (s *Service) CreateScanTask(ctx context.Context, in ScanTaskInput) (*entity.ChannelScanTasks, error) {
	if in.DataSourceID == "" || in.ChannelID <= 0 {
		return nil, gerror.New("数据源和频道不能为空")
	}
	if in.Mode != "continuous" {
		in.Mode = "once"
	}
	if in.InitialLimit < 1 || in.InitialLimit > 100 {
		in.InitialLimit = 10
	}
	if in.PollIntervalMinutes < 5 || in.PollIntervalMinutes > 30 {
		in.PollIntervalMinutes = 10
	}
	id := guid.S()
	_, err := dao.ChannelScanTasks.Ctx(ctx).Data(do.ChannelScanTasks{Id: id, DataSourceId: in.DataSourceID, ChannelId: in.ChannelID, ChannelTitle: in.ChannelTitle, Mode: in.Mode, InitialLimit: in.InitialLimit, PollIntervalMinutes: in.PollIntervalMinutes, Priority: in.Priority, SearchConfigId: in.SearchConfigID, Status: "queued", NextRunAt: time.Now()}).Insert()
	if err != nil {
		return nil, err
	}
	if s.enqueuer != nil {
		if err = s.enqueuer.EnqueueSourceScan(ctx, id, in.Priority); err != nil {
			return nil, err
		}
	}
	var row entity.ChannelScanTasks
	err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, id).Scan(&row)
	return &row, err
}
func (s *Service) ListScanTasks(ctx context.Context, page, pageSize int) (*ScanTaskPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	model := dao.ChannelScanTasks.Ctx(ctx)
	total, err := model.Count()
	if err != nil {
		return nil, err
	}
	var rows []entity.ChannelScanTasks
	if err = model.OrderDesc(dao.ChannelScanTasks.Columns().CreatedAt).Page(page, pageSize).Scan(&rows); err != nil {
		return nil, err
	}
	return &ScanTaskPage{Items: rows, Page: page, PageSize: pageSize, Total: int64(total), TotalPages: (total + pageSize - 1) / pageSize}, nil
}
func (s *Service) RunScan(ctx context.Context, taskID string) error {
	var task entity.ChannelScanTasks
	if err := dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, taskID).Scan(&task); err != nil {
		return err
	}
	if task.Id == "" || task.Status == "paused" || task.Status == "cancelled" {
		return nil
	}
	client, err := s.client(ctx, task.DataSourceId)
	if err != nil {
		return s.failScan(ctx, task, err)
	}
	_, _ = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, task.Id).Data(do.ChannelScanTasks{Status: "running", LastError: ""}).Update()
	if task.CursorValue == "" {
		if _, err := dao.SourceScanTaskNotes.Ctx(ctx).Where(dao.SourceScanTaskNotes.Columns().ScanTaskId, task.Id).Delete(); err != nil {
			return s.failScan(ctx, task, err)
		}
	}
	updatedAfter := ""
	if !task.Watermark.IsZero() {
		updatedAfter = task.Watermark.Format(time.RFC3339)
	}
	page, err := client.Notes(ctx, task.ChannelId, task.InitialLimit, task.CursorValue, updatedAfter)
	if err != nil {
		return s.failScan(ctx, task, err)
	}
	for _, note := range page.Items {
		raw, _ := json.Marshal(note)
		attrs, _ := json.Marshal(note.Attributes)
		noteID := guid.S()
		_, err = dao.SourceNotes.Ctx(ctx).Data(do.SourceNotes{Id: noteID, DataSourceId: task.DataSourceId, ScanTaskId: task.Id, ChannelId: task.ChannelId, ExternalNoteId: fmt.Sprint(note.ID), NoteCode: note.NoteCode, Title: note.Title, PlainText: note.PlainText, AttributesJson: string(attrs), RawJson: string(raw), Status: "received"}).InsertIgnore()
		if err != nil {
			return err
		}
		var saved entity.SourceNotes
		if err = dao.SourceNotes.Ctx(ctx).Where(dao.SourceNotes.Columns().DataSourceId, task.DataSourceId).Where(dao.SourceNotes.Columns().ChannelId, task.ChannelId).Where(dao.SourceNotes.Columns().ExternalNoteId, fmt.Sprint(note.ID)).Scan(&saved); err != nil {
			return err
		}
		if _, err = dao.SourceScanTaskNotes.Ctx(ctx).Data(do.SourceScanTaskNotes{Id: guid.S(), ScanTaskId: task.Id, NoteId: saved.Id}).InsertIgnore(); err != nil {
			return err
		}
		for _, media := range note.Media {
			if media.AssetType != "image" || media.PreviewURI == "" {
				continue
			}
			mediaRaw, _ := json.Marshal(media)
			_, err = dao.SourceNoteImages.Ctx(ctx).Data(do.SourceNoteImages{Id: guid.S(), NoteId: saved.Id, ExternalAssetId: fmt.Sprint(media.AssetID), AssetType: media.AssetType, SourceUrl: media.PreviewURI, Phash: media.PHash, DownloadStatus: "pending", PreprocessStatus: "pending", FilterDecision: "pending", RawJson: string(mediaRaw), ImageIndex: media.Sort}).InsertIgnore()
			if err != nil {
				return err
			}
		}
	}
	update := do.ChannelScanTasks{
		Status:        "preparing",
		CursorValue:   "",
		Watermark:     page.Watermark.Time,
		NextRunAt:     nil,
		LastSuccessAt: time.Now(),
		LastError:     "",
	}
	shouldContinue := page.HasMore && page.NextNo != ""
	if shouldContinue {
		update.Status = "queued"
		update.CursorValue = page.NextNo
		update.Watermark = task.Watermark
		update.NextRunAt = time.Now()
	} else if task.Mode == "continuous" {
		update.NextRunAt = time.Now().Add(time.Duration(task.PollIntervalMinutes) * time.Minute)
	}
	if _, err = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, task.Id).Data(update).Update(); err != nil {
		return err
	}
	if shouldContinue && s.enqueuer != nil {
		return s.enqueuer.EnqueueSourceScan(ctx, task.Id, task.Priority)
	}
	if !shouldContinue && s.enqueuer != nil {
		return s.enqueuer.EnqueueSourcePrepare(ctx, task.Id)
	}
	return nil
}
func (s *Service) failScan(ctx context.Context, task entity.ChannelScanTasks, cause error) error {
	_, _ = dao.ChannelScanTasks.Ctx(ctx).Where(dao.ChannelScanTasks.Columns().Id, task.Id).Data(do.ChannelScanTasks{Status: "failed", LastError: cause.Error(), NextRunAt: time.Now().Add(5 * time.Minute)}).Update()
	return cause
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
func (s *Service) Update(ctx context.Context, id string, in SaveInput) (*Source, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.BaseURL) == "" || strings.TrimSpace(in.AppID) == "" || strings.TrimSpace(in.AccessKey) == "" {
		return nil, gerror.New("名称、Base URL、APP ID、AK 均不能为空")
	}
	var current entity.DataSources
	if err := dao.DataSources.Ctx(ctx).Where(dao.DataSources.Columns().Id, id).Scan(&current); err != nil {
		return nil, err
	}
	if current.Id == "" {
		return nil, gerror.New("数据源不存在")
	}
	status := in.Status
	if status == "" {
		status = current.Status
	}
	update := do.DataSources{Name: strings.TrimSpace(in.Name), BaseUrl: strings.TrimRight(strings.TrimSpace(in.BaseURL), "/"), AppId: strings.TrimSpace(in.AppID), AccessKey: strings.TrimSpace(in.AccessKey), Status: status}
	if strings.TrimSpace(in.SecretKey) != "" {
		encrypted, err := s.encrypt(in.SecretKey)
		if err != nil {
			return nil, err
		}
		update.SecretKeyEncrypted = encrypted
	}
	if _, err := dao.DataSources.Ctx(ctx).Where(dao.DataSources.Columns().Id, id).Data(update).Update(); err != nil {
		return nil, gerror.Wrap(err, "更新数据源失败")
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
	return nil, gerror.New("数据源更新后读取失败")
}
func (s *Service) SyncChannels(ctx context.Context, id string) (int, error) {
	client, err := s.client(ctx, id)
	if err != nil {
		return 0, err
	}
	var (
		cursor string
		count  int
	)
	for pageNumber := 0; pageNumber < 1000; pageNumber++ {
		page, requestErr := client.Channels(ctx, 100, cursor, "")
		if requestErr != nil {
			return count, requestErr
		}
		for _, channel := range page.Items {
			if err = s.saveChannel(ctx, id, channel); err != nil {
				return count, err
			}
			count++
		}
		if !page.HasMore || page.NextCursor == "" {
			return count, nil
		}
		cursor = page.NextCursor
	}
	return count, gerror.New("频道同步分页超过安全上限")
}

func (s *Service) saveChannel(ctx context.Context, sourceID string, channel feiniu.Channel) error {
	columns := dao.SourceChannels.Columns()
	var current entity.SourceChannels
	if err := dao.SourceChannels.Ctx(ctx).
		Where(columns.DataSourceId, sourceID).
		Where(columns.ChannelId, channel.ID).
		Scan(&current); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	raw, _ := json.Marshal(channel)
	data := do.SourceChannels{Title: channel.Title, Username: channel.Username, ChatType: channel.ChatType, RawJson: string(raw), LastSyncedAt: time.Now()}
	if current.Id != "" {
		_, err := dao.SourceChannels.Ctx(ctx).Where(columns.Id, current.Id).Data(data).Update()
		return err
	}
	data.Id = guid.S()
	data.DataSourceId = sourceID
	data.ChannelId = channel.ID
	_, err := dao.SourceChannels.Ctx(ctx).Data(data).Insert()
	return err
}

func (s *Service) ListChannels(ctx context.Context, sourceID, keyword string, page, pageSize int) (*SourceChannelPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	columns := dao.SourceChannels.Columns()
	model := dao.SourceChannels.Ctx(ctx)
	if sourceID != "" {
		model = model.Where(columns.DataSourceId, sourceID)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		model = model.WhereLike(columns.Title, "%"+keyword+"%")
	}
	total, err := model.Count()
	if err != nil {
		return nil, err
	}
	var rows []entity.SourceChannels
	if err = model.OrderDesc(columns.LastSyncedAt).Page(page, pageSize).Scan(&rows); err != nil {
		return nil, err
	}
	return &SourceChannelPage{Items: rows, Page: page, PageSize: pageSize, Total: int64(total), TotalPages: (total + pageSize - 1) / pageSize}, nil
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
