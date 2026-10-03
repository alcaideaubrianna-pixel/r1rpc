package observability

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/glog"
)

func TestConfigureLoggingConnectsGoFrameLogger(t *testing.T) {
	logger := glog.Instance()
	original := logger.GetConfig()
	defer func() { _ = logger.SetConfig(original) }()
	manager, err := ConfigureLogging(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	g.Log().Info(context.Background(), g.Map{"event": "test_event", "job_id": "job-1"})
	path := manager.CurrentFile()
	if err = manager.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"event":"test_event"`) || !strings.Contains(content, `"job_id":"job-1"`) {
		t.Fatalf("structured log fields missing: %s", content)
	}
}

func TestDailyLogWriterRotatesByDayAndPrunesDirectory(t *testing.T) {
	dir := t.TempDir()
	current := time.Date(2026, 10, 4, 1, 2, 3, 0, time.Local)
	writer, err := newDailyLogWriter(dir, 50, func() time.Time { return current })
	if err != nil {
		t.Fatal(err)
	}
	writer.maxFileBytes = 32
	if _, err = writer.Write([]byte(strings.Repeat("a", 24))); err != nil {
		t.Fatal(err)
	}
	if _, err = writer.Write([]byte(strings.Repeat("b", 24))); err != nil {
		t.Fatal(err)
	}
	current = current.Add(24 * time.Hour)
	if _, err = writer.Write([]byte(strings.Repeat("c", 24))); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			t.Fatal(infoErr)
		}
		total += info.Size()
		if !strings.Contains(entry.Name(), "start-20261004T010203-day-") {
			t.Fatalf("unexpected log name %q", entry.Name())
		}
	}
	if total > 50 {
		t.Fatalf("log directory size=%d, want <=50", total)
	}
	if len(entries) != 2 {
		t.Fatalf("log files=%d, want 2 after pruning oldest file", len(entries))
	}
	if _, err = os.Stat(filepath.Join(dir, "r1rpc-start-20261004T010203-day-20261005-001.log")); err != nil {
		t.Fatalf("daily log was not created: %v", err)
	}
}
