package imaging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const MaxDownloadBytes int64 = 12 << 20

type Downloader struct {
	client          *http.Client
	allowedHTTPHost string
}

var (
	sharedDownloaderOnce sync.Once
	sharedDownloader     *Downloader
)

func NewDownloader() *Downloader {
	sharedDownloaderOnce.Do(func() {
		sharedDownloader = newDownloader("")
	})
	return sharedDownloader
}

// NewSourceDownloader 仅允许已由管理员配置的数据源主机使用 HTTP。
// 候选图片仍必须使用 HTTPS，避免扩大通用下载器的 SSRF 攻击面。
func NewSourceDownloader(baseURL string) (*Downloader, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Hostname() == "" {
		return nil, fmt.Errorf("数据源 Base URL 无效")
	}
	allowedHTTPHost := ""
	if parsed.Scheme == "http" {
		allowedHTTPHost = strings.ToLower(parsed.Hostname())
	}
	return newDownloader(allowedHTTPHost), nil
}

func newDownloader(allowedHTTPHost string) *Downloader {
	dialer := &net.Dialer{Timeout: 8 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		// CDN 在当前部署网络中需要通过 HTTP(S)_PROXY 访问；显式禁用代理会导致直连 EOF。
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          128,
		MaxIdleConnsPerHost:   32,
		MaxConnsPerHost:       32,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 12 * time.Second,
		// 目标 URL 已在 validateDownloadURL 中完成 SSRF 校验。这里不能再次对
		// 代理地址做同样校验，否则常见的内网 HTTP(S)_PROXY 会被误判为受限网络。
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 8 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("候选图片重定向次数过多")
			}
			return validateDownloadURL(req.Context(), req.URL, allowedHTTPHost)
		},
	}
	return &Downloader{client: client, allowedHTTPHost: allowedHTTPHost}
}

// IsRetryableDownloadError 区分临时网络故障，交由队列退避重试。
func IsRetryableDownloadError(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "eof") || strings.Contains(message, "tls handshake timeout") ||
		strings.Contains(message, "connection reset") || strings.Contains(message, "timeout")
}

func (d *Downloader) Download(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(NormalizeCandidateURL(rawURL))
	if err != nil {
		return nil, err
	}
	if err := validateDownloadURL(ctx, parsed, d.allowedHTTPHost); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "r1rpc-image-analyzer/1.0")
	request.Header.Set("Accept", "image/jpeg,image/png,image/webp;q=0.9,*/*;q=0.1")
	response, err := d.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("候选图片响应状态异常: %d", response.StatusCode)
	}
	if response.ContentLength > MaxDownloadBytes {
		return nil, fmt.Errorf("候选图片超过大小限制")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxDownloadBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) == 0 || int64(len(data)) > MaxDownloadBytes {
		return nil, fmt.Errorf("候选图片为空或超过大小限制")
	}
	return data, nil
}

func NormalizeCandidateURL(rawURL string) string {
	normalized := strings.TrimSpace(rawURL)
	replacements := []struct{ old, new string }{
		{old: "/format/heif/", new: "/format/jpg/"},
		{old: "/format/avif/", new: "/format/jpg/"},
		{old: "/w/1440/", new: "/w/576/"},
		{old: "%2Fformat%2Fheif%2F", new: "%2Fformat%2Fjpg%2F"},
		{old: "%2Fformat%2Favif%2F", new: "%2Fformat%2Fjpg%2F"},
	}
	for _, replacement := range replacements {
		normalized = strings.ReplaceAll(normalized, replacement.old, replacement.new)
	}
	return normalized
}

func validateURL(ctx context.Context, parsed *url.URL) error {
	return validateDownloadURL(ctx, parsed, "")
}

func validateDownloadURL(ctx context.Context, parsed *url.URL, allowedHTTPHost string) error {
	if parsed == nil || parsed.Hostname() == "" {
		return fmt.Errorf("候选图片只允许 HTTPS 地址")
	}
	host := strings.ToLower(parsed.Hostname())
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && allowedHTTPHost != "" && host == allowedHTTPHost) {
		return fmt.Errorf("候选图片只允许 HTTPS 地址")
	}
	_, err := safeIPs(ctx, parsed.Hostname())
	return err
}

func safeIPs(ctx context.Context, host string) ([]net.IP, error) {
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("候选图片域名没有可用地址")
	}
	ips := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if isUnsafeIP(address.IP) {
			return nil, fmt.Errorf("候选图片地址指向受限网络")
		}
		ips = append(ips, address.IP)
	}
	return ips, nil
}

func isUnsafeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast()
}
