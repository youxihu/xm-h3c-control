package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"

	"port-switch-backend/internal/domain/entity"
	"port-switch-backend/internal/infrastructure/config"
)

const (
	REDIS_KEY_OPERATION_LOGS = "operation_logs"
	MAX_LOGS_COUNT          = 100 // 最多保留100条日志
	LOG_EXPIRY_HOURS        = 24 * 7 // 日志保留7天
)

// RedisOperationLogRepository Redis操作日志仓储实现
type RedisOperationLogRepository struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisOperationLogRepository 创建Redis操作日志仓储
func NewRedisOperationLogRepository(cfg *config.RedisConfig) (*RedisOperationLogRepository, error) {
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

	return &RedisOperationLogRepository{
		client: client,
		ctx:    ctx,
	}, nil
}

// SaveLog 保存操作日志
func (r *RedisOperationLogRepository) SaveLog(logEntry *entity.OperationLog) error {
	// 序列化日志，直接存储格式化的时间字符串
	logData := map[string]interface{}{
		"id":             logEntry.ID(),
		"operator_ip":    logEntry.OperatorIP(),
		"operation":      logEntry.Operation(),
		"status":         logEntry.Status(),
		"message":        logEntry.Message(),
		"source_port_ip": logEntry.SourcePortIP(),
		"target_port_ip": logEntry.TargetPortIP(),
		"timestamp":      logEntry.Timestamp().Format("2006-01-02 15:04:05"), // 直接存储年月日时分秒格式
	}

	logJSON, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("序列化日志失败: %v", err)
	}

	// 使用LPUSH追加到列表头部（最新的在前面）
	err = r.client.LPush(r.ctx, REDIS_KEY_OPERATION_LOGS, logJSON).Err()
	if err != nil {
		return fmt.Errorf("保存日志到Redis失败: %v", err)
	}

	// 设置过期时间（7天）
	err = r.client.Expire(r.ctx, REDIS_KEY_OPERATION_LOGS, LOG_EXPIRY_HOURS*time.Hour).Err()
	if err != nil {
		log.Printf("设置日志过期时间失败: %v", err)
	}

	// 限制日志数量，删除多余的旧日志
	err = r.client.LTrim(r.ctx, REDIS_KEY_OPERATION_LOGS, 0, MAX_LOGS_COUNT-1).Err()
	if err != nil {
		log.Printf("清理旧日志失败: %v", err)
	}

	log.Printf("操作日志已保存: %s - %s", logEntry.OperatorIP(), logEntry.Operation())
	return nil
}

// GetRecentLogs 获取最近的操作日志
func (r *RedisOperationLogRepository) GetRecentLogs(limit int) ([]*entity.OperationLog, error) {
	// 从Redis获取日志列表
	logStrings, err := r.client.LRange(r.ctx, REDIS_KEY_OPERATION_LOGS, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("从Redis获取日志失败: %v", err)
	}

	var logs []*entity.OperationLog
	for _, logStr := range logStrings {
		var logData map[string]interface{}
		if err := json.Unmarshal([]byte(logStr), &logData); err != nil {
			log.Printf("反序列化日志失败: %v", err)
			continue
		}

		// 解析时间字符串
		timestampStr := logData["timestamp"].(string)
		timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
		if err != nil {
			log.Printf("解析时间失败: %v", err)
			continue
		}

		// 重建操作日志实体
		// 兼容新旧数据格式
		var operatorIP string
		if val, exists := logData["operator_ip"]; exists && val != nil {
			// 新格式
			operatorIP = val.(string)
		} else if val, exists := logData["operator_external_ip"]; exists && val != nil {
			// 旧格式，使用external_ip作为主要标识
			operatorIP = val.(string)
		} else if val, exists := logData["operator_internal_ip"]; exists && val != nil {
			// 旧格式，使用internal_ip作为备用标识
			operatorIP = val.(string)
		} else {
			operatorIP = "unknown"
		}
		
		logEntry := entity.NewOperationLogWithTime(
			logData["id"].(string),
			operatorIP,
			logData["operation"].(string),
			logData["status"].(string),
			logData["message"].(string),
			logData["source_port_ip"].(string),
			logData["target_port_ip"].(string),
			timestamp,
		)
		
		logs = append(logs, logEntry)
	}

	return logs, nil
}

// ClearOldLogs 清理旧日志
func (r *RedisOperationLogRepository) ClearOldLogs(keepCount int) error {
	err := r.client.LTrim(r.ctx, REDIS_KEY_OPERATION_LOGS, 0, int64(keepCount-1)).Err()
	if err != nil {
		return fmt.Errorf("清理旧日志失败: %v", err)
	}
	
	log.Printf("已清理旧日志，保留最近 %d 条", keepCount)
	return nil
}