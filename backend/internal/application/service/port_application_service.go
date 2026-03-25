package service

import (
	"fmt"
	"log"

	"port-switch-backend/internal/application/dto"
	"port-switch-backend/internal/domain/entity"
	"port-switch-backend/internal/domain/repository"
	"port-switch-backend/internal/domain/service"
)

// PortApplicationService 端口应用服务
type PortApplicationService struct {
	natService       *service.NATService
	portMappings     map[int]int
	portDescriptions map[int]PortDescription
	hosts            map[string]HostConfig
	externalIP       string
	logRepo          repository.OperationLogRepository
}

// PortDescription 端口描述配置
type PortDescription struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// HostConfig 主机配置
type HostConfig struct {
	Env      string         `yaml:"env"`
	Services map[string]int `yaml:"services"`
}

// NewPortApplicationService 创建端口应用服务
func NewPortApplicationService(
	natService *service.NATService,
	portMappings map[int]int,
	portDescriptions map[int]PortDescription,
	hosts map[string]HostConfig,
	externalIP string,
	logRepo repository.OperationLogRepository,
) *PortApplicationService {
	return &PortApplicationService{
		natService:       natService,
		portMappings:     portMappings,
		portDescriptions: portDescriptions,
		hosts:            hosts,
		externalIP:       externalIP,
		logRepo:          logRepo,
	}
}

// SwitchPort 切换单个端口
func (s *PortApplicationService) SwitchPort(req dto.SwitchPortRequest) error {
	// 1. 校验目标IP和端口配置是否合法
	if err := s.validatePortConfig(req.InternalPort, req.NewInternalIP); err != nil {
		return fmt.Errorf("目标配置校验失败: %v", err)
	}

	// 2. 校验当前IP和端口配置是否合法（如果提供了当前IP）
	if req.CurrentInternalIP != "" {
		if err := s.validatePortConfig(req.InternalPort, req.CurrentInternalIP); err != nil {
			return fmt.Errorf("当前配置校验失败: %v", err)
		}
	}

	return s.natService.SwitchPortMapping(
		req.CurrentInternalIP,
		req.NewInternalIP,
		req.InternalPort,
		s.portMappings,
		s.externalIP,
	)
}

// ApplyBatchConfig 批量应用配置
func (s *PortApplicationService) ApplyBatchConfig(req dto.ApplyConfigRequest) ([]map[string]interface{}, error) {
	return s.ApplyBatchConfigWithLog(req, "")
}

