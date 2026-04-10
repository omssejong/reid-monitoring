package util

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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

type DeleteFilesRequestST struct {
	Paths []string `json:"paths"`
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
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(LoggerMiddleware(log))
	r.Use(SecureMiddleware())
	r.Use(CORSMiddleware())

	appRouter(r)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", configs.SC.Setting.ServerPort),
		Handler: r,
	}

	return srv
}

// Router 설정 함수
func appRouter(r chi.Router) {

	// API 라우터 설정
	r.Route("/monitoring/mgmt", func(reidV1 chi.Router) {
		reidV1.Get("/server-info", reidServerInfo)
		reidV1.With(SseMiddleware()).Get("/info", sseInfo)
		reidV1.Get("/get-info", httpInfo)
		reidV1.Post("/reboot", restartServer)
		reidV1.Post("/servicectrl", serviceControl)
		reidV1.Post("/shutdown", shutdownServer)
		reidV1.Post("/log/download", downloadLog)
		reidV1.Post("/delete/files", deleteFiles)
		reidV1.Post("/upload/patch", patchService)
	})
	log.Info("version 1.0.0.260407")
}

// Server Information API
func reidServerInfo(w http.ResponseWriter, r *http.Request) {

	//infoDict := make(map[string]interface{})
	serverInfoDict := make(map[string]interface{})

	// 서버 CPU 정보 가져오기
	cpuName, cpuThreads, err := GetCPUModelNameAndPhysicalThreadCount()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	//cpuCores, err := GetCPUCores()
	//if err != nil {
	//	return
	//}

	// 서버 GPU 정보 가져오기
	gpuInfo, err := ReidGetGPUInfo()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	cpuSockets, err := GetCPUSocket(cpuThreads)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	//serverInfoDict["gpu"] = gpuInfo

	// 서버 메모리 정보 가져오기
	memSize, err := GetTotalMemorySize()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	// 서버 디스크 정보 가져오기
	diskInfo, err := GetDiskInfo()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	serverInfoDict["hardwareInfos"] = map[string]interface{}{
		"cpu":         cpuName,
		"cpu_sockets": cpuSockets,
		"cpu_threads": cpuThreads,
		"disk":        fmt.Sprintf("%dGB", diskInfo["total"]),
		"gpu":         gpuInfo,
		"gpu_sockets": len(gpuInfo),
		"mem":         fmt.Sprintf("%0.2fGB", memSize),
	}
	serverInfoDict["monitorVersion"] = "1.2"
	// 서버 정보 세팅
	//infoDict["server"] = serverInfoDict

	// 서버 네트워크 정보 가져오기
	networkInfos, err := GetNetworkInterfaces()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	// 네트워크 정보 세팅
	//infoDict["network"] = networkInfos
	serverInfoDict["networkInfos"] = networkInfos

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

	writeJSON(w, http.StatusOK, response)
}

func sseInfo(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErrorJSON(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}

	systemInfo := make(chan map[string]any)
	go GetSystemInfo(r.Context(), systemInfo)

	for {
		select {
		case <-r.Context().Done():
			return
		case info := <-systemInfo:
			if v, ok := info["error"]; ok {
				writeErrorJSON(w, http.StatusInternalServerError, v.(error))
				return
			}
			payload, err := json.Marshal(info)
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, err)
				return
			}
			if _, err := io.WriteString(w, "event: message\n"); err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func httpInfo(w http.ResponseWriter, r *http.Request) {
	systemInfo := make(chan map[string]any)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go GetSystemInfo(ctx, systemInfo)

	select {
	case <-r.Context().Done():
		return
	case info := <-systemInfo:
		if v, ok := info["error"]; ok {
			writeErrorJSON(w, http.StatusInternalServerError, v.(error))
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

// Server Stop API
func shutdownServer(w http.ResponseWriter, r *http.Request) {
	// Request Data 바인딩
	var request RequestST
	if err := decodeJSONBody(r, &request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		writeErrorJSON(w, http.StatusBadRequest, err)
		return
	}

	// 서버 종료
	if err := ShutdownServer(); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Server Shutdown Success"
	response.Data = nil

	writeJSON(w, http.StatusOK, response)
}

// Server Restart API
func restartServer(w http.ResponseWriter, r *http.Request) {

	// 서버 재시작
	if err := RestartServer(); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	// 응답 데이터 설정
	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Server Restart Success"
	response.Data = nil
	response.Success = true

	writeJSON(w, http.StatusOK, response)
}

type ServiceRestartStruct struct {
	Command     string   `json:"command"`
	ServiceType []string `json:"serviceType"`
}

func serviceControl(w http.ResponseWriter, r *http.Request) {
	var request ServiceRestartStruct
	if err := decodeJSONBody(r, &request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		writeErrorJSON(w, http.StatusBadRequest, err)
		return
	}

	command := strings.TrimSpace(strings.ToLower(request.Command))
	if command == "" {
		command = "restart"
	}

	actionMessage := map[string]string{
		"start":   "Service Start Success",
		"stop":    "Service Stop Success",
		"restart": "Service Restart Success",
	}

	log.Info(fmt.Sprintf("Request Target: %v, Command: %s", request.ServiceType, command))
	for _, targetType := range request.ServiceType {
		targets, err := ResolveServiceTargets(targetType)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, err)
			return
		}

		for _, target := range targets {
			if target == "middleserver" {
				if command != "restart" {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "middleserver target only supports restart command"})
					return
				}
				if err := MiddleserverRestart(); err != nil {
					writeErrorJSON(w, http.StatusInternalServerError, err)
					return
				}
				continue
			}

			switch command {
			case "start":
				err = StartService(target)
			case "stop":
				err = StopService(target)
			case "restart":
				err = RestartService(target)
			default:
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unsupported command: %s", command)})
				return
			}

			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, err)
				return
			}
		}
	}

	response := new(ResponseListST)
	response.Code = http.StatusOK
	response.Message = actionMessage[command]
	response.Data = []string{"success"}
	response.Success = true

	writeJSON(w, http.StatusOK, response)
}

