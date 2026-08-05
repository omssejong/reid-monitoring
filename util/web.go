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
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// appContext 비동기 Job 실행에 사용할 앱 전역 context (main에서 SetAppContext 호출)
var appContext = context.Background()

// SetAppContext 앱 수명 컨텍스트 주입 (main.go에서 호출)
func SetAppContext(ctx context.Context) {
	appContext = ctx
}

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

	log.Info(fmt.Sprintf("Build Info: %s", GetBuildInfo()))

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
		reidV1.Delete("/storage", deleteStorage)
		reidV1.Get("/jobs/{jobId}", getStorageJob)
		reidV1.Post("/upload/patch", patchService)
		reidV1.Get("/thresholds", getThresholds)
		reidV1.Post("/thresholds", updateThresholds)
		reidV1.Get("/alerts", getAlerts)
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
	serverInfoDict["serverRole"] = ServerRoleForAPI() // 내부 analyze → 외부 agent
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

	ch := GetCollector().Subscribe()
	defer GetCollector().Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case info := <-ch:
			payload, err := json.Marshal(info)
			if err != nil {
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
	snap := GetCollector().Snapshot()
	if snap == nil {
		writeErrorJSON(w, http.StatusServiceUnavailable, errors.New("system info not ready"))
		return
	}
	writeJSON(w, http.StatusOK, snap)
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

		existing, missing := filterExistingServices(targets)
		if len(missing) > 0 {
			log.Warn(fmt.Sprintf("서비스 누락 - missing: %v, requested: %s", missing, targetType))
		}
		if len(existing) > 0 {
			log.Info(fmt.Sprintf("실행 대상 - targets: %v, command: %s", existing, command))
		}

		for _, target := range existing {
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

// deleteStorage 정책 기반 스토리지 삭제 (percent 또는 from/to).
//
//	DELETE /monitoring/mgmt/storage?percent={0~100}
//	DELETE /monitoring/mgmt/storage?from=yyyyMMdd&to=yyyyMMdd
func deleteStorage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	percentStr := strings.TrimSpace(q.Get("percent"))
	fromStr := strings.TrimSpace(q.Get("from"))
	toStr := strings.TrimSpace(q.Get("to"))

	if percentStr == "" && fromStr == "" && toStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "percent or (from, to) is required",
		})
		return
	}

	store := GetStorageJobStore()
	if store == nil {
		writeErrorJSON(w, http.StatusInternalServerError, fmt.Errorf("storage job store not initialized"))
		return
	}

	// percent 모드 우선
	if percentStr != "" {
		deleteStoragePercent(w, r, store, percentStr)
		return
	}

	// date 모드
	deleteStorageDate(w, r, store, fromStr, toStr)
}

func deleteStoragePercent(w http.ResponseWriter, r *http.Request, store *StorageJobStore, percentStr string) {
	percent, err := strconv.ParseFloat(percentStr, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid percent: %s", percentStr),
		})
		return
	}
	if percent < 0 || percent > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("percent out of range: %.2f (must be 0~100)", percent),
		})
		return
	}

	job, info, err := store.StartPercent(appContext, percent)
	if err != nil {
		if errors.Is(err, ErrStorageJobBusy) {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error":   "another storage job is running",
				"current": store.currentJobID(),
			})
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	// 이미 목표 달성 → 동기 200 + 현재 상태
	if job == nil && info != nil {
		response := new(ResponseST)
		response.Code = http.StatusOK
		response.Message = "Target already met"
		response.Data = serverStorageInfoToMap(*info)
		response.Success = true
		writeJSON(w, http.StatusOK, response)
		return
	}

	// 비동기 Job 수락
	log.Info(fmt.Sprintf("storage percent accepted: id=%s target=%.2f", job.ID, percent))
	response := new(ResponseST)
	response.Code = http.StatusAccepted
	response.Message = "Storage job accepted"
	response.Data = map[string]interface{}{
		"mode":      "percent",
		"jobId":     job.ID,
		"statusUrl": fmt.Sprintf("/monitoring/mgmt/jobs/%s", job.ID),
	}
	response.Success = true
	writeJSON(w, http.StatusAccepted, response)
}

func deleteStorageDate(w http.ResponseWriter, r *http.Request, store *StorageJobStore, fromStr, toStr string) {
	if fromStr == "" || toStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "both from and to are required",
		})
		return
	}

	const layout = "20060102"
	from, err := time.Parse(layout, fromStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid from: %s (expected yyyyMMdd)", fromStr),
		})
		return
	}
	to, err := time.Parse(layout, toStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid to: %s (expected yyyyMMdd)", toStr),
		})
		return
	}
	if from.After(to) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("from is after to: %s > %s", fromStr, toStr),
		})
		return
	}

	// 날짜 범위: from 00:00:00 ~ to+1일 00:00:00 (end exclusive)
	toExclusive := to.Add(24 * time.Hour)

	job, err := store.StartDate(appContext, from, toExclusive, fromStr, toStr)
	if err != nil {
		if errors.Is(err, ErrStorageJobBusy) {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error":   "another storage job is running",
				"current": store.currentJobID(),
			})
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	log.Info(fmt.Sprintf("storage date accepted: id=%s from=%s to=%s", job.ID, fromStr, toStr))
	response := new(ResponseST)
	response.Code = http.StatusAccepted
	response.Message = "Storage job accepted"
	response.Data = map[string]interface{}{
		"mode":      "date",
		"jobId":     job.ID,
		"statusUrl": fmt.Sprintf("/monitoring/mgmt/jobs/%s", job.ID),
	}
	response.Success = true
	writeJSON(w, http.StatusAccepted, response)
}

