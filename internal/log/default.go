package log

import (
	"log/slog"
	"sync"
)

var (
	_log = slog.New(slog.DiscardHandler)
	mu   sync.Mutex
)

func SetLogger(log *slog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	_log = log
}

func Debug(msg string, args ...any) {
	_log.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	_log.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	_log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	_log.Error(msg, args...)
}
