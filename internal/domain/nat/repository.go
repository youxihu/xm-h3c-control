package nat

// Repository NAT仓储接口
type Repository interface {
	// GetAllEntries 获取所有NAT映射条目
	GetAllEntries() ([]*NATEntry, error)
	
	// DeleteEntry 删除指定的NAT映射条目
	DeleteEntry(entry *NATEntry) error
}