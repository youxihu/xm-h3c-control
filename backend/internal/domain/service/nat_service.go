package service

import (
	"fmt"
	"log"
	"time"

	"port-switch-backend/internal/domain/entity"
	"port-switch-backend/internal/domain/repository"
)

// NATService NAT映射领域服务
type NATService struct {
	natRepo   repository.NATRepository
	cacheRepo repository.CacheRepository
}

// NewNATService 创建NAT服务
func NewNATService(natRepo repository.NATRepository, cacheRepo repository.CacheRepository) *NATService {
	return &NATService{
		natRepo:   natRepo,
		cacheRepo: cacheRepo,
	}
}

// SwitchPortMapping 切换端口映射
func (s *NATService) SwitchPortMapping(currentInternalIP, newInternalIP string, internalPort int, portMappings map[int]int, externalIP string) error {
	log.Printf("开始切换端口映射: 内网端口%d -> 新IP%s", internalPort, newInternalIP)

	// 1. 从配置获取外网端口
	externalPort, exists := portMappings[internalPort]
	if !exists {
		return fmt.Errorf("配置中未找到内网端口 %d 对应的外网端口", internalPort)
	}

	// 2. 先从Redis缓存获取当前状态
	portKey := fmt.Sprintf("port_%d", externalPort)
	cachedStatus, err := s.cacheRepo.GetPortStatus()
	if err != nil {
		log.Printf("获取缓存状态失败，继续执行切换: %v", err)
	} else {
		// 检查当前IP是否已经是目标IP
		if currentIP, exists := cachedStatus[portKey]; exists && currentIP == newInternalIP {
			return fmt.Errorf("端口已经映射到目标地址 %s:%d，无需切换", newInternalIP, internalPort)
		}
		// 更新当前IP为缓存中的值（更准确）
		if currentIP, exists := cachedStatus[portKey]; exists && currentIP != "" {
			currentInternalIP = currentIP
		}
	}

	// 3. 创建时间戳
	timestamp := time.Now().Format("20060102150405")

	// 4. 如果有当前IP，先删除旧映射
	if currentInternalIP != "" {
		// 创建旧映射实体用于删除
		oldMapping, err := entity.NewNATMapping(
			"tcp",
			externalIP,
			externalPort,
			currentInternalIP,
			internalPort,
			"old-mapping",
		)
		if err != nil {
			return fmt.Errorf("创建旧映射实体失败: %v", err)
		}

		// 删除旧映射
		if err := s.natRepo.DeleteMapping(oldMapping); err != nil {
			log.Printf("删除旧映射失败，继续创建新映射: %v", err)
		} else {
			log.Printf("旧映射删除成功: %s:%d -> %s:%d", externalIP, externalPort, currentInternalIP, internalPort)
		}
	}

	// 5. 创建新映射
	newMapping, err := entity.NewNATMapping(
		"tcp",
		externalIP,
		externalPort,
		newInternalIP,
		internalPort,
		fmt.Sprintf("%s-switch", timestamp),
	)
	if err != nil {
		return fmt.Errorf("创建新映射实体失败: %v", err)
	}

	// 6. 执行创建新映射
	if err := s.natRepo.CreateMapping(newMapping); err != nil {
		return fmt.Errorf("创建新映射失败: %v", err)
	}

	log.Printf("端口映射切换成功: %s:%d -> %s:%d", externalIP, externalPort, newInternalIP, internalPort)
	return nil
}

// UpdateCacheAfterSwitch 切换后更新缓存
func (s *NATService) UpdateCacheAfterSwitch(internalPort int, newInternalIP string, portMappings map[int]int) error {
	// 获取外网端口
	externalPort, exists := portMappings[internalPort]
	if !exists {
		return fmt.Errorf("配置中未找到内网端口 %d 对应的外网端口", internalPort)
	}

	// 获取当前缓存状态
	currentStatus, err := s.cacheRepo.GetPortStatus()
	if err != nil {
		// 如果获取失败，创建新的状态映射
		currentStatus = make(map[string]string)
		for _, extPort := range portMappings {
			key := fmt.Sprintf("port_%d", extPort)
			currentStatus[key] = "" // 默认为空
		}
	}

	// 更新指定端口的状态
	portKey := fmt.Sprintf("port_%d", externalPort)
	currentStatus[portKey] = newInternalIP

	// 保存到缓存
	if err := s.cacheRepo.SetPortStatus(currentStatus); err != nil {
		return fmt.Errorf("更新缓存失败: %v", err)
	}

	log.Printf("缓存更新成功: %s -> %s", portKey, newInternalIP)
	return nil
}

// createNewMapping 创建新映射
func (s *NATService) createNewMapping(internalIP string, internalPort int, portMappings map[int]int, externalIP string) error {
	// 从配置获取外网端口
	externalPort, exists := portMappings[internalPort]
	if !exists {
		return fmt.Errorf("配置中未找到内网端口 %d 对应的外网端口", internalPort)
	}

	// 创建新映射
	newMapping, err := entity.NewNATMapping(
		"tcp", // 默认TCP协议
		externalIP,
		externalPort,
		internalIP,
		internalPort,
		fmt.Sprintf("new-mapping-%s-%d", internalIP, internalPort),
	)
	if err != nil {
		return fmt.Errorf("创建映射实体失败: %v", err)
	}

	// 执行创建
	if err := s.natRepo.CreateMapping(newMapping); err != nil {
		return fmt.Errorf("创建映射失败: %v", err)
	}

	log.Printf("新映射创建成功: %s", newMapping.String())
	return nil
}

