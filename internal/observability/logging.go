package observability

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/os/glog"
)

const (
	defaultLogDirMaxBytes = int64(200 << 20)
	maxLogFileBytes       = int64(20 << 20)
)

// LogManager 同时向容器标准输出和持久化文件写日志，并限制日志目录总容量。
type LogManager struct {
	writer *dailyLogWriter
}

func ConfigureLogging(dir string, maxDirMB int) (*LogManager, error) {
	maxBytes := int64(maxDirMB) << 20
	if maxBytes <= 0 {
		maxBytes = defaultLogDirMaxBytes
	}
	writer, err := newDailyLogWriter(dir, maxBytes, time.Now)
	if err != nil {
		return nil, err
	}
	output := io.MultiWriter(os.Stdout, writer)
	logger := glog.Instance()
	logger.SetWriter(output)
	logger.SetStdoutPrint(false)
	logger.SetWriterColorEnable(false)
	logger.SetHandlers(glog.HandlerJson)
	log.SetOutput(output)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	return &LogManager{writer: writer}, nil
}

func (m *LogManager) Close() error {
	if m == nil || m.writer == nil {
		return nil
	}
	return m.writer.Close()
}

func (m *LogManager) CurrentFile() string {
	if m == nil || m.writer == nil {
		return ""
	}
	return m.writer.CurrentFile()
}

type dailyLogWriter struct {
	mu           sync.Mutex
	dir          string
	maxDirBytes  int64
	maxFileBytes int64
	startedAt    string
	now          func() time.Time
	day          string
	sequence     int
	file         *os.File
	filePath     string
	fileSize     int64
	bytesToPrune int64
}

func newDailyLogWriter(dir string, maxDirBytes int64, now func() time.Time) (*dailyLogWriter, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("日志目录不能为空")
	}
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("创建日志目录: %w", err)
	}
	started := now()
	w := &dailyLogWriter{
		dir: dir, maxDirBytes: maxDirBytes, maxFileBytes: min(maxLogFileBytes, maxDirBytes),
		startedAt: started.Format("20060102T150405"), now: now,
	}
	if err := w.rotateLocked(started); err != nil {
		return nil, err
	}
	if err := w.pruneLocked(); err != nil {
		_ = w.file.Close()
		return nil, err
	}
	return w, nil
}

func (w *dailyLogWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.now()
	day := now.Format("20060102")
	if w.file == nil || day != w.day || w.fileSize+int64(len(data)) > w.maxFileBytes {
		if day != w.day {
			w.sequence = 0
		}
		if err := w.rotateLocked(now); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(data)
	w.fileSize += int64(n)
	w.bytesToPrune += int64(n)
	pruneInterval := min(int64(1<<20), max(int64(1), w.maxDirBytes/10))
	if err == nil && w.bytesToPrune >= pruneInterval {
		w.bytesToPrune = 0
		err = w.pruneLocked()
	}
	return n, err
}

func (w *dailyLogWriter) rotateLocked(now time.Time) error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
	}
	w.day = now.Format("20060102")
	w.sequence++
	name := fmt.Sprintf("r1rpc-start-%s-day-%s-%03d.log", w.startedAt, w.day, w.sequence)
	path := filepath.Join(w.dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("打开日志文件: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	w.file = file
	w.filePath = path
	w.fileSize = info.Size()
	return nil
}

func (w *dailyLogWriter) pruneLocked() error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}
	type logFile struct {
		path    string
		name    string
		size    int64
		modTime time.Time
	}
	files := make([]logFile, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "r1rpc-start-") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		files = append(files, logFile{path: filepath.Join(w.dir, entry.Name()), name: entry.Name(), size: info.Size(), modTime: info.ModTime()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].modTime.Equal(files[j].modTime) {
			return files[i].name < files[j].name
		}
		return files[i].modTime.Before(files[j].modTime)
	})
	for _, item := range files {
		if total <= w.maxDirBytes {
			break
		}
		if item.path == w.filePath {
			continue
		}
		if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		total -= item.size
	}
	return nil
}

func (w *dailyLogWriter) CurrentFile() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.filePath
}

func (w *dailyLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
