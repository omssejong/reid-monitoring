package util

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
//type ResponseST struct {
//	Code    int                    `json:"code"`
//	Message string                 `json:"message"`
//	Data    map[string]interface{} `json:"data"`
//}

type ResponseST struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"rows"`
	Success bool                   `json:"success"`
}

type ResponseListST struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []string `json:"rows"`
	Success bool     `json:"success"`
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
	//apiV1 := r.Group("/api/v1")
	//{
	//	apiV1.GET("/monitoring/server/info", serverInfo)
	//	apiV1.POST("/monitoring/server/restart", restartServer)
	//	apiV1.POST("/monitoring/server/stop", shutdownServer)
	//	apiV1.POST("/monitoring/service/start", startService)
	//	apiV1.POST("/monitoring/service/stop", stopService)
	//	apiV1.POST("/monitoring/service/restart", restartService)
	//	apiV1.POST("/monitoring/log/download", downloadLog)
	//}

	// 고속검색 전용 라우터. 추후 통합 및 삭제 필요
	reidV1 := r.Group("/monitoring/mgmt")
	{
		reidV1.GET("/server-info", reidServerInfo)
		reidV1.GET("/info", SseMiddleware(), sseInfo)
		reidV1.POST("/reboot", restartServer)
		reidV1.POST("/servicectrl", restartService)
		reidV1.POST("/log/download", downloadLog)
		reidV1.POST("/upload/patch", patchService)
	}
}

// Server Information API
func reidServerInfo(c *gin.Context) {

	//infoDict := make(map[string]interface{})
	serverInfoDict := make(map[string]interface{})

	// 서버 CPU 정보 가져오기
	cpuName, cpuThreads, err := GetCPUModelNameAndPhysicalThreadCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	//cpuCores, err := GetCPUCores()
	//if err != nil {
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	//	return
	//}

	// 서버 GPU 정보 가져오기
	gpuInfo, err := ReidGetGPUInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cpuSockets, err := GetCPUSocket(cpuThreads)
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

	// 서버 디스크 정보 가져오기
	diskInfo, err := GetDiskInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 서버 네트워크 정보 가져오기
	networkBandwidth, err := GetNetworkBandwidth(networkName)

	serverInfoDict["network"] = map[string]interface{}{
		"interfaceName": networkName,
		"bandwidth":     networkBandwidth,
	}
	serverInfoDict["hardwareInfos"] = map[string]interface{}{
		"cpu":           cpuName,
		"cpu_sockets":   cpuSockets,
		"cpu_threads":   cpuThreads,
		"disk":          fmt.Sprintf("%dGB", diskInfo["total"]),
		"gpu":           gpuInfo,
		"gpu_sockets":   len(gpuInfo),
		"mem":           fmt.Sprintf("%0.2fGB", memSize),
		"network_speed": networkBandwidth,
	}
	serverInfoDict["monitorVersion"] = "1.17"
	// 서버 정보 세팅
	//infoDict["server"] = serverInfoDict

	// 서버 네트워크 정보 가져오기
	networkInfo, err := GetNetworkInfo(networkName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 네트워크 정보 세팅
	//infoDict["network"] = networkInfo
	serverInfoDict["networkInfo"] = map[string]interface{}{
		"dns":     "8.8.8.8",
		"gateway": networkInfo["gateway"],
		"iface":   networkName,
		"ip":      networkInfo["ip"],
		"netmask": networkInfo["netmask"],
	}

	// Version 정보 세팅
	serverInfoDict["omeyeVersion"] = map[string]interface{}{
		"BE": configs.SC.Version.Backend,
		"AI": configs.SC.Version.AI,
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Server Information"
	response.Data = serverInfoDict
	response.Success = true

	//log.Info(fmt.Sprintf("Server Information: %v", infoDict))
	log.Info(fmt.Sprintf("Server Information: %v", serverInfoDict))

	c.JSON(http.StatusOK, response)
}

func sseInfo(c *gin.Context) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	systemInfo := make(chan map[string]any)
	go GetSystemInfo(ctx, systemInfo)
	//c.Writer.Flush()
	c.Stream(func(w io.Writer) bool {
		select {
		case info := <-systemInfo:
			if v, ok := info["error"]; ok {
				c.JSON(500, gin.H{"error": v.(error).Error()})
				return false
			}
			c.SSEvent("message", info)
			return true
		}
	})
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
	//var request RequestST
	//if err := c.Bind(&request); err != nil {
	//	log.Error(fmt.Errorf("request %v", err))
	//	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	//	return
	//}

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
	response.Success = true

	c.JSON(http.StatusOK, response)
}

type ServiceRestartStruct struct {
	Command     string   `json:"command"`
	ServiceType []string `json:"serviceType"`
}

// Service Restart API
func restartService(c *gin.Context) {

	// Request Data 바인딩
	var request ServiceRestartStruct
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 서비스 재시작
	var target string
	log.Info(fmt.Sprintf("Request Target: %v", request.ServiceType))
	for _, target = range request.ServiceType {
		switch target {
		case "back":
			target = configs.SC.Setting.BackendServiceName
		case "mediaserver":
			target = configs.SC.Setting.MediaStreamingServiceName
		case "main":
			target = configs.SC.Setting.AiServiceName
		case "middleserver":
			if err := MiddleserverRestart(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if err := RestartService(target); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 응답 데이터 설정
	response := new(ResponseListST)
	response.Code = http.StatusOK
	response.Message = "Service Restart Success"
	response.Data = []string{"success"}
	response.Success = true

	c.JSON(http.StatusOK, response)
}

type LogRequestStruct struct {
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	LogType   []string `json:"logType"`
}

// Monitoring Log Download API
func downloadLog(c *gin.Context) {

	// Request Data 바인딩
	var request LogRequestStruct
	if err := c.Bind(&request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	files, err := FilterLogFilesByDate(request.StartDate, request.EndDate)
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
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(compressPath)))
	log.Info(fmt.Sprintf("Compress Path: %s, Header: %v", compressPath, c.Writer.Header().Values("Content-Type")))
	c.File(compressPath)
}

type PatchResponseST struct {
	Code       int    `json:"code"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	ErrorCode  string `json:"errorCode"`
	ErrorTitle string `json:"errorTitle"`
	ExtraData  any    `json:"extraData"`
}

