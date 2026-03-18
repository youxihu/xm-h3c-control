package repository

import "port-switch-backend/internal/domain/entity"

// OperationLogRepository 操作日志仓储接口
type OperationLogRepository interface {
	// SaveLog 保存操作日志
	SaveLog(log *entity.OperationLog) error
	
	// GetRecentLogs 获取最近的操作日志
	GetRecentLogs(limit int) ([]*entity.OperationLog, error)
	
	// ClearOldLogs 清理旧日志（保留最近N条）
	ClearOldLogs(keepCount int) error
}