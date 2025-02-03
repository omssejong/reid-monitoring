package util

import (
	"fmt"
	"os"
)

var (
	configs, _             = NewConfig(os.Args[1])
	rootPath               = configs.SC.Setting.RootPath
	aiPath                 = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.AI)
	frontendPath           = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.Frontend)
	backendPath            = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.Backend)
	imageProcessingPath    = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.ImageProcessing)
	mediaStreamingPath     = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.MediaStreaming)
	monitoringPath         = fmt.Sprintf("%s/%s", rootPath, configs.SC.Category.Monitoring)
	logPath                = fmt.Sprintf("%s/%s", rootPath, "log")
	redisClient            = NewRedisClient(fmt.Sprintf("%s:%d", configs.AC.Redis.RedisHost, configs.AC.Redis.RedisPort), configs.AC.Redis.Password)
	networkName            = configs.SC.Setting.NetworkName
	monitoringChannelName  = "selective_server_status"
	aiLogName              = configs.SC.Category.AI
	frontendLogName        = configs.SC.Category.Frontend
	backendLogName         = configs.SC.Category.Backend
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
	log                    = NewLogger(logPath, monitoringLogName)
	password               = configs.SC.Setting.UserPassword
	encryptKey             = []byte(configs.SC.Setting.Token) // 암호화 키
	monitoringTarget       = fmt.Sprintf(
		"%s,%s,%s",
		configs.SC.Setting.MediaStreamingServiceName,
		configs.SC.Setting.BackendServiceName,
		configs.SC.Setting.AiServiceName,
	)
)
