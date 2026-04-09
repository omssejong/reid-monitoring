package util

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func NewLogger(fp, fn string) *Log {
	_ = os.MkdirAll(fp, 0755)
	logFilePath := filepath.Join(fp, fn+".log")

	writer := io.Writer(os.Stdout)
	if file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		writer = io.MultiWriter(os.Stdout, file)
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return &Log{logger: slog.New(handler)}
}

type Log struct {
	logger *slog.Logger
}

func (l *Log) Info(message string) {
	_, file, line, _ := runtime.Caller(1)
	fileName := filepath.Base(file)
	l.logger.Log(context.Background(), slog.LevelInfo, fmt.Sprintf("[%s:%d] %s", fileName, line, message))
}

func (l *Log) Warn(message string) {
	_, file, line, _ := runtime.Caller(1)
	fileName := filepath.Base(file)
	l.logger.Log(context.Background(), slog.LevelWarn, fmt.Sprintf("[%s:%d] %s", fileName, line, message))
}

func (l *Log) Error(err error) {
	_, file, line, _ := runtime.Caller(1)
	fileName := filepath.Base(file)
	l.logger.Log(context.Background(), slog.LevelError, fmt.Sprintf("[%s:%d] %s", fileName, line, err.Error()))
}

func (l *Log) HttpInfoWithFields(r *http.Request, status, size int, message string) {
	l.logger.Log(context.Background(), slog.LevelInfo, message,
		"ip", clientIP(r),
		"host", r.Host,
		"path", r.URL.Path,
		"method", r.Method,
		"status", status,
		"size", size,
	)
}

func (l *Log) HttpWarnWithFields(r *http.Request, status, size int, message string) {
	l.logger.Log(context.Background(), slog.LevelWarn, message,
		"ip", clientIP(r),
		"host", r.Host,
		"path", r.URL.Path,
		"method", r.Method,
		"status", status,
		"size", size,
	)
}

func (l *Log) HttpErrorWithFields(r *http.Request, status, size int, err error) {
	l.logger.Log(context.Background(), slog.LevelError, err.Error(),
		"ip", clientIP(r),
		"host", r.Host,
		"path", r.URL.Path,
		"method", r.Method,
		"status", status,
		"size", size,
	)
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
