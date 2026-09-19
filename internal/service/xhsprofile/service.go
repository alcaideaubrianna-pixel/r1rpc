package xhsprofile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"r1rpc/internal/dao"
	"r1rpc/internal/files"
	"r1rpc/internal/imaging"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
)

type Service struct{ files *files.Service }

type Profile struct {
	UserID         string    `json:"userId"`
	RedID          string    `json:"redId"`
	Nickname       string    `json:"nickname"`
	AvatarURL      string    `json:"avatarUrl"`
	Description    string    `json:"description"`
	Gender         int       `json:"gender"`
	IPLocation     string    `json:"ipLocation"`
	FansCount      int64     `json:"fansCount"`
	LikedCount     int64     `json:"likedCount"`
	CollectedCount int64     `json:"collectedCount"`
	NoteCount      int64     `json:"noteCount"`
	ShareLink      string    `json:"shareLink"`
	FetchedAt      time.Time `json:"fetchedAt"`
	Cached         bool      `json:"cached"`
}

type InvokeFunc func(ctx context.Context, clientID string, payload json.RawMessage) (json.RawMessage, error)

func New(fileService *files.Service) *Service { return &Service{files: fileService} }

func (s *Service) Get(ctx context.Context, userID, candidateID string, invoke InvokeFunc) (*Profile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, gerror.New("用户 ID 不能为空")
	}
	if cached, err := s.load(ctx, userID); err != nil {
		return nil, err
	} else if cached != nil && time.Since(cached.FetchedAt) < 24*time.Hour {
		cached.Cached = true
		return cached, nil
	}
	clientID, extParams, err := candidateContext(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"path": "/api/sns/v3/user/info", "method": "GET", "query": map[string]string{
		"user_id": userID, "cny_source": "note_detail_r10", "ext_params": extParams,
		"new_page_exp": "1", "profile_page_head_exp": "1"}, "timeoutMilliseconds": 15000})
	raw, err := invoke(ctx, clientID, payload)
	if err != nil {
		return nil, err
	}
	data, err := parseResponse(raw)
	if err != nil {
		return nil, err
	}
	avatarRemote := firstString(data, "imageb", "images")
	avatarFileID := ""
	if avatarRemote != "" {
		if bytes, downloadErr := imaging.NewDownloader().Download(ctx, avatarRemote); downloadErr == nil {
			if stored, saveErr := s.files.SaveBytes(ctx, bytes, "xhs-avatar.jpg", 0); saveErr == nil {
				avatarFileID = stored.ID
			}
		}
	}
	record := do.XhsUserProfiles{UserId: firstString(data, "userid", "user_id"), RedId: firstString(data, "red_id"),
		Nickname: firstString(data, "nickname"), AvatarUrl: avatarRemote, AvatarFileId: avatarFileID,
		Description: firstString(data, "desc", "description"), Gender: intValue(data["gender"]), IpLocation: firstString(data, "ip_location"),
		FansCount: int64Value(data["fans"]), LikedCount: int64Value(data["liked"]), CollectedCount: int64Value(data["collected"]),
		NoteCount: int64Value(data["ndiscovery"]), ShareLink: firstString(data, "share_link"), RawJson: string(raw), FetchedAt: time.Now()}
	if record.UserId == "" {
		record.UserId = userID
	}
	if _, err = dao.XhsUserProfiles.Ctx(ctx).Data(record).OnConflict(dao.XhsUserProfiles.Columns().UserId).Save(); err != nil {
		return nil, gerror.Wrap(err, "保存小红书用户资料失败")
	}
	return s.load(ctx, userID)
}

func (s *Service) load(ctx context.Context, userID string) (*Profile, error) {
	var record entity.XhsUserProfiles
	if err := dao.XhsUserProfiles.Ctx(ctx).Where(dao.XhsUserProfiles.Columns().UserId, userID).Scan(&record); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if record.UserId == "" {
		return nil, nil
	}
	avatarURL := record.AvatarUrl
	if record.AvatarFileId != "" {
		if value, err := s.files.URL(ctx, record.AvatarFileId); err == nil {
			avatarURL = value
		}
	}
	return &Profile{UserID: record.UserId, RedID: record.RedId, Nickname: record.Nickname, AvatarURL: avatarURL,
		Description: record.Description, Gender: record.Gender, IPLocation: record.IpLocation, FansCount: record.FansCount,
		LikedCount: record.LikedCount, CollectedCount: record.CollectedCount, NoteCount: record.NoteCount,
		ShareLink: record.ShareLink, FetchedAt: record.FetchedAt}, nil
}

func candidateContext(ctx context.Context, candidateID string) (string, string, error) {
	var candidate entity.ImageCandidates
	if err := dao.ImageCandidates.Ctx(ctx).Where(dao.ImageCandidates.Columns().Id, candidateID).Scan(&candidate); err != nil {
		return "", "", err
	}
	if candidate.Id == "" {
		return "", "", gerror.New("候选笔记不存在")
	}
	var job entity.ImageJobs
	if err := dao.ImageJobs.Ctx(ctx).Where(dao.ImageJobs.Columns().Id, candidate.JobId).Scan(&job); err != nil {
		return "", "", err
	}
	var image entity.ImageCandidateImages
	imageColumns := dao.ImageCandidateImages.Columns()
	_ = dao.ImageCandidateImages.Ctx(ctx).Where(imageColumns.CandidateId, candidate.Id).
		Where(imageColumns.DownloadStatus, "analyzed").OrderDesc(imageColumns.Score).OrderAsc(imageColumns.ImageIndex).Limit(1).Scan(&image)
	fileID := noteFileID(image.SourceUrl)
	ext, _ := json.Marshal(map[string]any{"mention_sku_note_file_ids": []string{fileID}, "mention_sku_note_id": candidate.ContentId, "mention_sku_note_type": "normal"})
	return job.AssignedClientId, string(ext), nil
}

func noteFileID(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	path := strings.TrimPrefix(parsed.Path, "/")
	if index := strings.Index(path, "notes_pre_post/"); index >= 0 {
		return path[index:]
	}
	return path
}

func parseResponse(raw json.RawMessage) (map[string]any, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, gerror.Wrap(err, "解析用户资料响应失败")
	}
	body := root
	if value, ok := root["body"].(map[string]any); ok {
		body = value
	}
	if success, exists := body["success"].(bool); exists && !success {
		return nil, fmt.Errorf("用户资料接口失败: %s", firstString(body, "msg", "message"))
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("用户资料响应缺少 data")
	}
	if firstString(data, "userid", "user_id") == "" {
		return nil, fmt.Errorf("用户资料响应为空")
	}
	return data, nil
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key].(string); ok {
			return value
		}
	}
	return ""
}
func int64Value(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		result, _ := typed.Int64()
		return result
	}
	return 0
}
func intValue(value any) int { return int(int64Value(value)) }
