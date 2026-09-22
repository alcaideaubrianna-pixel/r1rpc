package datasource

import (
	"testing"

	"r1rpc/internal/datasource/feiniu"
)

func TestMediaDownloadURLUsesConfiguredHostAndCOSPath(t *testing.T) {
	media := feiniu.Media{
		COSPath:    "/telegram/content/dev/image.jpg",
		ContentURL: "https://public.example/old.jpg",
	}
	got := mediaDownloadURL(media, "http://cos.internal:8080/")
	want := "http://cos.internal:8080/telegram/content/dev/image.jpg"
	if got != want {
		t.Fatalf("mediaDownloadURL() = %q, want %q", got, want)
	}
}

func TestMediaDownloadURLFallsBackWithoutCOSPath(t *testing.T) {
	media := feiniu.Media{ContentURL: "https://public.example/image.jpg"}
	if got := mediaDownloadURL(media, "http://cos.internal"); got != media.ContentURL {
		t.Fatalf("mediaDownloadURL() = %q, want original content URL", got)
	}
}

func TestVideoUsesPreviewOnly(t *testing.T) {
	media := feiniu.Media{AssetType: "video", ContentURL: "https://cdn.example/video.mp4", PreviewURL: "https://cdn.example/video.poster.jpg"}
	if got := media.PreviewDownloadURL(); got != media.PreviewURL {
		t.Fatalf("PreviewDownloadURL() = %q, want %q", got, media.PreviewURL)
	}
	if got := media.DownloadURL(); got != media.ContentURL {
		t.Fatalf("DownloadURL() = %q, want content URL for generic media", got)
	}
}

func TestValidateImageBaseURL(t *testing.T) {
	for _, value := range []string{"", "http://cos.internal:8080", "https://cdn.example.com/path"} {
		if err := validateImageBaseURL(value); err != nil {
			t.Fatalf("validateImageBaseURL(%q) returned %v", value, err)
		}
	}
	if err := validateImageBaseURL("cos.internal"); err == nil {
		t.Fatal("relative image base URL should be rejected")
	}
}
