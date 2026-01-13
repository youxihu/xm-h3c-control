package config

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// ExpiryTimeConfig 过期时间配置
type ExpiryTimeConfig struct {
	Hour   int `yaml:"hour"`   // 过期小时 (0-23)
	Minute int `yaml:"minute"` // 过期分钟 (0-59)
}

// RouterConfig 路由器配置
type RouterConfig struct {
	Host                     string           `yaml:"host"`
	User                     string           `yaml:"user"`
	Passwd                   string           `yaml:"passwd"`
	ReminderBeforeExpiration int              `yaml:"Reminder_before_expiration"`
	ExpiryTime               ExpiryTimeConfig `yaml:"expiry_time"`
}

// DingTalkGroupConfig 钉钉群组配置
type DingTalkGroupConfig struct {
	Webhook string   `yaml:"webhook"`
	Secret  string   `yaml:"secret"`
	Name    string   `yaml:"name"`
	Servers []string `yaml:"servers,omitempty"` // 只有groups才有servers字段
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	Default DingTalkGroupConfig            `yaml:"default"`
	Groups  map[string]DingTalkGroupConfig `yaml:"groups"`
}

// Config 应用配置
type Config struct {
	Router   RouterConfig   `yaml:"h3c-msr2600"`
	DingTalk DingTalkConfig `yaml:"dingtalk"`
}

// LoadConfig 加载配置文件
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}