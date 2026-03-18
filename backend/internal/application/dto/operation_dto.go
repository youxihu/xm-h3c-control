package dto

// OperationLogDTO 操作日志DTO
type OperationLogDTO struct {
	ID           string `json:"id"`
	OperatorIP   string `json:"operator_ip"`
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	SourcePortIP string `json:"source_port_ip"`
	TargetPortIP string `json:"target_port_ip"`
	Timestamp    string `json:"timestamp"`
}

// OperationLogsResponse 操作日志响应DTO
type OperationLogsResponse struct {
	Logs []OperationLogDTO `json:"logs"`
}