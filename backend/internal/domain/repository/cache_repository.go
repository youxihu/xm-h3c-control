package repository

// CacheRepository 缓存仓储接口
type CacheRepository interface {
	// SetPortStatus 设置端口状态缓存
	SetPortStatus(status map[string]string) error
	
	// GetPortStatus 获取端口状态缓存
	GetPortStatus() (map[string]string, error)
	
	// InvalidateCache 使缓存失效
	InvalidateCache() error
}