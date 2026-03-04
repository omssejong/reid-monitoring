package util

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func NewConfig(active string) (Config, error) {
	var config Config

	pwd, err := os.Getwd()
	if err != nil {
		return config, err
	}

	raw, err := os.ReadFile(filepath.Join(pwd, "conf.d", fmt.Sprintf("config-%s.yml", active)))
	if err != nil {
		return config, err
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
	MiddleServerServiceName    string `yaml:"middleServerServiceName"`
	AiServiceName              string `yaml:"aiServiceName"`
	MiddleServiceName          string `yaml:"middleServiceName"`
	Token                      string `yaml:"token"`
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
