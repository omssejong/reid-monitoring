package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func NewConfig(active string) (Config, error) {
	var config Config
	confPath := resolveConfigPath(active)

	raw, err := os.ReadFile(confPath)
	if err != nil {
		return config, err
	}

	ext := strings.ToLower(filepath.Ext(confPath))
	if ext == ".env" {
		cfg, err := newConfigFromDotEnv(raw)
		if err != nil {
			return config, err
		}
		return cfg, nil
	}

	var fileCfg configFile
	if err := yaml.Unmarshal(raw, &fileCfg); err != nil {
		return config, err
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
		return filepath.Join(confDir, ".env")
	}

	targetLower := strings.ToLower(target)
	if strings.HasSuffix(targetLower, ".env") || strings.HasSuffix(targetLower, ".yml") || strings.HasSuffix(targetLower, ".yaml") {
		return filepath.Join(confDir, target)
	}

	candidates := []string{
		filepath.Join(confDir, fmt.Sprintf("config-%s.yml", target)),
		filepath.Join(confDir, fmt.Sprintf("config-%s.env", target)),
		filepath.Join(confDir, ".env"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return candidates[0]
}

func newConfigFromDotEnv(raw []byte) (Config, error) {
	var config Config
	values, err := parseDotEnv(string(raw))
	if err != nil {
		return config, err
	}

	serverPort, err := getIntEnv(values, "SERVER_PORT", "SC_SETTING_SERVER_PORT")
	if err != nil {
		return config, err
	}
	redisPort, err := getIntEnv(values, "REDIS_PORT", "SC_REDIS_PORT")
	if err != nil {
		return config, err
	}
	zipRetentionHours := getOptionalIntEnv(values, 1, "ZIP_RETENTION_HOURS", "SC_SETTING_ZIP_RETENTION_HOURS")
	zipCleanupIntervalMinutes := getOptionalIntEnv(values, 10, "ZIP_CLEANUP_INTERVAL_MINUTES", "SC_SETTING_ZIP_CLEANUP_INTERVAL_MINUTES")

	config.SC = SettingConfig{
		Setting: SettingSection{
			RootPath:                   getStringEnv(values, "ROOT_PATH", "SC_SETTING_ROOT_PATH"),
			ServerIP:                   getStringEnv(values, "SERVER_IP", "SC_SETTING_SERVER_IP"),
			ServerPort:                 serverPort,
			UserPassword:               getStringEnv(values, "USER_PASSWORD", "SC_SETTING_USER_PASSWORD"),
			NetworkName:                getStringEnv(values, "NETWORK_NAME", "SC_SETTING_NETWORK_NAME"),
			ImageProcessingServiceName: getStringEnv(values, "IMAGE_PROCESSING_SERVICE_NAME", "SC_SETTING_IMAGE_PROCESSING_SERVICE_NAME"),
			MediaStreamingServiceName:  getStringEnv(values, "MEDIA_STREAMING_SERVICE_NAME", "SC_SETTING_MEDIA_STREAMING_SERVICE_NAME"),
			BackendServiceName:         getStringEnv(values, "BACKEND_SERVICE_NAME", "SC_SETTING_BACKEND_SERVICE_NAME"),
			OSRMServiceName:            getStringEnv(values, "OSRM_SERVICE_NAME", "SC_SETTING_OSRM_SERVICE_NAME"),
			OSRMContainerName:          getStringEnv(values, "OSRM_CONTAINER_NAME", "SC_SETTING_OSRM_CONTAINER_NAME"),
			NominatimContainerName:     getStringEnv(values, "NOMINATIM_CONTAINER_NAME", "SC_SETTING_NOMINATIM_CONTAINER_NAME"),
			RedisServiceName:           getStringEnv(values, "REDIS_SERVICE_NAME", "SC_SETTING_REDIS_SERVICE_NAME"),
			MiddleServerServiceName:    getStringEnv(values, "MIDDLE_SERVER_SERVICE_NAME", "SC_SETTING_MIDDLE_SERVER_SERVICE_NAME"),
			AiServiceName:              getStringEnv(values, "AI_SERVICE_NAME", "SC_SETTING_AI_SERVICE_NAME"),
			MiddleServiceName:          getStringEnv(values, "MIDDLE_SERVICE_NAME", "SC_SETTING_MIDDLE_SERVICE_NAME"),
			Token:                      getStringEnv(values, "TOKEN", "SC_SETTING_TOKEN"),
			ServerType:                 getStringEnv(values, "SERVER_TYPE", "SC_SETTING_SERVER_TYPE"),
			ZipRetentionHours:          zipRetentionHours,
			ZipCleanupIntervalMinutes:  zipCleanupIntervalMinutes,
		},
		Version: VersionSection{
			Frontend:        getStringEnv(values, "VERSION_FRONTEND", "SC_VERSION_FRONTEND"),
			Backend:         getStringEnv(values, "VERSION_BACKEND", "SC_VERSION_BACKEND"),
			AI:              getStringEnv(values, "VERSION_AI", "SC_VERSION_AI"),
			ImageProcessing: getStringEnv(values, "VERSION_IMAGEPROCESSING", "SC_VERSION_IMAGE_PROCESSING"),
			MediaStreaming:  getStringEnv(values, "VERSION_MEDIASTREAMING", "SC_VERSION_MEDIA_STREAMING"),
		},
		Category: CategorySection{
			AI:              getStringEnv(values, "CATEGORY_AI", "SC_CATEGORY_AI"),
			Backend:         getStringEnv(values, "CATEGORY_BACKEND", "SC_CATEGORY_BACKEND"),
			PastBackend:     getStringEnv(values, "CATEGORY_PAST_BACKEND", "SC_CATEGORY_PAST_BACKEND"),
			Frontend:        getStringEnv(values, "CATEGORY_FRONTEND", "SC_CATEGORY_FRONTEND"),
			ImageProcessing: getStringEnv(values, "CATEGORY_IMAGEPROCESSING", "SC_CATEGORY_IMAGE_PROCESSING"),
			MediaStreaming:  getStringEnv(values, "CATEGORY_MEDIASTREAMING", "SC_CATEGORY_MEDIA_STREAMING"),
			Monitoring:      getStringEnv(values, "CATEGORY_MONITORING", "SC_CATEGORY_MONITORING"),
		},
	}

	config.Redis = RedisConfig{
		RedisContainerName: getStringEnv(values, "REDIS_CONTAINER_NAME", "SC_REDIS_CONTAINER_NAME"),
		RedisHost:          getStringEnv(values, "REDIS_HOST", "SC_REDIS_HOST"),
		RedisPort:          redisPort,
		Username:           getStringEnv(values, "REDIS_USERNAME", "SC_REDIS_USERNAME"),
		Password:           getStringEnv(values, "REDIS_PASSWORD", "SC_REDIS_PASSWORD"),
	}

	return config, nil
}

func getStringEnv(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if val, ok := values[key]; ok {
			return val
		}
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func getIntEnv(values map[string]string, keys ...string) (int, error) {
	raw, ok := "", false
	for _, key := range keys {
		if raw, ok = values[key]; ok && raw != "" {
			break
		}
		if env := os.Getenv(key); env != "" {
			raw, ok = env, true
			break
		}
	}
	if !ok {
		return 0, fmt.Errorf("missing config key: %s", keys[0])
	}

	val, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %v", keys[0], err)
	}
	return val, nil
}

func getOptionalIntEnv(values map[string]string, defaultValue int, keys ...string) int {
	for _, key := range keys {
		if raw, ok := values[key]; ok && strings.TrimSpace(raw) != "" {
			if val, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
				return val
			}
			return defaultValue
		}
		if env := os.Getenv(key); strings.TrimSpace(env) != "" {
			if val, err := strconv.Atoi(strings.TrimSpace(env)); err == nil {
				return val
			}
			return defaultValue
		}
	}
	return defaultValue
}

func parseDotEnv(raw string) (map[string]string, error) {
	values := make(map[string]string)
	for i, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("invalid env line %d: %q", i+1, line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			value = unquoted
		}
		values[key] = value
	}
	return values, nil
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
