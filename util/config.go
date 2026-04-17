package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func NewConfig(active string) (Config, error) {
	var config Config
	confPath := resolveConfigPath(active)

	raw, err := os.ReadFile(confPath)
	if err != nil {
		return config, fmt.Errorf("config 파일을 읽을 수 없습니다 (%s): %w", confPath, err)
	}

	var fileCfg configFile
	if err := yaml.Unmarshal(raw, &fileCfg); err != nil {
		return config, fmt.Errorf("config 파일 파싱 실패 (%s): %w", confPath, err)
	}

	config.SC = SettingConfig{
		Setting:  fileCfg.Setting,
		Version:  fileCfg.Version,
		Category: fileCfg.Category,
	}
	config.Redis = fileCfg.Redis

	return config, nil
}

func resolveConfigPath(active string) string {
	pwd, _ := os.Getwd()
	confDir := filepath.Join(pwd, "conf.d")

	target := strings.TrimSpace(active)
	if target == "" {
		return filepath.Join(confDir, "config.yml")
	}

	targetLower := strings.ToLower(target)
	if strings.HasSuffix(targetLower, ".yml") || strings.HasSuffix(targetLower, ".yaml") {
		return filepath.Join(confDir, target)
	}

	candidates := []string{
		filepath.Join(confDir, fmt.Sprintf("config-%s.yml", target)),
		filepath.Join(confDir, fmt.Sprintf("config-%s.yaml", target)),
		filepath.Join(confDir, "config.yml"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return candidates[0]
}

type Config struct {
	SC    SettingConfig
	Redis RedisConfig
}

type SettingConfig struct {
	Setting  SettingSection
	Version  VersionSection
	Category CategorySection
}

type SettingSection struct {
	RootPath                   string `yaml:"rootPath"`
	ServerIP                   string `yaml:"serverIP"`
	ServerPort                 int    `yaml:"serverPort"`
	UserPassword               string `yaml:"userPassword"`
	NetworkName                string `yaml:"networkName"`
	ImageProcessingServiceName string `yaml:"imageProcessingServiceName"`
	MediaStreamingServiceName  string `yaml:"mediaStreamingServiceName"`
	BackendServiceName         string `yaml:"backendServiceName"`
	OSRMServiceName            string `yaml:"osrmServiceName"`
	OSRMContainerName          string `yaml:"osrmContainerName"`
	NominatimContainerName     string `yaml:"nominatimContainerName"`
	RedisServiceName           string `yaml:"redisServiceName"`
	MiddleServerServiceName    string `yaml:"middleServerServiceName"`
	AiServiceName              string `yaml:"aiServiceName"`
	MiddleServiceName          string `yaml:"middleServiceName"`
	Token                      string `yaml:"token"`
	ServerType                 string `yaml:"serverType"`
	ZipRetentionHours          int    `yaml:"zipRetentionHours"`
	ZipCleanupIntervalMinutes  int    `yaml:"zipCleanupIntervalMinutes"`

	// 스토리지 삭제 API 설정 (없으면 기본값 적용)
	StorageRootDir       string   `yaml:"storageRootDir"`
	StorageProtectedDirs []string `yaml:"storageProtectedDirs"`
	StorageReidResultDir string   `yaml:"storageReidResultDir"`
}

type VersionSection struct {
	Frontend        string `yaml:"frontend"`
	Backend         string `yaml:"backend"`
	AI              string `yaml:"ai"`
	ImageProcessing string `yaml:"imageProcessing"`
	MediaStreaming  string `yaml:"mediaStreaming"`
}

type CategorySection struct {
	AI              string `yaml:"ai"`
	Backend         string `yaml:"backend"`
	PastBackend     string `yaml:"past_backend"`
	Frontend        string `yaml:"frontend"`
	ImageProcessing string `yaml:"imageProcessing"`
	MediaStreaming  string `yaml:"mediaStreaming"`
	Monitoring      string `yaml:"monitoring"`
}

type RedisConfig struct {
	RedisContainerName string `yaml:"RedisContainerName"`
	RedisHost          string `yaml:"RedisHost"`
	RedisPort          int    `yaml:"RedisPort"`
	Username           string `yaml:"Username"`
	Password           string `yaml:"Password"`
}

type configFile struct {
	Setting  SettingSection  `yaml:"SETTING"`
	Version  VersionSection  `yaml:"VERSION"`
	Category CategorySection `yaml:"CATEGORY"`
	Redis    RedisConfig     `yaml:"REDIS"`
}