type PatchRequestST struct {
	FileData []byte `form:"file"`
	Hash     string `form:"hash"`
}

// file upload = formdata
func patchService(c *gin.Context) {
	log.Info("start reid patch")
	encryptedFile, err := c.FormFile("file")
	if err != nil {
		log.Error(err)
		response := PatchResponseST{
			Code:       400,
			Message:    "파일이 존재하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotFoundFile",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}

	tmpFile, err := encryptedFile.Open()
	if err != nil {
		log.Error(err)
		response := PatchResponseST{
			Code:       400,
			Message:    "패치 파일 로드에 실패하였습니다",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "FileLoadError",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}

	receiveHash, isExist := c.GetPostForm("hash")
	if !isExist {
		response := PatchResponseST{
			Code:       400,
			Message:    "해쉬 값이 존재하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotFoundHash",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}
	log.Info(fmt.Sprintf("receive hash: %s", receiveHash))

	patchFileBuffer := bytes.Buffer{}
	patchFileSize, err := patchFileBuffer.ReadFrom(tmpFile)
	if err != nil {
		log.Error(err)
		response := PatchResponseST{
			Code:       400,
			Message:    "해쉬 값이 존재하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotFoundHash",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}

	encryptedZipFile := patchFileBuffer.Bytes()
	copyData := make([]byte, patchFileSize)
	_ = copy(copyData, encryptedZipFile)
	err = checkSha256Sum(copyData, receiveHash)
	if err != nil {
		log.Error(err)
		response := PatchResponseST{
			Code:       400,
			Message:    "해쉬 값이 일치하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotMatchHash",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}
	log.Info(fmt.Sprintf("patch file size: %dbyte", patchFileSize))
	decryptedZipFile, err := decryptZipFile(encryptedZipFile)
	if err != nil {
		log.Error(err)
		response := PatchResponseST{
			Code:       400,
			Message:    "압축파일 추출에 실패하였습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "FailExtractedZipFile",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}

	mkdirErr := os.Mkdir("temp", 0755)
	if mkdirErr != nil {
		log.Error(mkdirErr)
		response := PatchResponseST{
			Code:       500,
			Message:    "서버에러입니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "serverError",
			ExtraData:  nil,
		}
		c.JSON(500, response)
		return
	}

	//defer func() {
	//	os.RemoveAll("temp")
	//}()

	saveZipFileErr := os.WriteFile("temp/patch.zip", decryptedZipFile, 0755)
	if saveZipFileErr != nil {
		log.Error(saveZipFileErr)
		response := PatchResponseST{
			Code:       500,
			Message:    "서버에러입니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "serverError",
			ExtraData:  nil,
		}
		c.JSON(500, response)
		return
	}
	unzipErr := unzipPatchFile()
	if unzipErr != nil {
		response := PatchResponseST{
			Code:       400,
			Message:    "압축 해제에 실패하였습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "FailUnzipProcess",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}

	patchErr := executePatch()
	if patchErr != nil {
		response := PatchResponseST{
			Code:       400,
			Message:    "패치를 실패하였습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "FailPatchProcess",
			ExtraData:  nil,
		}
		c.JSON(400, response)
		return
	}

	response := PatchResponseST{
		Code:       200,
		Message:    "성공적으로 패치하였습니다.",
		Success:    false,
		ErrorCode:  "",
		ErrorTitle: "PatchSuccess",
		ExtraData:  nil,
	}
	c.JSON(200, response)
}
