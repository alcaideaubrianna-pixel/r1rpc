package imaging

import (
	"context"
	"image"
	"image/color"
	"net/url"
	"strings"
	"testing"
)

func TestCompareIdenticalImages(t *testing.T) {
	input := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			input.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 7), B: uint8((x + y) * 3), A: 255})
		}
	}
	hashes, err := ComputeHashes(input)
	if err != nil {
		t.Fatal(err)
	}
	comparison := Compare(hashes, hashes)
	if comparison.Score != 1 || comparison.PHashDistance != 0 || comparison.DHashDistance != 0 || comparison.AHashDistance != 0 {
		t.Fatalf("comparison=%+v", comparison)
	}
}

func TestValidateURLRejectsUnsafeTargets(t *testing.T) {
	for _, rawURL := range []string{"http://example.com/a.jpg", "https://127.0.0.1/a.jpg", "https://localhost/a.jpg"} {
		if _, err := NewDownloader().Download(t.Context(), rawURL); err == nil {
			t.Fatalf("Download(%q) should fail", rawURL)
		}
	}
}

func TestSourceDownloaderOnlyAllowsConfiguredHTTPHost(t *testing.T) {
	downloader, err := NewSourceDownloader("http://127.0.0.1/prod-api")
	if err != nil {
		t.Fatal(err)
	}
	allowed, _ := url.Parse("http://127.0.0.1/telegram/resource/1")
	other, _ := url.Parse("http://other.example.test/telegram/resource/1")
	if err = validateDownloadURL(context.Background(), allowed, downloader.allowedHTTPHost); err == nil || !strings.Contains(err.Error(), "受限网络") {
		// 受限网络错误证明 HTTP scheme/host 校验已通过，SSRF 校验仍然生效。
		t.Fatalf("configured host validation returned %v", err)
	}
	if err = validateDownloadURL(context.Background(), other, downloader.allowedHTTPHost); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("other HTTP host should be rejected, got %v", err)
	}
}

func TestNormalizeCandidateURLUsesDecodableFormat(t *testing.T) {
	raw := "https://sns.example/image?imageView2/2/w/576/format/heif/q/58"
	want := "https://sns.example/image?imageView2/2/w/576/format/jpg/q/58"
	if got := NormalizeCandidateURL(raw); got != want {
		t.Fatalf("NormalizeCandidateURL() = %q, want %q", got, want)
	}
}

func TestParseHashesRoundTrip(t *testing.T) {
	want := Hashes{PHash: 0x0123456789abcdef, DHash: 0xfedcba9876543210, AHash: 1}
	got, err := ParseHashes(FormatHash(want.PHash), FormatHash(want.DHash), FormatHash(want.AHash))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ParseHashes() = %+v, want %+v", got, want)
	}
}
