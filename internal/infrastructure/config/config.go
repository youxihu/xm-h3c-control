package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net"
	"net/url"
)

// ExpiryTimeConfig 过期时间配置
type ExpiryTimeConfig struct {
	Hour   int `yaml:"hour"`   // 过期小时 (0-23)
	Minute int `yaml:"minute"` // 过期分钟 (0-59)
}

// Validate 验证过期时间配置
func (e *ExpiryTimeConfig) Validate() error {
	if e.Hour < 0 || e.Hour > 23 {
		return fmt.Errorf("过期小时必须在0-23之间，当前值: %d", e.Hour)
	}
	if e.Minute < 0 || e.Minute > 59 {
		return fmt.Errorf("过期分钟必须在0-59之间，当前值: %d", e.Minute)
	}
	return nil
}

// RouterConfig 路由器配置
type RouterConfig struct {
	Host                     string           `yaml:"host"`
	User                     string           `yaml:"user"`
	Passwd                   string           `yaml:"passwd"`
	ReminderBeforeExpiration int              `yaml:"Reminder_before_expiration"`
	ExpiryTime               ExpiryTimeConfig `yaml:"expiry_time"`
}

// Validate 验证路由器配置
func (r *RouterConfig) Validate() error {
	if r.Host == "" {
		return fmt.Errorf("路由器主机地址不能为空")
	}
	
	// 验证IP地址格式
	if net.ParseIP(r.Host) == nil {
		return fmt.Errorf("无效的IP地址格式: %s", r.Host)
	}
	
	if r.User == "" {
		return fmt.Errorf("路由器用户名不能为空")
	}
	
	if r.Passwd == "" {
		return fmt.Errorf("路由器密码不能为空")
	}
	
	if r.ReminderBeforeExpiration <= 0 {
		return fmt.Errorf("提醒天数必须大于0，当前值: %d", r.ReminderBeforeExpiration)
	}
	
	return r.ExpiryTime.Validate()
}

// DingTalkGroupConfig 钉钉群组配置
type DingTalkGroupConfig struct {
	Webhook string   `yaml:"webhook"`
	Secret  string   `yaml:"secret"`
	Name    string   `yaml:"name"`
	Servers []string `yaml:"servers,omitempty"` // 只有groups才有servers字段
}

// Validate 验证钉钉群组配置
func (d *DingTalkGroupConfig) Validate() error {
	if d.Webhook == "" {
		return fmt.Errorf("钉钉Webhook不能为空")
	}
	
	// 验证Webhook URL格式
	if _, err := url.Parse(d.Webhook); err != nil {
		return fmt.Errorf("无效的Webhook URL格式: %s", d.Webhook)
	}
	
	if d.Secret == "" {
		return fmt.Errorf("钉钉Secret不能为空")
	}
	
	if d.Name == "" {
		return fmt.Errorf("钉钉群组名称不能为空")
	}
	
	// 验证服务器IP地址
	for _, server := range d.Servers {
		if net.ParseIP(server) == nil {
			return fmt.Errorf("无效的服务器IP地址: %s", server)
		}
	}
	
	return nil
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	Default DingTalkGroupConfig            `yaml:"default"`
	Groups  map[string]DingTalkGroupConfig `yaml:"groups"`
}

// Validate 验证钉钉配置
func (d *DingTalkConfig) Validate() error {
	if err := d.Default.Validate(); err != nil {
		return fmt.Errorf("默认钉钉配置验证失败: %v", err)
	}
	
	for groupName, groupConfig := range d.Groups {
		if err := groupConfig.Validate(); err != nil {
			return fmt.Errorf("钉钉群组 '%s' 配置验证失败: %v", groupName, err)
		}
	}
	
	return nil
}

// Config 应用配置
type Config struct {
	Router   RouterConfig   `yaml:"h3c-msr2600"`
	DingTalk DingTalkConfig `yaml:"dingtalk"`
}

// Validate 验证整个配置
func (c *Config) Validate() error {
	if err := c.Router.Validate(); err != nil {
		return fmt.Errorf("路由器配置验证失败: %v", err)
	}
	
	if err := c.DingTalk.Validate(); err != nil {
		return fmt.Errorf("钉钉配置验证失败: %v", err)
	}
	
	return nil
}

// LoadConfig 加载配置文件
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return &config, nil
}