type LogRequestStruct struct {
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	LogType   []string `json:"logType"`
}

func deleteFiles(w http.ResponseWriter, r *http.Request) {
	var request DeleteFilesRequestST
	if err := decodeJSONBody(r, &request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		writeErrorJSON(w, http.StatusBadRequest, err)
		return
	}

	if len(request.Paths) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "paths is required"})
		return
	}

	result, err := DeleteFilesByList(request.Paths)
	if err != nil {
		log.Error(err)
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Delete Files Success"
	response.Data = map[string]interface{}{
		"requested": result.Requested,
		"deleted":   result.Deleted,
		"excluded":  result.Excluded,
	}
	response.Success = true

	writeJSON(w, http.StatusOK, response)
}

// Monitoring Log Download API
func downloadLog(w http.ResponseWriter, r *http.Request) {

	// Request Data 바인딩
	var request LogRequestStruct
	if err := decodeJSONBody(r, &request); err != nil {
		log.Error(fmt.Errorf("request %v", err))
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Success:    false,
			Message:    "요청 데이터가 잘못되었습니다",
			ErrorCode:  "dl-01",
			ErrorTitle: "InvalidData",
			ExtraData:  nil,
		})
		return
	}

	files, err := FilterLogFilesByDate(request.StartDate, request.EndDate)
	if err != nil {
		log.Error(err)
		writeJSON(w, http.StatusInternalServerError, PatchResponseST{
			Code:       500,
			Success:    false,
			Message:    "기간설정이 잘못되었습니다",
			ErrorCode:  "dl-02",
			ErrorTitle: "InvalidDate",
			ExtraData:  nil,
		})
		return
	}

	compressPath, err := EncryptCompress(files)
	if err != nil {
		log.Error(err)
		writeJSON(w, http.StatusInternalServerError, PatchResponseST{
			Code:       500,
			Success:    false,
			Message:    "압축파일 생성에 실패하였습니다",
			ErrorCode:  "dl-03",
			ErrorTitle: "FailCreatedCompressFile",
			ExtraData:  nil,
		})
		return
	}

	fileStat, _ := os.Stat(compressPath)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(compressPath)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileStat.Size()))
	log.Info(fmt.Sprintf("Compress Path: %s, Header: %v", compressPath, w.Header().Values("Content-Type")))
	http.ServeFile(w, r, compressPath)
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
func patchService(w http.ResponseWriter, r *http.Request) {
	log.Info("start reid patch")
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Error(err)
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "파일이 존재하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotFoundFile",
			ExtraData:  nil,
		})
		return
	}

	encryptedFile, _, err := r.FormFile("file")
	if err != nil {
		log.Error(err)
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "파일이 존재하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotFoundFile",
			ExtraData:  nil,
		})
		return
	}
	defer encryptedFile.Close()

	receiveHash := r.FormValue("hash")
	if receiveHash == "" {
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "해쉬 값이 존재하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotFoundHash",
			ExtraData:  nil,
		})
		return
	}
	log.Info(fmt.Sprintf("receive hash: %s", receiveHash))

	patchFileBuffer := bytes.Buffer{}
	patchFileSize, err := patchFileBuffer.ReadFrom(encryptedFile)
	if err != nil {
		log.Error(err)
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "해쉬 값이 존재하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotFoundHash",
			ExtraData:  nil,
		})
		return
	}

	encryptedZipFile := patchFileBuffer.Bytes()
	copyData := make([]byte, patchFileSize)
	_ = copy(copyData, encryptedZipFile)
	err = checkSha256Sum(copyData, receiveHash)
	if err != nil {
		log.Error(err)
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "해쉬 값이 일치하지 않습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "NotMatchHash",
			ExtraData:  nil,
		})
		return
	}
	log.Info(fmt.Sprintf("patch file size: %dbyte", patchFileSize))
	decryptedZipFile, err := decryptZipFile(encryptedZipFile)
	if err != nil {
		log.Error(err)
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "압축파일 추출에 실패하였습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "FailExtractedZipFile",
			ExtraData:  nil,
		})
		return
	}

	mkdirErr := os.Mkdir("temp", 0755)
	if mkdirErr != nil {
		log.Error(mkdirErr)
		writeJSON(w, http.StatusInternalServerError, PatchResponseST{
			Code:       500,
			Message:    "서버에러입니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "serverError",
			ExtraData:  nil,
		})
		return
	}

	defer func() {
		os.RemoveAll("temp")
	}()

	saveZipFileErr := os.WriteFile("temp/patch.zip", decryptedZipFile, 0755)
	if saveZipFileErr != nil {
		log.Error(saveZipFileErr)
		writeJSON(w, http.StatusInternalServerError, PatchResponseST{
			Code:       500,
			Message:    "서버에러입니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "serverError",
			ExtraData:  nil,
		})
		return
	}
	unzipErr := unzipPatchFile()
	if unzipErr != nil {
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "압축 해제에 실패하였습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "FailUnzipProcess",
			ExtraData:  nil,
		})
		return
	}

	patchErr := executePatch()
	if patchErr != nil {
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Message:    "패치를 실패하였습니다.",
			Success:    false,
			ErrorCode:  "",
			ErrorTitle: "FailPatchProcess",
			ExtraData:  nil,
		})
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
	writeJSON(w, http.StatusOK, response)
}

func decodeJSONBody(r *http.Request, dest any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(err)
	}
}

func writeErrorJSON(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
