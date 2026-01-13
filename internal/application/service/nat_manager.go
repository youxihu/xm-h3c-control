package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"h3c-nat-manager/internal/domain/nat"
	"h3c-nat-manager/internal/domain/notification"
	"h3c-nat-manager/internal/infrastructure/config"
	"h3c-nat-manager/internal/infrastructure/description"
)

const (
	// 操作类型常量
	OperationNotify  = "notify"
	OperationCleanup = "cleanup"
	OperationSmart   = "smart"
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
	return s.processEntries(OperationNotify)
}

// CleanupExpired 清理已过期的条目
func (s *NATManagerService) CleanupExpired() error {
	return s.processEntries(OperationCleanup)
}

// SmartProcess 智能处理模式：自动决定通知或删除
func (s *NATManagerService) SmartProcess() error {
	return s.processEntries(OperationSmart)
}

// processEntries 统一的条目处理方法
func (s *NATManagerService) processEntries(operation string) error {
	log.Printf("开始执行%s操作", s.getOperationName(operation))

	entries, err := s.natRepo.GetAllEntries()
	if err != nil {
		return fmt.Errorf("获取NAT条目失败: %v", err)
	}

	reminderDays := s.config.Router.ReminderBeforeExpiration
	log.Printf("获取NAT条目成功，总条目数: %d，提前 %d 天 提醒", len(entries), reminderDays)

	// 添加调试信息：显示有过期信息的条目
	expiredCount := 0
	willExpireCount := 0
	noExpiryCount := 0
	
	for _, entry := range entries {
		if !entry.HasExpiryInfo() {
			noExpiryCount++
			continue
		}
		
		if entry.IsExpired() {
			expiredCount++
			log.Printf("发现已过期条目: %s -> %s, 过期时间: %s", 
				entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.ExpiryDate.Format(time.DateTime))
		} else if entry.WillExpireIn(reminderDays) {
			willExpireCount++
			log.Printf("发现即将过期条目: %s -> %s, 过期时间: %s", 
				entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.ExpiryDate.Format(time.DateTime))
		}
	}
	
	log.Printf("条目统计 - 无过期信息: %d, 即将过期: %d, 已过期: %d", 
		noExpiryCount, willExpireCount, expiredCount)

	// 使用并发处理提高效率
	results := s.processEntriesConcurrently(entries, operation, reminderDays)
	
	log.Printf("%s完成，发送通知数量: %d，删除条目数量: %d", 
		s.getOperationName(operation), results.NotifyCount, results.CleanupCount)
	
	return nil
}

// ProcessResult 处理结果
type ProcessResult struct {
	NotifyCount  int
	CleanupCount int
	Errors       []error
}

// processEntriesConcurrently 并发处理条目
func (s *NATManagerService) processEntriesConcurrently(entries []*nat.NATEntry, operation string, reminderDays int) *ProcessResult {
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := &ProcessResult{}

	// 限制并发数量，避免过多连接
	semaphore := make(chan struct{}, 5)

	for _, entry := range entries {
		if !entry.HasExpiryInfo() {
			continue
		}

		wg.Add(1)
		go func(e *nat.NATEntry) {
			defer wg.Done()
			semaphore <- struct{}{} // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			switch operation {
			case OperationNotify:
				if e.WillExpireIn(reminderDays) {
					if err := s.sendExpiryNotification(e); err != nil {
						mu.Lock()
						result.Errors = append(result.Errors, fmt.Errorf("发送通知失败 - %s: %v", e.GetGlobalAddress(), err))
						mu.Unlock()
						return
					}
					mu.Lock()
					result.NotifyCount++
					mu.Unlock()
					log.Printf("已发送过期提醒 - %s -> %s, 过期时间: %s", 
						e.GetGlobalAddress(), e.GetLocalAddress(), e.ExpiryDate.Format(time.DateTime))
				}

			case OperationCleanup:
				if e.IsExpired() {
					if err := s.deleteAndNotify(e); err != nil {
						mu.Lock()
						result.Errors = append(result.Errors, err)
						mu.Unlock()
						return
					}
					mu.Lock()
					result.CleanupCount++
					mu.Unlock()
				}

			case OperationSmart:
				if e.IsExpired() {
					if err := s.deleteAndNotify(e); err != nil {
						mu.Lock()
						result.Errors = append(result.Errors, err)
						mu.Unlock()
						return
					}
					mu.Lock()
					result.CleanupCount++
					mu.Unlock()
				} else if e.WillExpireIn(reminderDays) {
					if err := s.sendExpiryNotification(e); err != nil {
						mu.Lock()
						result.Errors = append(result.Errors, fmt.Errorf("发送通知失败 - %s: %v", e.GetGlobalAddress(), err))
						mu.Unlock()
						return
					}
					mu.Lock()
					result.NotifyCount++
					mu.Unlock()
					log.Printf("已发送过期提醒 - %s -> %s, 过期时间: %s",
						e.GetGlobalAddress(), e.GetLocalAddress(), e.ExpiryDate.Format(time.DateTime))
				}
			}
		}(entry)
	}

	wg.Wait()

	// 记录错误
	for _, err := range result.Errors {
		log.Printf("处理错误: %v", err)
	}

	return result
}

// deleteAndNotify 删除条目并发送通知
func (s *NATManagerService) deleteAndNotify(entry *nat.NATEntry) error {
	// 删除条目
	if err := s.natRepo.DeleteEntry(entry); err != nil {
		return fmt.Errorf("删除过期条目失败 - %s -> %s (%s), 过期时间: %s, 错误: %v",
			entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.Protocol, 
			entry.ExpiryDate.Format(time.DateTime), err)
	}

	// 删除成功后发送删除通知
	if err := s.sendDeletionNotification(entry); err != nil {
		log.Printf("发送删除通知失败 - %s: %v", entry.GetGlobalAddress(), err)
	}

	log.Printf("已删除过期条目 - %s -> %s (%s), 过期时间: %s",
		entry.GetGlobalAddress(), entry.GetLocalAddress(), entry.Protocol, 
		entry.ExpiryDate.Format(time.DateTime))
	
	return nil
}

// getOperationName 获取操作名称
func (s *NATManagerService) getOperationName(operation string) string {
	switch operation {
	case OperationNotify:
		return "通知检查"
	case OperationCleanup:
		return "过期清理"
	case OperationSmart:
		return "智能处理"
	default:
		return "未知操作"
	}
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