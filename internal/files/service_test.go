package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	storagelocal "r1rpc/internal/storage/local"
)

func TestDetectImage(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "jpeg", data: []byte{0xff, 0xd8, 0xff, 0xdb}, want: "image/jpeg"},
		{name: "png", data: append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 8)...), want: "image/png"},
		{name: "heic", data: []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c'}, want: "image/heic"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "image")
			if err := os.WriteFile(path, test.data, 0o600); err != nil {
				t.Fatal(err)
			}
			got, _, err := detectImage(path)
			if err != nil || got != test.want {
				t.Fatalf("type=%q err=%v", got, err)
			}
		})
	}
}

func TestDetectImageRejectsUnknownData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image")
	if err := os.WriteFile(path, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := detectImage(path); err == nil {
		t.Fatal("expected invalid image error")
	}
}

func TestObjectPathRejectsTraversal(t *testing.T) {
	backend, err := storagelocal.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Open(context.Background(), "../secret"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