// ApplyBatchConfigWithLog 批量应用配置并记录日志
// ApplyBatchConfigWithLog 批量应用配置并记录日志
func (s *PortApplicationService) ApplyBatchConfigWithLog(req dto.ApplyConfigRequest, operatorIP string) ([]map[string]interface{}, error) {
	log.Printf("开始批量应用配置，共 %d 个端口", len(req.Configs))

	// 直接从Redis缓存获取当前状态，不查询路由器
	currentStatus, err := s.natService.GetCachedPortStatus()
	if err != nil {
		log.Printf("获取缓存状态失败，使用空状态继续: %v", err)
		currentStatus = make(map[string]string)
	}

	var results []map[string]interface{}
	var switchConfigs []service.SwitchConfig
	var hasError bool

	// 第一阶段：校验所有配置并准备批量操作
	for _, config := range req.Configs {
		result := map[string]interface{}{
			"internal_port": config.InternalPort,
			"target_ip":     config.InternalIP,
		}

		// 先校验目标配置是否合法
		if err := s.validatePortConfig(config.InternalPort, config.InternalIP); err != nil {
			log.Printf("端口 %d 配置校验失败: %v", config.InternalPort, err)
			result["status"] = "error"
			result["message"] = fmt.Sprintf("配置校验失败: %v", err)
			results = append(results, result)
			hasError = true

			// 记录校验失败日志
			if operatorIP != "" {
				sourcePortIP := fmt.Sprintf("%d_unknown", config.InternalPort)
				targetPortIP := fmt.Sprintf("%d_%s", config.InternalPort, config.InternalIP)
				s.LogOperation(operatorIP, "端口切换", "失败", fmt.Sprintf("配置校验失败: %v", err), sourcePortIP, targetPortIP)
			}
			continue
		}

		// 从缓存获取当前映射状态
		currentIP := ""
		if externalPort, exists := s.portMappings[config.InternalPort]; exists {
			portKey := fmt.Sprintf("port_%d", externalPort)
			if val, exists := currentStatus[portKey]; exists {
				currentIP = val
			}
		}

		// 如果目标IP与当前IP相同，跳过
		if currentIP == config.InternalIP {
			result["status"] = "unchanged"
			result["message"] = "已经是目标IP，无需切换"
			results = append(results, result)
			continue
		}

		// 添加到批量切换配置
		switchConfigs = append(switchConfigs, service.SwitchConfig{
			InternalPort:      config.InternalPort,
			CurrentInternalIP: currentIP,
			NewInternalIP:     config.InternalIP,
		})

		result["status"] = "pending"
		result["message"] = "准备切换"
		results = append(results, result)
	}

	// 第二阶段：如果有需要切换的端口，执行批量切换
	if len(switchConfigs) > 0 && !hasError {
		log.Printf("执行批量切换，共 %d 个端口", len(switchConfigs))

		if err := s.natService.BatchSwitchPortMappings(switchConfigs, s.portMappings, s.externalIP); err != nil {
			log.Printf("批量切换失败: %v", err)

			// 更新所有待切换端口的状态为失败
			for i, result := range results {
				if result["status"] == "pending" {
					results[i]["status"] = "error"
					results[i]["message"] = fmt.Sprintf("批量切换失败: %v", err)

					// 记录失败日志
					if operatorIP != "" {
						config := req.Configs[i]
						sourcePortIP := fmt.Sprintf("%d_%s", config.InternalPort, switchConfigs[i].CurrentInternalIP)
						targetPortIP := fmt.Sprintf("%d_%s", config.InternalPort, config.InternalIP)
						s.LogOperation(operatorIP, "端口切换", "失败", err.Error(), sourcePortIP, targetPortIP)
					}
				}
			}

			return results, fmt.Errorf("批量切换失败: %v", err)
		}

		// 批量切换成功，更新所有待切换端口的状态
		switchIndex := 0
		for i, result := range results {
			if result["status"] == "pending" {
				results[i]["status"] = "success"
				results[i]["message"] = "切换成功"

				// 记录成功日志
				if operatorIP != "" {
					config := req.Configs[i]
					sourcePortIP := fmt.Sprintf("%d_%s", config.InternalPort, switchConfigs[switchIndex].CurrentInternalIP)
					targetPortIP := fmt.Sprintf("%d_%s", config.InternalPort, config.InternalIP)
					s.LogOperation(operatorIP, "端口切换", "成功", "端口切换成功", sourcePortIP, targetPortIP)
				}
				switchIndex++
			}
		}
	}

	if hasError {
		return results, fmt.Errorf("部分配置应用失败")
	}

	return results, nil
}

// GetPortConfig 获取端口配置
func (s *PortApplicationService) GetPortConfig() dto.PortConfigResponse {
	// 定义固定的端口顺序
	portOrder := []int{61002, 61100, 62201, 48080}

	var ports []dto.PortInfoDTO

	for _, internalPort := range portOrder {
		externalPort, exists := s.portMappings[internalPort]
		if !exists {
			log.Printf("警告: 配置文件中未找到内网端口 %d 的映射", internalPort)
			continue
		}

		// 从配置文件获取端口描述
		portDesc, exists := s.portDescriptions[internalPort]
		if !exists {
			log.Printf("警告: 配置文件中未找到内网端口 %d 的描述", internalPort)
			continue
		}

		// 创建端口配置实体
		portConfig, err := entity.NewPortConfig(
			internalPort,
			externalPort,
			s.externalIP,
			portDesc.Name,
			portDesc.Description,
		)
		if err != nil {
			log.Printf("创建端口配置失败: %v", err)
			continue
		}

		// 添加IP选项
		s.addIPOptions(portConfig, internalPort, portDesc.Name)

		// 转换为DTO
		portDTO := s.convertToPortInfoDTO(portConfig)
		ports = append(ports, portDTO)
	}

	return dto.PortConfigResponse{Ports: ports}
}

// GetPortStatus 获取端口状态
func (s *PortApplicationService) GetPortStatus() (map[string]string, error) {
	return s.natService.GetCurrentPortStatus(s.portMappings)
}

// UpdateCache 更新缓存
func (s *PortApplicationService) UpdateCache() error {
	return s.natService.UpdateCache(s.portMappings)
}

// UpdateCacheAfterSwitch 切换后更新缓存
func (s *PortApplicationService) UpdateCacheAfterSwitch(internalPort int, newInternalIP string) error {
	return s.natService.UpdateCacheAfterSwitch(internalPort, newInternalIP, s.portMappings)
}

