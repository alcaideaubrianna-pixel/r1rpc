package xhsprofile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	clientID, noteID, err := candidateContext(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{
		"userId": userID, "noteId": noteID, "channelTab": "note_detail_r10",
		"timeoutMilliseconds": 15000, "allowCacheFallback": true,
	})
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
	if !usableProfileRecord(record) {
		// 历史版本可能把 code=-100 的失败响应写入缓存，不能阻断后续真实请求。
		_, _ = dao.XhsUserProfiles.Ctx(ctx).Where(dao.XhsUserProfiles.Columns().UserId, userID).Delete()
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

func usableProfileRecord(record entity.XhsUserProfiles) bool {
	if strings.TrimSpace(record.Nickname) == "" && strings.TrimSpace(record.RedId) == "" {
		return false
	}
	if strings.TrimSpace(record.RawJson) == "" {
		return true
	}
	var raw map[string]any
	if json.Unmarshal([]byte(record.RawJson), &raw) != nil {
		return false
	}
	if code, ok := raw["code"].(float64); ok && code != 0 {
		return false
	}
	if success, ok := raw["success"].(bool); ok && !success {
		return false
	}
	if body, ok := raw["body"].(map[string]any); ok {
		if code, ok := body["code"].(float64); ok && code != 0 {
			return false
		}
		if success, ok := body["success"].(bool); ok && !success {
			return false
		}
	}
	return true
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
	return job.AssignedClientId, candidate.ContentId, nil
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
	// content.user_info 返回扁平用户对象；兼容旧的 network.request envelope。
	if _, flat := body["userid"]; flat {
		return body, nil
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
	case string:
		result, _ := strconv.ParseInt(typed, 10, 64)
		return result
	case json.Number:
		result, _ := typed.Int64()
		return result
	}
	return 0
}
func intValue(value any) int { return int(int64Value(value)) }
