package util

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

/*
 * NewLogger
 * Logger Service 호출 함수
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.11
 */
func NewLogger(fp, fn string) *Log {

	// 로거 생성
	logger := logrus.New()

	// 매일 자정에 호출될 함수
	dailyRotation := func() {
		// currentDay := time.Now().Format("2006-01-02")
		fileLogger := &lumberjack.Logger{
			Filename: fp + "/" + fn + ".log",
			MaxSize:  10,    // 메가바이트
			Compress: false, // 압축
			MaxAge:   30,    // 일
		}

		// 로그 포맷 설정
		logger.Formatter = &logrus.TextFormatter{
			FullTimestamp: true,
		}

		// 로그 레벨 설정
		logger.Level = logrus.DebugLevel

		// 콘솔과 파일에 로그 출력
		logger.Out = io.MultiWriter(os.Stdout, fileLogger)
	}

	// 최초 로테이션 설정
	dailyRotation()

	// 매일 자정에 로테이션 설정
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				dailyRotation()
			}
		}
	}()

	// 로그 파일 설정 실패 시 표준 출력으로 로그 출력
	defer func() {
		err := recover()
		if err != nil {
			entry := err.(*logrus.Entry)
			logger.WithFields(logrus.Fields{
				"time":        time.Now().Format("2006-01-02 15:04:05"),
				"err_animal":  entry.Data["animal"],
				"err_size":    entry.Data["size"],
				"err_level":   entry.Level,
				"err_message": entry.Message,
				"number":      100,
			}).Error("The ice breaks!") // or use Fatal() to force the process to exit with a nonzero code
		}
	}()

	return &Log{logger}
}

/**
 * Log 구조체
 * 로그 초기화 구조체
 *
 * @auther: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.11
 */
type Log struct {
	logger *logrus.Logger
}

/**
 * Info
 * Info Level 로그 출력 함수
 *
 * @auther: Han Seong San
 * @version: 1.1.0
 * @since: 2024.06.21
 */
func (l *Log) Info(message string) {
	_, file, line, _ := runtime.Caller(1)
	filepath := strings.Split(file, "/")
	l.logger.Info(fmt.Sprintf("INFO %s [%s:%d] %s", time.Now().Format("15:04:05.000000"), filepath[len(filepath)-1], line, message))
}

/**
 * Warn
 * Warn Level 로그 출력 함수
 *
 * @auther: Han Seong San
 * @version: 1.1.0
 * @since: 2024.06.21
 */
func (l *Log) Warn(message string) {
	_, file, line, _ := runtime.Caller(1)
	filepath := strings.Split(file, "/")
	l.logger.Warn(fmt.Sprintf("WARN %s [%s:%d] %s", time.Now().Format("15:04:05.000000"), filepath[len(filepath)-1], line, message))
}

/**
 * Error
 * Error Level 로그 출력 함수
 *
 * @auther: Han Seong San
 * @version: 1.1.0
 * @since: 2024.06.21
 */
func (l *Log) Error(err error) {
	_, file, line, _ := runtime.Caller(1)
	filepath := strings.Split(file, "/")
	l.logger.Error(fmt.Sprintf("ERROR %s [%s:%d] %s", time.Now().Format("15:04:05.000000"), filepath[len(filepath)-1], line, err.Error()))
}

/**
 * HttpInfoWithFields
 * Http Info Level 로그 출력 함수 (Fields 포함)
 *
 * @auther: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
func (l *Log) HttpInfoWithFields(c *gin.Context, message string) {
	l.logger.WithFields(logrus.Fields{
		"ip":     c.ClientIP(),
		"host":   c.Request.Host,
		"path":   c.Request.URL.Path,
		"method": c.Request.Method,
		"status": c.Writer.Status(),
		"size":   c.Writer.Size(),
	}).Info(fmt.Sprintf("[INFO] %s", message))
}

/**
 * HttpWarnWithFields
 * Http Warn Level 로그 출력 함수 (Fields 포함)
 *
 * @auther: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
func (l *Log) HttpWarnWithFields(c *gin.Context, message string) {
	l.logger.WithFields(logrus.Fields{
		"ip":     c.ClientIP(),
		"host":   c.Request.Host,
		"path":   c.Request.URL.Path,
		"method": c.Request.Method,
		"status": c.Writer.Status(),
		"size":   c.Writer.Size(),
	}).Warn(fmt.Sprintf("[WARN] %s", message))
}

/**
 * HttpErrorWithFields
 * Http Error Level 로그 출력 함수 (Fields 포함)
 *
 * @auther: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
func (l *Log) HttpErrorWithFields(c *gin.Context, err error) {
	l.logger.WithFields(logrus.Fields{
		"ip":     c.ClientIP(),
		"host":   c.Request.Host,
		"path":   c.Request.URL.Path,
		"method": c.Request.Method,
		"status": c.Writer.Status(),
		"size":   c.Writer.Size(),
	}).Error(fmt.Sprintf("[ERROR] %s", err.Error()))
}
