package util

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"
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

func (l *Log) HttpInfoWithFields(c *gin.Context, message string) {
	l.logger.Log(context.Background(), slog.LevelInfo, message,
		"ip", c.ClientIP(),
		"host", c.Request.Host,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"status", c.Writer.Status(),
		"size", c.Writer.Size(),
	)
}

func (l *Log) HttpWarnWithFields(c *gin.Context, message string) {
	l.logger.Log(context.Background(), slog.LevelWarn, message,
		"ip", c.ClientIP(),
		"host", c.Request.Host,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"status", c.Writer.Status(),
		"size", c.Writer.Size(),
	)
}

func (l *Log) HttpErrorWithFields(c *gin.Context, err error) {
	l.logger.Log(context.Background(), slog.LevelError, err.Error(),
		"ip", c.ClientIP(),
		"host", c.Request.Host,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"status", c.Writer.Status(),
		"size", c.Writer.Size(),
	)
}
