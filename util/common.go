package util

import (
	"fmt"
	"os"
	"path"
)

var (
	configs, _             = NewConfig(configFileArg())
	rootPath               = configs.SC.Setting.RootPath
	aiPath                 = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.AI)
	frontendPath           = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.Frontend)
	backendPath            = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.Backend)
	imageProcessingPath    = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.ImageProcessing)
	mediaStreamingPath     = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.MediaStreaming)
	monitoringPath         = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.Monitoring)
	logPath                = resolveLogPath()
	networkName            = configs.SC.Setting.NetworkName
	monitoringChannelName  = "selective_server_status"
	aiLogName              = configs.SC.Category.AI
	frontendLogName        = configs.SC.Category.Frontend
	backendLogName         = configs.SC.Category.Backend
	pastBackendLogName     = configs.SC.Category.PastBackend
	backendAuthLogName     = fmt.Sprintf("%s_auth", configs.SC.Category.Backend)
	backendGatewayLogName  = fmt.Sprintf("%s_gateway", configs.SC.Category.Backend)
	backendMainLogName     = fmt.Sprintf("%s_main", configs.SC.Category.Backend)
	backendAiLogName       = fmt.Sprintf("%s_selective-ai", configs.SC.Category.Backend)
	backendSettingsLogName = fmt.Sprintf("%s_settings", configs.SC.Category.Backend)
	backendUserLogName     = fmt.Sprintf("%s_user", configs.SC.Category.Backend)
	backendVmsLogName      = fmt.Sprintf("%s_vms", configs.SC.Category.Backend)
	imageProcessingLogName = configs.SC.Category.ImageProcessing
	mediaStreamingLogName  = configs.SC.Category.MediaStreaming
	monitoringLogName      = configs.SC.Category.Monitoring
	log                    = NewLogger(fmt.Sprintf("%s/%s", logPath, "monitoring"), monitoringLogName)
	encryptKey             = []byte(configs.SC.Setting.Token) // ?뷀샇????
)

// activeServices serverType 값에 따라 모니터링/제어 대상 서비스 목록을 반환한다.
//
//	main    : 컨트롤 + 분석 통합 서버 (전체 서비스)
//	analyze : 분석 전용 서버 (analyze, downloader)
func activeServices() []string {
	s := configs.SC.Setting
	switch s.ServerType {
	case "analyze":
		return collectServiceNames(
			s.AnalyzeServiceName,
			s.DownloaderServiceName,
		)
	default: // "main" (기본값) — 전체 서비스 포함
		return collectServiceNames(
			s.MediaStreamingServiceName,
			s.BackendServiceName,
			s.OSRMServiceName,
			s.RedisServiceName,
			s.MiddleServerServiceName,
			s.MiddleServiceName,
			s.AnalyzeServiceName,
			s.DownloaderServiceName,
		)
	}
}

// ServerRoleForAPI API 응답용 serverType 변환. 내부 "analyze" → 외부 "agent"
func ServerRoleForAPI() string {
	if configs.SC.Setting.ServerType == "analyze" {
		return "agent"
	}
	return "main"
}

// activeDockerServices serverType 별 docker 서비스 모니터링 대상 키워드 목록 반환.
// 키워드 정의: "route" (OSRM+Nominatim → omeye3.route.service), "redis" (Redis+PING)
func activeDockerServices() []string {
	if configs.SC.Setting.ServerType == "analyze" {
		return configs.SC.Setting.AnalyzeDockerServices
	}
	return configs.SC.Setting.MainDockerServices
}

func configFileArg() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return ""
}

// resolveLogPath config의 logPath가 지정되어 있으면 그 값, 아니면 {rootPath}/log
func resolveLogPath() string {
	if p := configs.SC.Setting.LogPath; p != "" {
		return p
	}
	//return fmt.Sprintf("%s/%s", configs.SC.Setting.RootPath, "log", "monitoring")
	return path.Join(configs.SC.Setting.RootPath, "log", "monitoring")
}
