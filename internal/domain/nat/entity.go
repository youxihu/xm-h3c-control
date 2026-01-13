package nat

import (
	"strconv"
	"strings"
	"time"
)

// NATEntry NAT映射条目实体
type NATEntry struct {
	Interface   string     // 接口名称 如: GigabitEthernet0/0
	Protocol    string     // 协议类型 TCP/UDP
	GlobalIP    string     // 外网IP
	GlobalPort  int        // 外网端口
	LocalIP     string     // 内网IP
	LocalPort   int        // 内网端口
	Description string     // 原始描述
	Status      string     // 配置状态 Active/Inactive
	ExpiryDate  *time.Time // 过期时间
}

// HasExpiryInfo 检查是否包含过期信息
func (n *NATEntry) HasExpiryInfo() bool {
	return strings.Contains(n.Description, "vp=")
}

// ParseExpiryDate 从描述中解析过期时间
func (n *NATEntry) ParseExpiryDate() error {
	return n.ParseExpiryDateWithTime(21, 30) // 默认21:30
}

// ParseExpiryDateWithTime 从描述中解析过期时间，使用指定的小时和分钟
func (n *NATEntry) ParseExpiryDateWithTime(hour, minute int) error {
	if !n.HasExpiryInfo() {
		return nil
	}

	// 查找vp=YYMMDD格式
	parts := strings.Split(n.Description, "vp=")
	if len(parts) < 2 {
		return nil
	}

	dateStr := strings.TrimSpace(parts[1])
	if len(dateStr) < 6 {
		return nil
	}

	// 取前6位作为日期
	dateStr = dateStr[:6]
	
	// 解析YYMMDD格式
	year, err := strconv.Atoi("20" + dateStr[:2])
	if err != nil {
		return err
	}
	
	month, err := strconv.Atoi(dateStr[2:4])
	if err != nil {
		return err
	}
	
	day, err := strconv.Atoi(dateStr[4:6])
	if err != nil {
		return err
	}

	// 使用配置的过期时间
	expiryDate := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
	n.ExpiryDate = &expiryDate
	
	return nil
}

// IsExpired 检查是否已过期
func (n *NATEntry) IsExpired() bool {
	if n.ExpiryDate == nil {
		return false
	}
	
	// 直接比较当前时间和过期时间
	return time.Now().After(*n.ExpiryDate)
}

// WillExpireIn 检查是否在指定天数内过期（不包括已过期的）
func (n *NATEntry) WillExpireIn(days int) bool {
	if n.ExpiryDate == nil {
		return false
	}
	
	// 如果已经过期，不需要通知
	if n.IsExpired() {
		return false
	}
	
	// 检查是否在指定天数内过期
	now := time.Now()
	checkDate := now.AddDate(0, 0, days)
	
	return n.ExpiryDate.Before(checkDate) || n.ExpiryDate.Equal(checkDate)
}

// GetGlobalAddress 获取外网地址端口组合
func (n *NATEntry) GetGlobalAddress() string {
	return n.GlobalIP + ":" + strconv.Itoa(n.GlobalPort)
}

// GetLocalAddress 获取内网地址端口组合
func (n *NATEntry) GetLocalAddress() string {
	return n.LocalIP + ":" + strconv.Itoa(n.LocalPort)
}