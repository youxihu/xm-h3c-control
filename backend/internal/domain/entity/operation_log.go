package entity

import (
	"fmt"
	"time"
)

// OperationLog 操作日志实体
type OperationLog struct {
	id           string
	operatorIP   string // 操作者IP
	operation    string // 操作类型
	status       string // 状态
	message      string // 消息
	sourcePortIP string // 源端口IP (格式: 端口_IP)
	targetPortIP string // 目标端口IP (格式: 端口_IP)
	timestamp    time.Time
}

// NewOperationLog 创建操作日志实体
func NewOperationLog(operatorIP, operation, status, message, sourcePortIP, targetPortIP string) *OperationLog {
	return &OperationLog{
		id:           fmt.Sprintf("%d", time.Now().UnixNano()),
		operatorIP:   operatorIP,
		operation:    operation,
		status:       status,
		message:      message,
		sourcePortIP: sourcePortIP,
		targetPortIP: targetPortIP,
		timestamp:    time.Now(),
	}
}

// NewOperationLogWithTime 创建带指定时间的操作日志实体
func NewOperationLogWithTime(id, operatorIP, operation, status, message, sourcePortIP, targetPortIP string, timestamp time.Time) *OperationLog {
	return &OperationLog{
		id:           id,
		operatorIP:   operatorIP,
		operation:    operation,
		status:       status,
		message:      message,
		sourcePortIP: sourcePortIP,
		targetPortIP: targetPortIP,
		timestamp:    timestamp,
	}
}

// Getters
func (o *OperationLog) ID() string           { return o.id }
func (o *OperationLog) OperatorIP() string   { return o.operatorIP }
func (o *OperationLog) Operation() string    { return o.operation }
func (o *OperationLog) Status() string       { return o.status }
func (o *OperationLog) Message() string      { return o.message }
func (o *OperationLog) SourcePortIP() string { return o.sourcePortIP }
func (o *OperationLog) TargetPortIP() string { return o.targetPortIP }
func (o *OperationLog) Timestamp() time.Time { return o.timestamp }