// GetCachedPortStatus 从缓存获取端口状态（不查询路由器）
func (s *NATService) GetCachedPortStatus() (map[string]string, error) {
	return s.cacheRepo.GetPortStatus()
}

// GetCurrentPortStatus 获取当前端口状态
func (s *NATService) GetCurrentPortStatus(portMappings map[int]int) (map[string]string, error) {
	// 获取所有映射
	mappings, err := s.natRepo.GetAllMappings()
	if err != nil {
		return nil, fmt.Errorf("获取NAT映射失败: %v", err)
	}

	// 构建状态映射
	status := make(map[string]string)
	
	for internalPort, externalPort := range portMappings {
		portKey := fmt.Sprintf("port_%d", externalPort)
		status[portKey] = "" // 默认为空
		
		// 查找对应的映射
		for _, mapping := range mappings {
			if mapping.ExternalPort() == externalPort {
				status[portKey] = mapping.InternalIP()
				log.Printf("找到映射: 内网端口%d -> 外网端口%d -> 内网IP%s", 
					internalPort, externalPort, mapping.InternalIP())
				break
			}
		}
		
		if status[portKey] == "" {
			log.Printf("未找到映射: 内网端口%d -> 外网端口%d", internalPort, externalPort)
		}
	}

	return status, nil
}

// UpdateCache 更新缓存
func (s *NATService) UpdateCache(portMappings map[int]int) error {
	status, err := s.GetCurrentPortStatus(portMappings)
	if err != nil {
		return fmt.Errorf("获取端口状态失败: %v", err)
	}

	if err := s.cacheRepo.SetPortStatus(status); err != nil {
		return fmt.Errorf("更新缓存失败: %v", err)
	}

	return nil
}

// BatchSwitchPortMappings 批量切换端口映射（在单个SSH会话中完成）
func (s *NATService) BatchSwitchPortMappings(configs []SwitchConfig, portMappings map[int]int, externalIP string) error {
	if len(configs) == 0 {
		return nil
	}

	log.Printf("开始批量切换端口映射，共 %d 个端口", len(configs))

	// 先从Redis缓存获取当前状态
	cachedStatus, err := s.cacheRepo.GetPortStatus()
	if err != nil {
		log.Printf("获取缓存状态失败，继续执行切换: %v", err)
		cachedStatus = make(map[string]string)
	}

	// 构建批量操作
	var operations []repository.SwitchOperation
	var cacheUpdates = make(map[string]string)

	for _, config := range configs {
		// 1. 从配置获取外网端口
		externalPort, exists := portMappings[config.InternalPort]
		if !exists {
			return fmt.Errorf("配置中未找到内网端口 %d 对应的外网端口", config.InternalPort)
		}

		// 2. 获取当前IP
		portKey := fmt.Sprintf("port_%d", externalPort)
		currentIP := ""
		if val, exists := cachedStatus[portKey]; exists {
			currentIP = val
		}

		// 3. 检查是否需要切换
		if currentIP == config.NewInternalIP {
			log.Printf("端口 %d 已经映射到目标地址 %s，跳过", config.InternalPort, config.NewInternalIP)
			continue
		}

		// 4. 构建切换操作
		var oldMapping *entity.NATMapping
		if currentIP != "" {
			oldMapping, err = entity.NewNATMapping(
				"tcp",
				externalIP,
				externalPort,
				currentIP,
				config.InternalPort,
				"old-mapping",
			)
			if err != nil {
				return fmt.Errorf("创建旧映射实体失败: %v", err)
			}
		}

		newMapping, err := entity.NewNATMapping(
			"tcp",
			externalIP,
			externalPort,
			config.NewInternalIP,
			config.InternalPort,
			fmt.Sprintf("batch-switch-%d", time.Now().Unix()),
		)
		if err != nil {
			return fmt.Errorf("创建新映射实体失败: %v", err)
		}

		operations = append(operations, repository.SwitchOperation{
			OldMapping: oldMapping,
			NewMapping: newMapping,
		})

		// 准备缓存更新
		cacheUpdates[portKey] = config.NewInternalIP

		log.Printf("准备切换端口 %d: %s -> %s", config.InternalPort, currentIP, config.NewInternalIP)
	}

	if len(operations) == 0 {
		log.Printf("没有需要切换的端口")
		return nil
	}

	// 执行批量切换
	if err := s.natRepo.BatchSwitchMappings(operations); err != nil {
		return fmt.Errorf("批量切换失败: %v", err)
	}

	// 批量更新缓存
	for key, value := range cacheUpdates {
		cachedStatus[key] = value
	}
	if err := s.cacheRepo.SetPortStatus(cachedStatus); err != nil {
		log.Printf("批量更新缓存失败: %v", err)
	}

	log.Printf("批量端口映射切换成功")
	return nil
}

// SwitchConfig 切换配置
type SwitchConfig struct {
	InternalPort    int
	CurrentInternalIP string
	NewInternalIP   string
}
