package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"port-switch-backend/internal/infrastructure/config"
)

// Redis键名常量
const (
	REDIS_KEY_PORT_STATUS = "port_status"
	REDIS_KEY_LAST_UPDATE = "last_update"
)

// RedisCacheRepository Redis缓存仓储实现
type RedisCacheRepository struct {
	client      *redis.Client
	ctx         context.Context
	cacheExpiry time.Duration
}

// NewRedisCacheRepository 创建Redis缓存仓储
func NewRedisCacheRepository(cfg *config.RedisConfig, cacheExpiry time.Duration) (*RedisCacheRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()

	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("Redis连接失败: %v", err)
	}

	log.Printf("Redis连接成功，缓存过期时间: %v", cacheExpiry)
	return &RedisCacheRepository{
		client:      client,
		ctx:         ctx,
		cacheExpiry: cacheExpiry,
	}, nil
}

// SetPortStatus 设置端口状态到缓存
func (r *RedisCacheRepository) SetPortStatus(status map[string]string) error {
	// 将状态序列化为JSON
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("序列化状态失败: %v", err)
	}

	// 存储到Redis，设置过期时间
	err = r.client.Set(r.ctx, REDIS_KEY_PORT_STATUS, statusJSON, r.cacheExpiry).Err()
	if err != nil {
		return fmt.Errorf("存储到Redis失败: %v", err)
	}

	// 更新最后更新时间，也设置相同的过期时间
	err = r.client.Set(r.ctx, REDIS_KEY_LAST_UPDATE, time.Now().Unix(), r.cacheExpiry).Err()
	if err != nil {
		return fmt.Errorf("更新时间戳失败: %v", err)
	}

	log.Printf("端口状态缓存更新成功，过期时间: %v, 数据: %+v", r.cacheExpiry, status)
	return nil
}

// GetPortStatus 从缓存获取端口状态
func (r *RedisCacheRepository) GetPortStatus() (map[string]string, error) {
	// 从Redis获取缓存的状态
	statusJSON, err := r.client.Get(r.ctx, REDIS_KEY_PORT_STATUS).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("缓存中没有端口状态数据")
		}
		return nil, fmt.Errorf("从Redis获取状态失败: %v", err)
	}

	// 反序列化状态
	var status map[string]string
	err = json.Unmarshal([]byte(statusJSON), &status)
	if err != nil {
		return nil, fmt.Errorf("反序列化状态失败: %v", err)
	}

	// 获取最后更新时间
	lastUpdateStr, err := r.client.Get(r.ctx, REDIS_KEY_LAST_UPDATE).Result()
	if err == nil {
		if lastUpdate, err := strconv.ParseInt(lastUpdateStr, 10, 64); err == nil {
			updateTime := time.Unix(lastUpdate, 0)
			log.Printf("返回缓存的端口状态，最后更新时间: %s", updateTime.Format("2006-01-02 15:04:05"))
		}
	}

	return status, nil
}

// InvalidateCache 使缓存失效
func (r *RedisCacheRepository) InvalidateCache() error {
	err := r.client.Del(r.ctx, REDIS_KEY_PORT_STATUS, REDIS_KEY_LAST_UPDATE).Err()
	if err != nil {
		return fmt.Errorf("清除缓存失败: %v", err)
	}

	log.Printf("缓存已清除")
	return nil
}

// Close 关闭Redis连接
func (r *RedisCacheRepository) Close() error {
	return r.client.Close()
}