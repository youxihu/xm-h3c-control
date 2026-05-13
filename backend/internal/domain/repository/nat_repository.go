package repository

import "port-switch-backend/internal/domain/entity"

// NATRepository NAT映射仓储接口
type NATRepository interface {
	// GetAllMappings 获取所有NAT映射
	GetAllMappings() ([]*entity.NATMapping, error)

	// CreateMapping 创建新映射
	CreateMapping(mapping *entity.NATMapping) error

	// DeleteMapping 删除映射
	DeleteMapping(mapping *entity.NATMapping) error

	// BatchSwitchMappings 批量切换映射（在单个SSH会话中完成）
	BatchSwitchMappings(operations []SwitchOperation) error
}

// SwitchOperation 切换操作
type SwitchOperation struct {
	OldMapping *entity.NATMapping
	NewMapping *entity.NATMapping
}
