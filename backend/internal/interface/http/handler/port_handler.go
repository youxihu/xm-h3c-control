package handler

import (
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"

	"port-switch-backend/internal/application/dto"
	"port-switch-backend/internal/application/service"
	"port-switch-backend/internal/infrastructure/utils"
)

// PortHandler 端口处理器
type PortHandler struct {
	portService *service.PortApplicationService
	cacheRepo   CacheRepository
}

// CacheRepository 缓存仓储接口（避免循环依赖）
type CacheRepository interface {
	GetPortStatus() (map[string]string, error)
}

// NewPortHandler 创建端口处理器
func NewPortHandler(portService *service.PortApplicationService, cacheRepo CacheRepository) *PortHandler {
	return &PortHandler{
		portService: portService,
		cacheRepo:   cacheRepo,
	}
}

// SwitchPort 切换端口处理器
func (h *PortHandler) SwitchPort(c *gin.Context) {
	var req dto.SwitchPortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 获取操作者IP信息
	operatorIP := utils.GetClientIP(c.Request)  // 获取客户端IP

	log.Printf("开始切换端口 %d 到内网IP %s，操作者: %s", req.InternalPort, req.NewInternalIP, operatorIP)

	// 构建源端口IP和目标端口IP
	sourcePortIP := fmt.Sprintf("%d_%s", req.InternalPort, req.CurrentInternalIP)
	targetPortIP := fmt.Sprintf("%d_%s", req.InternalPort, req.NewInternalIP)
	
	if err := h.portService.SwitchPort(req); err != nil {
		log.Printf("端口切换失败: %v", err)
		
		// 记录失败日志
		h.portService.LogOperation(operatorIP, "端口切换", "失败", err.Error(), sourcePortIP, targetPortIP)
		
		// 检查是否是"已经映射"的情况
		if strings.Contains(err.Error(), "已经映射到") {
			c.JSON(200, gin.H{
				"message":             err.Error(),
				"status":              "already_mapped",
				"internal_port":       req.InternalPort,
				"current_internal_ip": req.NewInternalIP,
			})
			return
		}
		
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 端口切换成功后立即更新缓存
	log.Printf("单个端口切换成功，立即更新缓存")
	if err := h.portService.UpdateCacheAfterSwitch(req.InternalPort, req.NewInternalIP); err != nil {
		log.Printf("更新缓存失败: %v", err)
	}

	// 记录成功日志
	h.portService.LogOperation(operatorIP, "端口切换", "成功", "端口切换成功", sourcePortIP, targetPortIP)

	c.JSON(200, gin.H{
		"message":         "端口切换成功",
		"status":          "switched",
		"internal_port":   req.InternalPort,
		"new_internal_ip": req.NewInternalIP,
	})
}

// ApplyConfig 批量配置应用处理器
func (h *PortHandler) ApplyConfig(c *gin.Context) {
	var req dto.ApplyConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 获取操作者IP信息
	operatorIP := utils.GetClientIP(c.Request)  // 获取客户端IP

	log.Printf("开始批量应用配置，操作者: %s，共 %d 个端口", operatorIP, len(req.Configs))

	results, err := h.portService.ApplyBatchConfigWithLog(req, operatorIP)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
			"results": results,
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "所有配置应用成功",
		"results": results,
	})
}

// GetPortConfig 获取端口配置处理器
func (h *PortHandler) GetPortConfig(c *gin.Context) {
	response := h.portService.GetPortConfig()
	log.Printf("返回端口配置，共 %d 个端口，顺序固定", len(response.Ports))
	c.JSON(200, response)
}

// GetPortStatus 获取端口状态处理器
func (h *PortHandler) GetPortStatus(c *gin.Context) {
	status, err := h.portService.GetPortStatus()
	if err != nil {
		log.Printf("获取端口状态失败: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, status)
}

// GetPortStatusFromCache 从缓存获取端口状态处理器
func (h *PortHandler) GetPortStatusFromCache(c *gin.Context) {
	// 先尝试从缓存获取
	cachedStatus, err := h.cacheRepo.GetPortStatus()
	
	if err != nil {
		log.Printf("从缓存获取状态失败，需要查询实时数据并更新缓存: %v", err)
		h.getStatusAndUpdateCache(c)
		return
	}
	
	if len(cachedStatus) == 0 {
		log.Printf("缓存为空，需要查询实时数据并更新缓存")
		h.getStatusAndUpdateCache(c)
		return
	}

	// 缓存有效，直接返回缓存数据
	log.Printf("返回缓存数据，共 %d 个端口状态", len(cachedStatus))
	c.JSON(200, cachedStatus)
}

// getStatusAndUpdateCache 获取状态并更新缓存
func (h *PortHandler) getStatusAndUpdateCache(c *gin.Context) {
	status, err := h.portService.GetPortStatus()
	if err != nil {
		log.Printf("获取端口状态失败: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 更新缓存
	go h.portService.UpdateCache()

	c.JSON(200, status)
}

// isStatusEqual 比较两个状态是否相等
func (h *PortHandler) isStatusEqual(status1, status2 map[string]string) bool {
	if len(status1) != len(status2) {
		return false
	}

	for key, value1 := range status1 {
		if value2, exists := status2[key]; !exists || value1 != value2 {
			return false
		}
	}

	return true
}
// GetOperationLogs 获取操作日志处理器
func (h *PortHandler) GetOperationLogs(c *gin.Context) {
	// 默认获取最近20条日志
	limit := 20
	
	logs, err := h.portService.GetOperationLogs(limit)
	if err != nil {
		log.Printf("获取操作日志失败: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, logs)
}