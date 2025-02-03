package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
 * CORSMiddleware
 * Gin Framework CORS Middleware
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
func CORSMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")                                                                                                                            // 모든 도메인 허용
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")                                                                                                                    // 쿠키 허용
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With") // 헤더 허용
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")                                                                                             // 메소드 허용

		// OPTIONS 요청에 대한 처리
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent) // 상태 코드 204
			return
		}

		// 다음 미들웨어 호출
		c.Next()

	}

}

/*
 * Logger
 * Gin Framework Logger Middleware
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
func LoggerMiddleware(logger *Log) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 요청 전 로그
		logger.HttpInfoWithFields(c, "Request Start!")

		// 다음 미들웨어 호출
		c.Next()

		// 요청 후 로그
		if c.Err() != nil {
			logger.HttpErrorWithFields(c, c.Err()) // 에러 로그
		} else {
			logger.HttpInfoWithFields(c, "Response End!") // 성공 로그
		}
	}
}

/**
 * SecurityHeaders
 * Gin Framework Security Headers Middleware
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
func SecureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security Headers 설정
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'") // CSP 설정
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")             // MIME 스니핑 방지
		c.Writer.Header().Set("X-Frame-Options", "DENY")                       // Clickjacking 방지

		// 다음 미들웨어 호출
		c.Next()
	}
}