// getStorageJob Job 상태 조회
func getStorageJob(w http.ResponseWriter, r *http.Request) {
	jobId := chi.URLParam(r, "jobId")
	if jobId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "jobId is required"})
		return
	}

	store := GetStorageJobStore()
	if store == nil {
		writeErrorJSON(w, http.StatusInternalServerError, fmt.Errorf("storage job store not initialized"))
		return
	}

	job, ok := store.GetJob(jobId)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}

	snap := job.Snapshot()
	data := map[string]interface{}{
		"jobId":        snap.ID,
		"mode":         snap.Mode,
		"status":       snap.Status,
		"progress":     snap.Progress,
		"deletedCount": snap.DeletedCount,
		"deletedBytes": snap.DeletedBytes,
		"startedAt":    snap.StartedAt,
		"finishedAt":   snap.FinishedAt,
	}
	switch snap.Mode {
	case "percent":
		data["targetPercent"] = snap.TargetPercent
		data["startPercent"] = snap.StartPercent
		data["currentPercent"] = snap.CurrentPercent
		if snap.TargetReached != nil {
			data["targetReached"] = *snap.TargetReached
		}
	case "date":
		data["from"] = snap.From
		data["to"] = snap.To
		data["totalCandidates"] = snap.TotalCandidates
		data["processed"] = snap.Processed
	case "retention":
		data["retentionDays"] = snap.RetentionDays
		data["cutoff"] = snap.Cutoff
		data["totalCandidates"] = snap.TotalCandidates
		data["processed"] = snap.Processed
	}
	if snap.ErrorMsg != "" {
		data["error"] = snap.ErrorMsg
	}
	if snap.StorageInfo != nil {
		data["storageInfo"] = serverStorageInfoToMap(*snap.StorageInfo)
	}

	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Storage Job Status"
	response.Data = data
	response.Success = true
	writeJSON(w, http.StatusOK, response)
}

// serverStorageInfoToMap ServerStorageInfo를 map으로 변환 (응답용)
func serverStorageInfoToMap(info ServerStorageInfo) map[string]interface{} {
	return map[string]interface{}{
		"total":        info.Total,
		"used":         info.Used,
		"avail":        info.Avail,
		"usedPercent":  info.UsedPercent,
		"availPercent": info.AvailPercent,
	}
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

	// 날짜 범위에 해당하는 로그 파일이 0건이면 별도 에러 반환
	if len(files) == 0 {
		log.Info(fmt.Sprintf("no log files matched range: start=%s end=%s", request.StartDate, request.EndDate))
		writeJSON(w, http.StatusBadRequest, PatchResponseST{
			Code:       400,
			Success:    false,
			Message:    "해당 기간에 로그가 존재하지 않습니다",
			ErrorCode:  "dl-04",
			ErrorTitle: "NoLogsFound",
			ExtraData:  nil,
		})
		return
	}

	compressPath, err := EncryptCompress(files, rootPath)
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

// getThresholds 현재 임계값 조회
func getThresholds(w http.ResponseWriter, r *http.Request) {
	store := GetThresholdStore()
	if store == nil {
		writeErrorJSON(w, http.StatusInternalServerError, fmt.Errorf("threshold store not initialized"))
		return
	}
	disk := store.GetDisk()

	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Thresholds"
	response.Data = map[string]interface{}{
		"disk": disk,
	}
	response.Success = true
	writeJSON(w, http.StatusOK, response)
}

// updateThresholds 임계값 변경 요청 처리
func updateThresholds(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Disk *DiskThreshold `json:"disk"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		log.Error(fmt.Errorf("update thresholds request: %v", err))
		writeErrorJSON(w, http.StatusBadRequest, err)
		return
	}

	store := GetThresholdStore()
	if store == nil {
		writeErrorJSON(w, http.StatusInternalServerError, fmt.Errorf("threshold store not initialized"))
		return
	}

	if request.Disk != nil {
		if err := store.UpdateDisk(*request.Disk); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, err)
			return
		}
		log.Info(fmt.Sprintf("disk threshold updated: warning=%.2f", request.Disk.Warning))
	}

	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Thresholds Updated"
	response.Data = map[string]interface{}{
		"disk": store.GetDisk(),
	}
	response.Success = true
	writeJSON(w, http.StatusOK, response)
}

// getAlerts 현재 디스크 사용률이 임계값을 초과했는지 조회
func getAlerts(w http.ResponseWriter, r *http.Request) {
	store := GetThresholdStore()
	if store == nil {
		writeErrorJSON(w, http.StatusInternalServerError, fmt.Errorf("threshold store not initialized"))
		return
	}

	usage, err := GetDiskUsage()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	breach := store.CheckDisk(usage)
	data := map[string]interface{}{
		"currentValue": usage,
	}
	if breach != nil {
		data["hasBreach"] = true
		data["metric"] = breach.Metric
		data["threshold"] = breach.Threshold
	} else {
		data["hasBreach"] = false
	}

	response := new(ResponseST)
	response.Code = http.StatusOK
	response.Message = "Alerts"
	response.Data = data
	response.Success = true
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
