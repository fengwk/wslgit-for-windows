package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DebugLogger struct {
	enabled bool
	file    *os.File
	mu      sync.Mutex
}

func NewDebugLogger(config Config) *DebugLogger {
	logger := &DebugLogger{enabled: config.Debug}
	if !config.Debug {
		return logger
	}

	file, err := openLogFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wslgit: 打开调试日志失败: %v\n", err)
		return logger
	}

	logger.file = file
	logger.Printf("debug logging enabled")
	return logger
}

func (l *DebugLogger) Printf(format string, args ...any) {
	if l == nil || !l.enabled || l.file == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(l.file, "%s %s\n", time.Now().Format(time.RFC3339), message)
}

func (l *DebugLogger) Close() {
	if l == nil || l.file == nil {
		return
	}
	_ = l.file.Close()
}

func openLogFile() (*os.File, error) {
	baseDir := os.Getenv("LOCALAPPDATA")
	if baseDir == "" {
		var err error
		baseDir, err = os.UserCacheDir()
		if err != nil {
			baseDir = os.TempDir()
		}
	}

	logDir := filepath.Join(baseDir, "wslgit", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}

	logFile := filepath.Join(logDir, time.Now().Format("20060102")+".log")
	return os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}
