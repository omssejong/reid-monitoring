package util

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
)

func NewLogger(fp, fn string) *Log {
	_ = os.MkdirAll(fp, 0755)
	logFilePath := filepath.Join(fp, fn+".log")

	writer := io.Writer(os.Stdout)
	if file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		writer = io.MultiWriter(os.Stdout, file)
	}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        writer,
		TimeFormat: "2006-01-02 15:04:05",
	}

	logger := zerolog.New(consoleWriter).With().Timestamp().Logger()
	return &Log{logger: logger}
}

type Log struct {
	logger zerolog.Logger
}

func (l *Log) Info(message string) {
	_, file, line, _ := runtime.Caller(1)
	fileName := filepath.Base(file)
	l.logger.Info().
		Str("caller", fmt.Sprintf("%s:%d", fileName, line)).
		Msg(message)
}

func (l *Log) Warn(message string) {
	_, file, line, _ := runtime.Caller(1)
	fileName := filepath.Base(file)
	l.logger.Warn().
		Str("caller", fmt.Sprintf("%s:%d", fileName, line)).
		Msg(message)
}

func (l *Log) Error(err error) {
	_, file, line, _ := runtime.Caller(1)
	fileName := filepath.Base(file)
	l.logger.Error().
		Str("caller", fmt.Sprintf("%s:%d", fileName, line)).
		Err(err).
		Msg(err.Error())
}

func (l *Log) HttpInfoWithFields(r *http.Request, status, size int, message string) {
	l.logger.Info().
		Str("ip", clientIP(r)).
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Int("status", status).
		Int("size", size).
		Msg(message)
}

func (l *Log) HttpWarnWithFields(r *http.Request, status, size int, message string) {
	l.logger.Warn().
		Str("ip", clientIP(r)).
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Int("status", status).
		Int("size", size).
		Msg(message)
}

func (l *Log) HttpErrorWithFields(r *http.Request, status, size int, err error) {
	l.logger.Error().
		Err(err).
		Str("ip", clientIP(r)).
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Int("status", status).
		Int("size", size).
		Msg(err.Error())
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