// addIPOptions 添加IP选项到端口配置
func (s *PortApplicationService) addIPOptions(portConfig *entity.PortConfig, internalPort int, serviceName string) {
	// 按环境顺序添加选项
	envOrder := []string{"dev", "zc-test", "dw-test"}

	for _, env := range envOrder {
		for hostIP, hostConfig := range s.hosts {
			if hostConfig.Env == env {
				for name, port := range hostConfig.Services {
					if port == internalPort && name == serviceName {
						var envDesc string
						switch env {
						case "dev":
							envDesc = "开发环境"
						case "zc-test":
							envDesc = "测试环境"
						case "dw-test":
							envDesc = "东吴环境"
						default:
							envDesc = env
						}

						portConfig.AddOption(hostIP, env, envDesc)
					}
				}
			}
		}
	}
}

// convertToPortInfoDTO 转换为DTO
func (s *PortApplicationService) convertToPortInfoDTO(portConfig *entity.PortConfig) dto.PortInfoDTO {
	var options []dto.IPOptionDTO
	for _, opt := range portConfig.Options() {
		options = append(options, dto.IPOptionDTO{
			IP:          opt.IP,
			Environment: opt.Environment,
			Description: opt.Description,
		})
	}

	return dto.PortInfoDTO{
		InternalPort: portConfig.InternalPort(),
		ExternalPort: portConfig.ExternalPort(),
		ExternalIP:   portConfig.ExternalIP(),
		Name:         portConfig.Name(),
		Description:  portConfig.Description(),
		Options:      options,
	}
}

// validatePortConfig 校验端口配置是否合法
func (s *PortApplicationService) validatePortConfig(internalPort int, internalIP string) error {
	// 1. 校验端口是否在配置的端口映射中
	if _, exists := s.portMappings[internalPort]; !exists {
		return fmt.Errorf("端口 %d 不在允许的配置范围内，允许的端口: %v",
			internalPort, s.getAllowedPorts())
	}

	// 2. 校验IP是否在配置的主机列表中
	hostConfig, exists := s.hosts[internalIP]
	if !exists {
		return fmt.Errorf("IP地址 %s 不在允许的配置范围内，允许的IP: %v",
			internalIP, s.getAllowedIPs())
	}

	// 3. 校验该IP是否支持该端口的服务
	portSupported := false
	for _, servicePort := range hostConfig.Services {
		if servicePort == internalPort {
			portSupported = true
			break
		}
	}

	if !portSupported {
		return fmt.Errorf("IP地址 %s 不支持端口 %d 的服务，该IP支持的端口: %v",
			internalIP, internalPort, s.getSupportedPorts(internalIP))
	}

	return nil
}

// getAllowedPorts 获取所有允许的端口
func (s *PortApplicationService) getAllowedPorts() []int {
	var ports []int
	for port := range s.portMappings {
		ports = append(ports, port)
	}
	return ports
}

// getAllowedIPs 获取所有允许的IP地址
func (s *PortApplicationService) getAllowedIPs() []string {
	var ips []string
	for ip := range s.hosts {
		ips = append(ips, ip)
	}
	return ips
}

// getSupportedPorts 获取指定IP支持的端口
func (s *PortApplicationService) getSupportedPorts(ip string) []int {
	var ports []int
	if hostConfig, exists := s.hosts[ip]; exists {
		for _, port := range hostConfig.Services {
			ports = append(ports, port)
		}
	}
	return ports
}

// LogOperation 记录操作日志
// LogOperation 记录操作日志
func (s *PortApplicationService) LogOperation(operatorIP, operation, status, message, sourcePortIP, targetPortIP string) {
	logEntry := entity.NewOperationLog(operatorIP, operation, status, message, sourcePortIP, targetPortIP)
	if err := s.logRepo.SaveLog(logEntry); err != nil {
		log.Printf("保存操作日志失败: %v", err)
	}
}

// GetOperationLogs 获取操作日志
func (s *PortApplicationService) GetOperationLogs(limit int) (dto.OperationLogsResponse, error) {
	logs, err := s.logRepo.GetRecentLogs(limit)
	if err != nil {
		return dto.OperationLogsResponse{}, fmt.Errorf("获取操作日志失败: %v", err)
	}

	var logDTOs []dto.OperationLogDTO
	for _, logEntry := range logs {
		logDTOs = append(logDTOs, dto.OperationLogDTO{
			ID:           logEntry.ID(),
			OperatorIP:   logEntry.OperatorIP(),
			Operation:    logEntry.Operation(),
			Status:       logEntry.Status(),
			Message:      logEntry.Message(),
			SourcePortIP: logEntry.SourcePortIP(),
			TargetPortIP: logEntry.TargetPortIP(),
			Timestamp:    logEntry.Timestamp().Format("2006-01-02 15:04:05"),
		})
	}

	return dto.OperationLogsResponse{Logs: logDTOs}, nil
}
