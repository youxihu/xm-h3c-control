package entity

import (
	"fmt"
)

// NATMapping NAT映射实体
type NATMapping struct {
	protocol     string
	externalIP   string
	externalPort int
	internalIP   string
	internalPort int
	description  string
}

// NewNATMapping 创建NAT映射实体
func NewNATMapping(protocol, externalIP string, externalPort int, internalIP string, internalPort int, description string) (*NATMapping, error) {
	if protocol == "" {
		return nil, fmt.Errorf("协议不能为空")
	}
	if externalIP == "" {
		return nil, fmt.Errorf("外网IP不能为空")
	}
	if externalPort <= 0 || externalPort > 65535 {
		return nil, fmt.Errorf("外网端口必须在1-65535范围内")
	}
	if internalIP == "" {
		return nil, fmt.Errorf("内网IP不能为空")
	}
	if internalPort <= 0 || internalPort > 65535 {
		return nil, fmt.Errorf("内网端口必须在1-65535范围内")
	}

	return &NATMapping{
		protocol:     protocol,
		externalIP:   externalIP,
		externalPort: externalPort,
		internalIP:   internalIP,
		internalPort: internalPort,
		description:  description,
	}, nil
}

// Getters
func (n *NATMapping) Protocol() string    { return n.protocol }
func (n *NATMapping) ExternalIP() string  { return n.externalIP }
func (n *NATMapping) ExternalPort() int   { return n.externalPort }
func (n *NATMapping) InternalIP() string  { return n.internalIP }
func (n *NATMapping) InternalPort() int   { return n.internalPort }
func (n *NATMapping) Description() string { return n.description }

// IsSameTarget 检查是否指向相同的内网目标
func (n *NATMapping) IsSameTarget(internalIP string, internalPort int) bool {
	return n.internalIP == internalIP && n.internalPort == internalPort
}

// String 字符串表示
func (n *NATMapping) String() string {
	return fmt.Sprintf("%s:%d -> %s:%d (%s)",
		n.externalIP, n.externalPort,
		n.internalIP, n.internalPort,
		n.protocol)
}
