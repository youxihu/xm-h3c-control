package config

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	H3CMSR2600 RouterConfig `yaml:"h3c-msr2600"`
	Redis      RedisConfig  `yaml:"redis"`
	Cache      CacheConfig  `yaml:"cache"`
}

// RouterConfig 路由器配置
type RouterConfig struct {
	Host       string `yaml:"host"`
	User       string `yaml:"user"`
	Passwd     string `yaml:"passwd"`
	ExternalIP string `yaml:"external_ip"`
	ExpiryTime struct {
		Hour   int `yaml:"hour"`
		Minute int `yaml:"minute"`
	} `yaml:"expiry_time"`
	ReminderBeforeExpiration int `yaml:"Reminder_before_expiration"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	UpdateIntervalMinutes int  `yaml:"update_interval_minutes"`
	TestIntervalMinutes   int  `yaml:"test_interval_minutes"`
	UseTestInterval       bool `yaml:"use_test_interval"`
}

// RevertConfig 端口映射配置
type RevertConfig struct {
	PortMappings     map[int][]int                       `yaml:"port_mappings"`
	PortDescriptions map[int]PortDescription             `yaml:"port_descriptions"`
	Hosts            map[string]HostConfig               `yaml:"hosts"`
}

// PortDescription 端口描述配置
type PortDescription struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// HostConfig 主机配置
type HostConfig struct {
	Env      string         `yaml:"env"`
	Services map[string]int `yaml:"services"`
}

// LoadConfig 加载配置文件
func LoadConfig() (*Config, *RevertConfig, error) {
	var config Config
	var revertConfig RevertConfig

	// 加载主配置文件
	data, err := ioutil.ReadFile("config/config.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 加载revert配置文件
	revertData, err := ioutil.ReadFile("config/revert.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("读取revert配置文件失败: %v", err)
	}

	if err := yaml.Unmarshal(revertData, &revertConfig); err != nil {
		return nil, nil, fmt.Errorf("解析revert配置文件失败: %v", err)
	}

	log.Printf("加载配置成功: 路由器地址 %s, 用户 %s, 外网IP %s", 
		config.H3CMSR2600.Host, config.H3CMSR2600.User, config.H3CMSR2600.ExternalIP)
	log.Printf("端口映射配置: %+v", revertConfig.PortMappings)
	log.Printf("Redis配置: %s:%d", config.Redis.Host, config.Redis.Port)

	return &config, &revertConfig, nil
}