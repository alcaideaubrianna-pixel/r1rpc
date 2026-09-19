package imaging

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const MaxDownloadBytes int64 = 12 << 20

type Downloader struct {
	client *http.Client
}

func NewDownloader() *Downloader {
	dialer := &net.Dialer{Timeout: 8 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := safeIPs(ctx, host)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
		TLSHandshakeTimeout: 8 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("候选图片重定向次数过多")
			}
			return validateURL(req.Context(), req.URL)
		},
	}
	return &Downloader{client: client}
}

func (d *Downloader) Download(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(NormalizeCandidateURL(rawURL))
	if err != nil {
		return nil, err
	}
	if err := validateURL(ctx, parsed); err != nil {
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
	if parsed == nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
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
