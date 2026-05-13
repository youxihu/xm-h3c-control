package entity

import "fmt"

// PortConfig 端口配置实体
type PortConfig struct {
	internalPort int
	externalPort int
	externalIP   string
	name         string
	description  string
	options      []IPOption
}

// IPOption IP选项值对象
type IPOption struct {
	IP           string
	InternalPort int
	Environment  string
	Description  string
}

// NewPortConfig 创建端口配置实体
func NewPortConfig(internalPort, externalPort int, externalIP, name, description string) (*PortConfig, error) {
	if internalPort <= 0 || internalPort > 65535 {
		return nil, fmt.Errorf("内网端口必须在1-65535范围内")
	}
	if externalPort <= 0 || externalPort > 65535 {
		return nil, fmt.Errorf("外网端口必须在1-65535范围内")
	}
	if externalIP == "" {
		return nil, fmt.Errorf("外网IP不能为空")
	}
	if name == "" {
		return nil, fmt.Errorf("端口名称不能为空")
	}

	return &PortConfig{
		internalPort: internalPort,
		externalPort: externalPort,
		externalIP:   externalIP,
		name:         name,
		description:  description,
		options:      make([]IPOption, 0),
	}, nil
}

// Getters
func (p *PortConfig) InternalPort() int   { return p.internalPort }
func (p *PortConfig) ExternalPort() int   { return p.externalPort }
func (p *PortConfig) ExternalIP() string  { return p.externalIP }
func (p *PortConfig) Name() string        { return p.name }
func (p *PortConfig) Description() string { return p.description }
func (p *PortConfig) Options() []IPOption { return p.options }

// AddOption 添加IP选项
func (p *PortConfig) AddOption(ip string, internalPort int, env, desc string) error {
	if ip == "" {
		return fmt.Errorf("IP地址不能为空")
	}
	if env == "" {
		return fmt.Errorf("环境标识不能为空")
	}

	option := IPOption{
		IP:           ip,
		InternalPort: internalPort,
		Environment:  env,
		Description:  desc,
	}

	p.options = append(p.options, option)
	return nil
}
