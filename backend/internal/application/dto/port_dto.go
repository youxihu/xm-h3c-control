package dto

// SwitchPortRequest 切换端口请求DTO
type SwitchPortRequest struct {
	CurrentInternalIP string `json:"current_internal_ip" binding:"required"`
	NewInternalIP     string `json:"new_internal_ip" binding:"required"`
	InternalPort      int    `json:"internal_port" binding:"required"`
}

// ApplyConfigRequest 批量配置应用请求DTO
type ApplyConfigRequest struct {
	Configs []PortConfigDTO `json:"configs" binding:"required"`
}

// PortConfigDTO 端口配置DTO
type PortConfigDTO struct {
	InternalPort int    `json:"internal_port" binding:"required"`
	InternalIP   string `json:"internal_ip" binding:"required"`
}

// PortConfigResponse 端口配置响应DTO
type PortConfigResponse struct {
	Ports []PortInfoDTO `json:"ports"`
}

// PortInfoDTO 端口信息DTO
type PortInfoDTO struct {
	InternalPort int           `json:"internal_port"`
	ExternalPort int           `json:"external_port"`
	ExternalIP   string        `json:"external_ip"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Options      []IPOptionDTO `json:"options"`
}

// IPOptionDTO IP选项DTO
type IPOptionDTO struct {
	IP          string `json:"ip"`
	Environment string `json:"environment"`
	Description string `json:"description"`
}