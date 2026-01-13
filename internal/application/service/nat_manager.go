package service

import (
	"log"
	"time"

	"h3c-nat-manager/internal/domain/nat"
	"h3c-nat-manager/internal/domain/notification"
	"h3c-nat-manager/internal/infrastructure/config"
	"h3c-nat-manager/internal/infrastructure/description"
)

// NATManagerService NAT管理应用服务
type NATManagerService struct {
	natRepo         nat.Repository
	notificationSvc notification.Service
	descMapper      *description.Mapper
	config          *config.Config
}

// NewNATManagerService 创建NAT管理服务
func NewNATManagerService(
	natRepo nat.Repository,
	notificationSvc notification.Service,
	descMapper *description.Mapper,
	cfg *config.Config,
) *NATManagerService {
	return &NATManagerService{
		natRepo:         natRepo,
		notificationSvc: notificationSvc,
		descMapper:      descMapper,
		config:          cfg,
	}
}

// CheckAndNotify 检查并发送过期通知
func (s *NATManagerService) CheckAndNotify() error {
	log.Println("开始检查即将过期的NAT映射条目")

	entries, err := s.natRepo.GetAllEntries()
	if err != nil {
		log.Printf("获取NAT条目失败: %v", err)
		return err
	}

	reminderDays := s.config.Router.ReminderBeforeExpiration
	notifyCount := 0

	log.Printf("获取NAT条目成功，总条目数: %d，提醒天数: %d", len(entries), reminderDays)

	for _, entry := range entries {
		// 只处理有过期信息的条目
		if !entry.HasExpiryInfo() {
			continue
		}

		// 检查是否即将过期
		if entry.WillExpireIn(reminderDays) {
			if err := s.sendExpiryNotification(entry); err != nil {
				log.Printf("发送通知失败 - 外网地址: %s, 错误: %v", entry.GetGlobalAddress(), err)
				continue
			}
			notifyCount++
			log.Printf("已发送过期提醒 - 外网地址: %s, 内网地址: %s, 过期时间: %s", 
				entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.ExpiryDate.Format(time.DateTime))
		}
	}

	log.Printf("检查完成，发送通知数量: %d", notifyCount)
	return nil
}

// CleanupExpired 清理已过期的条目
func (s *NATManagerService) CleanupExpired() error {
	log.Println("开始清理已过期的NAT映射条目")

	entries, err := s.natRepo.GetAllEntries()
	if err != nil {
		log.Printf("获取NAT条目失败: %v", err)
		return err
	}

	cleanupCount := 0

	log.Printf("获取NAT条目成功，总条目数: %d", len(entries))

	for _, entry := range entries {
		// 只处理有过期信息且已过期的条目
		if !entry.HasExpiryInfo() || !entry.IsExpired() {
			continue
		}

		// 先发送删除通知
		if err := s.sendDeletionNotification(entry); err != nil {
			log.Printf("发送删除通知失败 - 外网地址: %s, 错误: %v", entry.GetGlobalAddress(), err)
		}

		// 删除条目
		if err := s.natRepo.DeleteEntry(entry); err != nil {
			log.Printf("删除过期条目失败 - 外网地址: %s, 内网地址: %s, 协议: %s, 过期时间: %s, 错误: %v",
				entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.Protocol, 
				entry.ExpiryDate.Format(time.DateTime), err)
			continue
		}

		cleanupCount++
		log.Printf("已删除过期条目 - 外网地址: %s, 内网地址: %s, 协议: %s, 过期时间: %s",
			entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.Protocol, 
			entry.ExpiryDate.Format(time.DateTime))
	}

	log.Printf("清理完成，删除条目数量: %d", cleanupCount)
	return nil
}

// SmartProcess 智能处理模式：自动决定通知或删除
func (s *NATManagerService) SmartProcess() error {
	log.Println("开始智能处理NAT映射条目")

	entries, err := s.natRepo.GetAllEntries()
	if err != nil {
		log.Printf("获取NAT条目失败: %v", err)
		return err
	}

	reminderDays := s.config.Router.ReminderBeforeExpiration
	notifyCount := 0
	cleanupCount := 0

	log.Printf("获取NAT条目成功，总条目数: %d，提醒天数: %d", len(entries), reminderDays)

	for _, entry := range entries {
		// 只处理有过期信息的条目
		if !entry.HasExpiryInfo() {
			continue
		}

		// 如果已经过期，先发送删除通知，然后删除
		if entry.IsExpired() {
			// 发送删除通知
			if err := s.sendDeletionNotification(entry); err != nil {
				log.Printf("发送删除通知失败 - 外网地址: %s, 错误: %v", entry.GetGlobalAddress(), err)
			}

			// 删除条目
			if err := s.natRepo.DeleteEntry(entry); err != nil {
				log.Printf("删除过期条目失败 - 外网地址: %s, 内网地址: %s, 协议: %s, 过期时间: %s, 错误: %v",
					entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.Protocol, 
					entry.ExpiryDate.Format(time.DateTime), err)
				continue
			}
			cleanupCount++
			log.Printf("已删除过期条目 - 外网地址: %s, 内网地址: %s, 协议: %s, 过期时间: %s",
				entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.Protocol, 
				entry.ExpiryDate.Format(time.DateTime))
		} else if entry.WillExpireIn(reminderDays) {
			// 如果即将过期，发送通知
			if err := s.sendExpiryNotification(entry); err != nil {
				log.Printf("发送通知失败 - 外网地址: %s, 错误: %v", entry.GetGlobalAddress(), err)
				continue
			}
			notifyCount++
			log.Printf("已发送过期提醒 - 外网地址: %s, 内网地址: %s, 过期时间: %s",
				entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.ExpiryDate.Format(time.DateTime))
		}
	}

	log.Printf("智能处理完成，发送通知数量: %d，删除条目数量: %d", notifyCount, cleanupCount)
	return nil
}

// sendExpiryNotification 发送过期通知
func (s *NATManagerService) sendExpiryNotification(entry *nat.NATEntry) error {
	// 获取正确的中文描述
	description := s.descMapper.GetDescription(entry.GetGlobalAddress())

	notify := &notification.ExpiryNotification{
		GlobalAddress: entry.GetGlobalAddress(),
		LocalAddress:  entry.GetLocalAddress(),
		Protocol:      entry.Protocol,
		Description:   description,
		ExpiryDate:    *entry.ExpiryDate,
		NotifyTime:    time.Now(),
	}

	return s.notificationSvc.SendNotification(notify)
}

// sendDeletionNotification 发送删除通知
func (s *NATManagerService) sendDeletionNotification(entry *nat.NATEntry) error {
	// 获取正确的中文描述
	description := s.descMapper.GetDescription(entry.GetGlobalAddress())

	notify := &notification.DeletionNotification{
		GlobalAddress: entry.GetGlobalAddress(),
		LocalAddress:  entry.GetLocalAddress(),
		Protocol:      entry.Protocol,
		Description:   description,
		ExpiryDate:    *entry.ExpiryDate,
		DeleteTime:    time.Now(),
	}

	return s.notificationSvc.SendDeletionNotification(notify)
}