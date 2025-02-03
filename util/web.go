package util

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

/**
 * RequestData
 * Request Data 구조체
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
type RequestST struct {
	UserID  string `json:"userID"`
	UserNum int    `json:"userNum"`
	Target  string `json:"target"`
}

/**
 * ResponseST
 * Response Data 구조체
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
type ResponseST struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

/**
 * WebApp
 * Web Application 설정 함수
 *
 * @return *http.Server
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
func WebApp() *http.Server {
	gin.SetMode(gin.ReleaseMode)

	// Gin Framework 초기화
	r := gin.Default()

	// Logger 설정
	r.Use(gin.Logger())

	// Recovery Middleware 설정
	r.Use(gin.Recovery())

	// Security Middleware 설정
	r.Use(SecureMiddleware())

	// CORS 설정
	r.Use(CORSMiddleware())

	// Router 설정
	appRouter(r)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", configs.SC.Setting.ServerPort),
		Handler: r.Handler(),
	}

	return srv
}

// Router 설정 함수
func appRouter(r *gin.Engine) {

	// API 라우터 설정
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/monitoring/server/info", serverInfo)
		apiV1.POST("/monitoring/server/restart", restartServer)
		apiV1.POST("/monitoring/server/stop", shutdownServer)
		apiV1.POST("/monitoring/service/start", startService)
		apiV1.POST("/monitoring/service/stop", stopService)
		apiV1.POST("/monitoring/service/restart", restartService)
		apiV1.POST("/monitoring/log/download", downloadLog)
	}
}

// Server Information API
func serverInfo(c *gin.Context) {

	infoDict := make(map[string]interface{})
	serverInfoDict := make(map[string]interface{})

	// 서버 CPU 정보 가져오기
	cpuName, err := GetCPUModelName()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cpuCores, err := GetCPUCores()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	serverInfoDict["cpu"] = map[string]interface{}{
		"model":   cpuName,
		"threads": cpuCores,
	}

	// 서버 GPU 정보 가져오기
	gpuInfo, err := GetGPUInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	serverInfoDict["gpu"] = gpuInfo

	// 서버 메모리 정보 가져오기
	memSize, err := GetTotalMemorySize()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	serverInfoDict["memory"] = map[string]interface{}{
		"total": memSize,
	}

	// 서버 디스크 정보 가져오기
	diskInfo, err := GetDiskInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 서버 네트워크 정보 가져오기
	networkBandwidth, err := GetNetworkBandwidth(networkName)

	serverInfoDict["disk"] = diskInfo
	serverInfoDict["network"] = map[string]interface{}{
		"interfaceName": networkName,
		"bandwidth":     networkBandwidth,
	}

	// 서버 정보 세팅
	infoDict["server"] = serverInfoDict

	// 서버 네트워크 정보 가져오기
	networkInfo, err := GetNetworkInfo(networkName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 네트워크 정보 세팅
	infoDict["network"] = networkInfo

	// Version 정보 세팅
	infoDict["version"] = map[string]interface{}{
		"frontend":        configs.SC.Version.Frontend,
		"backend":         configs.SC.Version.Backend,
		"ai":              configs.SC.Version.AI,
		"imageProcessing": configs.SC.Version.ImageProcessing,
		"mediaStreaming":  configs.SC.Version.MediaStreaming,
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Server Information"
	response.Data = infoDict

	log.Info(fmt.Sprintf("Server Information: %v", infoDict))

	c.JSON(http.StatusOK, response)
}

// Server Stop API
func shutdownServer(c *gin.Context) {
	// Request Data 바인딩
	var request RequestST
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 서버 종료
	if err := ShutdownServer(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Server Shutdown Success"
	response.Data = nil

	c.JSON(http.StatusOK, response)
}

// Server Restart API
func restartServer(c *gin.Context) {

	// Request Data 바인딩
	var request RequestST
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 서버 재시작
	if err := RestartServer(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Server Restart Success"
	response.Data = nil

	c.JSON(http.StatusOK, response)
}

// Service Start API
func startService(c *gin.Context) {

	// Request Data 바인딩
	var request RequestST
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 서비스 시작
	var target string
	switch request.Target {
	case "backend":
		target = configs.SC.Setting.BackendServiceName
	case "image-processing":
		target = configs.SC.Setting.ImageProcessingServiceName
	case "media-streaming":
		target = configs.SC.Setting.MediaStreamingServiceName
	case "ai":
		target = configs.SC.Setting.AiServiceName
	}

	if err := StartService(target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Service Start Success"
	response.Data = nil

	c.JSON(http.StatusOK, response)
}

// Service Stop API
func stopService(c *gin.Context) {

	// Request Data 바인딩
	var request RequestST
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 서비스 중지
	var target string
	switch request.Target {
	case "backend":
		target = configs.SC.Setting.BackendServiceName
	case "image-processing":
		target = configs.SC.Setting.ImageProcessingServiceName
	case "media-streaming":
		target = configs.SC.Setting.MediaStreamingServiceName
	case "ai":
		target = configs.SC.Setting.AiServiceName
	}

	if err := StopService(target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Service Stop Success"
	response.Data = nil

	c.JSON(http.StatusOK, response)
}

// Service Restart API
func restartService(c *gin.Context) {

	// Request Data 바인딩
	var request RequestST
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 서비스 재시작
	var target string
	log.Info(fmt.Sprintf("Request Target: %s", request.Target))
	switch request.Target {
	case "backend":
		target = configs.SC.Setting.BackendServiceName
	case "image-processing":
		target = configs.SC.Setting.ImageProcessingServiceName
	case "media":
		target = configs.SC.Setting.MediaStreamingServiceName
	case "ai":
		target = configs.SC.Setting.AiServiceName
	}

	if err := RestartService(target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Service Restart Success"
	response.Data = nil

	c.JSON(http.StatusOK, response)
}

// Monitoring Log Download API
func downloadLog(c *gin.Context) {

	// Request Data 바인딩
	var request RequestST
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Target == "" || request.Target == "null" {
		log.Error(fmt.Errorf("invalid target: %s", request.Target))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Target"})
		return
	}

	dates := strings.Split(request.Target, "~")
	if len(dates) != 2 {
		log.Error(fmt.Errorf("invalid target: %s", request.Target))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Target"})
		return
	}

	files, err := FilterLogFilesByDate(dates[0], dates[1])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	compressPath, err := EncryptCompress(files)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 응답 데이터 설정
	// response := new(ResponseST)
	// response.Code = http.StatusOK
	// response.Message = "Log Download Success"
	// response.Data = map[string]interface{}{
	// 	"path": filepath.Join(monitoringPath, compressPath),
	// }

	// c.JSON(response.Code, response)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(compressPath)))
	log.Info(fmt.Sprintf("Compress Path: %s, Header: %v", compressPath, c.Writer.Header().Values("Content-Type")))
	c.File(compressPath)
}
