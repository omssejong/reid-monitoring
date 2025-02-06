package util

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

func NewConfig(active string) (Config, error) {

	// 기본 설정 구조체 생성
	var config Config

	// 현제 경로 확인
	pwd, err := os.Getwd()
	if err != nil {
		return config, err
	}

	// YAML 파일 로드
	yml, err := os.ReadFile(path.Join(pwd, "conf.d", fmt.Sprintf("config-%s.yml", active)))
	if err != nil {
		return config, err
	}

	// APP 설정 구조체 생성
	var appConfig AppConfig
	err = yaml.Unmarshal(yml, &appConfig)
	if err != nil {
		return config, err
	}

	var settingConfig SettingConfig
	if err := ini.MapTo(&settingConfig, appConfig.Config.ConfigPath); err != nil {
		return config, err
	}

	config.AC = appConfig
	config.SC = settingConfig

	return config, nil
}

/**
 * Config
 * 설정 파일 구조체
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
type Config struct {
	AC AppConfig
	SC SettingConfig
}

/**
 * SettingConfig
 * 글로벌 설정 파일 구조체
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
type SettingConfig struct {
	Setting struct {
		RootPath                   string `ini:"rootPath"`
		ServerIP                   string `ini:"serverIP"`
		ServerPort                 int    `ini:"serverPort"`
		UserPassword               string `ini:"userPassword"`
		NetworkName                string `ini:"networkName"`
		ImageProcessingServiceName string `ini:"imageProcessingServiceName"`
		MediaStreamingServiceName  string `ini:"mediaStreamingServiceName"`
		BackendServiceName         string `ini:"backendServiceName"`
		AiServiceName              string `ini:"aiServiceName"`
		MiddleServiceName          string `ini:"middleServiceName"`
		Token                      string `ini:"token"`
	} `ini:"SETTING"`
	Version struct {
		Frontend        string `ini:"frontend"`
		Backend         string `ini:"backend"`
		AI              string `ini:"ai"`
		ImageProcessing string `ini:"imageProcessing"`
		MediaStreaming  string `ini:"mediaStreaming"`
	} `ini:"VERSION"`
	Category struct {
		AI              string `ini:"ai"`
		Backend         string `ini:"backend"`
		Frontend        string `ini:"frontend"`
		ImageProcessing string `ini:"imageProcessing"`
		MediaStreaming  string `ini:"mediaStreaming"`
		Monitoring      string `ini:"monitoring"`
	} `ini:"CATEGORY"`
}

/**
 * AppConfig
 * 앱 설정 파일 구조체
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.01.12
 */
type AppConfig struct {
	Config AppConfigChild `yaml:"CONFIG"`
	Redis  RedisConfig    `yaml:"REDIS"`
}

// AppConfigChild
type AppConfigChild struct {
	ConfigPath string `yaml:"ConfigPath"`
}

// RedisConfig
type RedisConfig struct {
	RedisContainerName string `yaml:"RedisContainerName"`
	RedisHost          string `yaml:"RedisHost"`
	RedisPort          int    `yaml:"RedisPort"`
	Username           string `yaml:"Username"`
	Password           string `yaml:"Password"`
}
