package feiniu

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type Client struct {
	BaseURL, AppID, AccessKey, SecretKey string
	HTTPClient                           *http.Client
}
type Timestamp struct{ time.Time }

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	value := strings.Trim(string(data), `"`)
	if value == "" || value == "null" {
		t.Time = time.Time{}
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("不支持的 FeiNiu 时间格式 %q", value)
}

type Channel struct {
	ID       int64  `json:"channel_id"`
	Title    string `json:"title"`
	Username string `json:"username"`
	ChatType string `json:"chat_type"`
}
type Media struct {
	AssetID    int64  `json:"asset_id"`
	AssetType  string `json:"asset_type"`
	PreviewURI string `json:"preview_uri"`
	BinaryMD5  string `json:"binary_md5"`
	PHash      string `json:"phash"`
	Sort       int    `json:"sort"`
}
type Note struct {
	ID              int64          `json:"note_id"`
	NoteCode        string         `json:"note_code"`
	SourceMessageID string         `json:"source_message_id"`
	Title           string         `json:"title"`
	PlainText       *string        `json:"plain_text"`
	Attributes      map[string]any `json:"attributes"`
	Media           []Media        `json:"media"`
	CreatedAt       Timestamp      `json:"created_at"`
	UpdatedAt       Timestamp      `json:"updated_at"`
}
type Page[T any] struct {
	Items      []T       `json:"items"`
	NextCursor string    `json:"next_cursor"`
	NextNo     string    `json:"next_no"`
	HasMore    bool      `json:"has_more"`
	Watermark  Timestamp `json:"watermark"`
}

func (c *Client) request(ctx context.Context, path string, query url.Values, out any) error {
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	base := strings.TrimRight(c.BaseURL, "/")
	u, err := url.Parse(base + path)
	if err != nil {
		return err
	}
	u.RawQuery = query.Encode()
	bodyHash := sha256.Sum256(nil)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	canonical := strings.Join([]string{"GET", u.Path, u.RawQuery, ts, nonce, hex.EncodeToString(bodyHash[:])}, "\n")
	mac := hmac.New(sha256.New, []byte(c.SecretKey))
	_, _ = mac.Write([]byte(canonical))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-App-Id", c.AppID)
	req.Header.Set("X-App-Key", c.AccessKey)
	req.Header.Set("X-App-Timestamp", ts)
	req.Header.Set("X-App-Nonce", nonce)
	req.Header.Set("X-App-Signature", hex.EncodeToString(mac.Sum(nil)))
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		preview := responsePreview(data)
		g.Log().Errorf(ctx, "FeiNiu OpenAPI 请求失败 method=%s url=%s status=%d contentType=%q body=%q", http.MethodGet, u.String(), resp.StatusCode, resp.Header.Get("Content-Type"), preview)
		return fmt.Errorf("FeiNiu OpenAPI HTTP %d (%s): %s", resp.StatusCode, resp.Header.Get("Content-Type"), preview)
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		preview := responsePreview(data)
		g.Log().Errorf(ctx, "FeiNiu OpenAPI 返回非 JSON method=%s url=%s status=%d contentType=%q body=%q err=%v", http.MethodGet, u.String(), resp.StatusCode, resp.Header.Get("Content-Type"), preview, err)
		if strings.HasPrefix(strings.TrimSpace(string(data)), "<") {
			return fmt.Errorf("FeiNiu OpenAPI 返回了 HTML 而非 JSON，请检查 Base URL 和网关路径（当前 URL: %s）", u.String())
		}
		return fmt.Errorf("FeiNiu OpenAPI 响应解析失败: %w", err)
	}
	if envelope.Code != 0 {
		return fmt.Errorf("FeiNiu OpenAPI code=%d", envelope.Code)
	}
	return json.Unmarshal(envelope.Data, out)
}

func responsePreview(data []byte) string {
	const max = 512
	text := strings.TrimSpace(string(data))
	if len(text) > max {
		return text[:max] + "…"
	}
	return text
}

func (c *Client) Channels(ctx context.Context, limit int, cursor, updatedAfter string) (Page[Channel], error) {
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if updatedAfter != "" {
		q.Set("updated_after", updatedAfter)
	}
	var out Page[Channel]
	err := c.request(ctx, "/open-api/v1/channels", q, &out)
	return out, err
}
func (c *Client) Notes(ctx context.Context, channelID int64, limit int, nextNo, updatedAfter string) (Page[Note], error) {
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	if nextNo != "" {
		q.Set("next_no", nextNo)
	}
	if updatedAfter != "" {
		q.Set("updated_after", updatedAfter)
	}
	var out Page[Note]
	err := c.request(ctx, fmt.Sprintf("/open-api/v1/channels/%d/notes", channelID), q, &out)
	return out, err
}
