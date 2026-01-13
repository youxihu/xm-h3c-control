package notification

import (
	"log"
	"strings"
	"h3c-nat-manager/internal/domain/notification"
	"h3c-nat-manager/internal/infrastructure/config"
	"github.com/youxihu/dingtalk/dingtalk"
)

// DingTalkService 钉钉通知服务
type DingTalkService struct {
	config *config.DingTalkConfig
}

// NewDingTalkService 创建钉钉通知服务
func NewDingTalkService(dingTalkConfig *config.DingTalkConfig) *DingTalkService {
	return &DingTalkService{
		config: dingTalkConfig,
	}
}

// SendNotification 发送过期通知
func (d *DingTalkService) SendNotification(notify *notification.ExpiryNotification) error {
	// 根据本地IP地址确定服务器IP
	serverIP := d.extractServerIP(notify.LocalAddress)
	
	// 选择对应的钉钉群组配置
	groupConfig := d.selectGroupConfig(serverIP)
	
	title := "[通知] 端口映射即将过期"
	message := notify.FormatMessage()
	
	log.Printf("发送过期通知 - 群组: %s, 服务器: %s, 外网地址: %s", 
		groupConfig.Name, serverIP, notify.GlobalAddress)
	
	return dingtalk.SendDingDingNotification(
		groupConfig.Webhook,
		groupConfig.Secret,
		title,
		message,
		nil,   // atMobiles
		false, // isAtAll
	)
}

// SendDeletionNotification 发送删除通知
func (d *DingTalkService) SendDeletionNotification(notify *notification.DeletionNotification) error {
	// 根据本地IP地址确定服务器IP
	serverIP := d.extractServerIP(notify.LocalAddress)
	
	// 选择对应的钉钉群组配置
	groupConfig := d.selectGroupConfig(serverIP)
	
	title := "[通知] 端口映射条目删除"
	message := notify.FormatMessage()
	
	log.Printf("发送删除通知 - 群组: %s, 服务器: %s, 外网地址: %s", 
		groupConfig.Name, serverIP, notify.GlobalAddress)
	
	return dingtalk.SendDingDingNotification(
		groupConfig.Webhook,
		groupConfig.Secret,
		title,
		message,
		nil,   // atMobiles
		false, // isAtAll
	)
}

// extractServerIP 从本地地址中提取服务器IP
func (d *DingTalkService) extractServerIP(localAddress string) string {
	// 本地地址格式通常是 "192.168.1.112/8080" 或 "192.168.1.112:22"
	// 先处理冒号分隔的情况
	if idx := strings.Index(localAddress, ":"); idx != -1 {
		return localAddress[:idx]
	}
	// 再处理斜杠分隔的情况
	if idx := strings.Index(localAddress, "/"); idx != -1 {
		return localAddress[:idx]
	}
	// 如果没有端口，直接返回IP
	return localAddress
}

// selectGroupConfig 根据服务器IP选择对应的群组配置
func (d *DingTalkService) selectGroupConfig(serverIP string) config.DingTalkGroupConfig {
	// 遍历所有群组，查找包含该服务器IP的群组
	for groupName, groupConfig := range d.config.Groups {
		for _, ip := range groupConfig.Servers {
			if ip == serverIP {
				log.Printf("服务器匹配到群组 - 服务器: %s, 群组: %s", serverIP, groupName)
				return groupConfig
			}
		}
	}
	
	// 如果没有找到匹配的群组，使用默认配置
	log.Printf("服务器未找到匹配群组，使用默认群组 - 服务器: %s", serverIP)
	return d.config.Default
}