package main

import (
	"log"
	"os"
	"time"

	"port-switch-backend/internal/application/service"
	domainservice "port-switch-backend/internal/domain/service"
	"port-switch-backend/internal/infrastructure/config"
	"port-switch-backend/internal/infrastructure/repository"
	"port-switch-backend/internal/interface/http/handler"
	"port-switch-backend/internal/interface/http/router"
)

func main() {
	// 1. 加载配置
	cfg, revertCfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 2. 初始化基础设施层
	// 计算缓存过期时间
	var cacheExpiry time.Duration
	if cfg.Cache.UseTestInterval {
		cacheExpiry = time.Duration(cfg.Cache.TestIntervalMinutes) * time.Minute
		log.Printf("使用测试缓存过期时间: %v", cacheExpiry)
	} else {
		cacheExpiry = time.Duration(cfg.Cache.UpdateIntervalMinutes) * time.Minute
		log.Printf("使用生产缓存过期时间: %v", cacheExpiry)
	}

	// Redis缓存仓储
	cacheRepo, err := repository.NewRedisCacheRepository(&cfg.Redis, cacheExpiry)
	if err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}

	// Redis操作日志仓储
	logRepo, err := repository.NewRedisOperationLogRepository(&cfg.Redis)
	if err != nil {
		log.Fatalf("Redis操作日志仓储初始化失败: %v", err)
	}

	// SSH NAT仓储
	natRepo := repository.NewSSHNATRepository(&cfg.H3CMSR2600)

	// 3. 初始化领域服务
	natDomainService := domainservice.NewNATService(natRepo, cacheRepo)

	// 4. 初始化应用服务
	// 转换主机配置格式
	hosts := make(map[string]service.HostConfig)
	for ip, hostCfg := range revertCfg.Hosts {
		hosts[ip] = service.HostConfig{
			Env:      hostCfg.Env,
			Services: hostCfg.Services,
		}
	}

	portAppService := service.NewPortApplicationService(
		natDomainService,
		revertCfg.PortMappings,
		hosts,
		cfg.H3CMSR2600.ExternalIP,
		logRepo,
	)

	// 5. 初始化接口层
	portHandler := handler.NewPortHandler(portAppService, cacheRepo)

	// 6. 启动时立即更新一次缓存
	go func() {
		if err := portAppService.UpdateCache(); err != nil {
			log.Printf("初始缓存更新失败: %v", err)
		}
	}()

	// 7. 启动定时更新任务
	go startCacheUpdateScheduler(portAppService, &cfg.Cache)

	// 8. 设置路由并启动服务器
	r := router.SetupRoutes(portHandler)

	// 获取端口配置
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// startCacheUpdateScheduler 启动缓存更新调度器
func startCacheUpdateScheduler(portService *service.PortApplicationService, cacheConfig *config.CacheConfig) {
	var interval time.Duration
	
	if cacheConfig.UseTestInterval {
		interval = time.Duration(cacheConfig.TestIntervalMinutes) * time.Minute
		log.Printf("使用测试间隔: %d分钟", cacheConfig.TestIntervalMinutes)
	} else {
		interval = time.Duration(cacheConfig.UpdateIntervalMinutes) * time.Minute
		log.Printf("使用生产间隔: %d分钟", cacheConfig.UpdateIntervalMinutes)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := portService.UpdateCache(); err != nil {
				log.Printf("定时缓存更新失败: %v", err)
			}
		}
	}
}