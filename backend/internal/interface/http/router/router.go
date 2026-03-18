package router

import (
	"github.com/gin-gonic/gin"

	"port-switch-backend/internal/interface/http/handler"
)

// SetupRoutes 设置路由
func SetupRoutes(portHandler *handler.PortHandler) *gin.Engine {
	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Client-Internal-IP, X-Client-External-IP")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API路由组
	api := r.Group("/api")
	{
		// 端口切换API - 单个端口切换（保留兼容性）
		api.POST("/switch-port", portHandler.SwitchPort)
		
		// 批量配置应用API
		api.POST("/apply-config", portHandler.ApplyConfig)
		
		// 获取当前映射状态 - 从Redis获取（智能缓存管理）
		api.GET("/port-status", portHandler.GetPortStatusFromCache)
		
		// 获取端口配置信息
		api.GET("/port-config", portHandler.GetPortConfig)
		
		// 获取操作日志
		api.GET("/operation-logs", portHandler.GetOperationLogs)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}