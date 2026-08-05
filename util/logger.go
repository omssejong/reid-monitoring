package util

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

const logTimeFormat = "2006-01-02 15:04:05"

func NewLogger(fp, fn string) *Log {
	_ = os.MkdirAll(fp, 0755)
	logFilePath := filepath.Join(fp, fn+".log")

	// 색을 입힌 출력은 ANSI 이스케이프가 그대로 섞여 vi 등으로 열었을 때 읽기 어렵다.
	// 파일에는 항상 색 없이 쓰고, stdout은 터미널일 때만 색을 입힌다.
	// (운영에서는 systemd가 stdout도 파일로 받으므로 — StandardOutput=file:... —
	//  터미널이 아니면 색을 끄는 것이 곧 그 파일도 깨끗해진다는 뜻이다.)
	writers := []io.Writer{
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: logTimeFormat,
			NoColor:    !isTerminal(os.Stdout),
		},
	}

	if file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		writers = append(writers, zerolog.ConsoleWriter{
			Out:        file,
			TimeFormat: logTimeFormat,
			NoColor:    true,
		})
	}

	logger := zerolog.New(zerolog.MultiLevelWriter(writers...)).With().Timestamp().Logger()
	return &Log{logger: logger}
}

// isTerminal 출력 대상이 실제 터미널인지 판정한다. 파이프/파일로 리다이렉트되면 false.
func isTerminal(f *os.File) bool {
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
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
