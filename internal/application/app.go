package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"h3c-nat-manager/internal/application/service"
	"h3c-nat-manager/internal/infrastructure/config"
	"h3c-nat-manager/internal/infrastructure/description"
	"h3c-nat-manager/internal/infrastructure/notification"
	"h3c-nat-manager/internal/infrastructure/router"
)

// App 应用程序结构
type App struct {
	natManager *service.NATManagerService
	h3cClient  *router.H3CClient
}

// Config 应用配置
type Config struct {
	Mode       string
	ConfigFile string
	DescFile   string
}

// NewApp 创建应用程序实例
func NewApp(cfg *Config) (*App, error) {
	// 加载配置
	appConfig, err := config.LoadConfig(cfg.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %v", err)
	}
	log.Println("配置文件加载成功")

	// 创建描述映射器
	descMapper := description.NewMapper()
	if err := descMapper.LoadMappings(cfg.DescFile); err != nil {
		return nil, fmt.Errorf("加载描述映射失败: %v", err)
	}
	log.Println("描述映射文件加载成功")

	// 创建H3C客户端
	h3cClient := router.NewH3CClientWithExpiryTime(
		appConfig.Router.Host,
		appConfig.Router.User,
		appConfig.Router.Passwd,
		appConfig.Router.ExpiryTime.Hour,
		appConfig.Router.ExpiryTime.Minute,
	)

	// 创建钉钉通知服务
	dingTalkSvc := notification.NewDingTalkService(&appConfig.DingTalk)

	// 创建NAT管理服务
	natManager := service.NewNATManagerService(
		h3cClient,
		dingTalkSvc,
		descMapper,
		appConfig,
	)

	return &App{
		natManager: natManager,
		h3cClient:  h3cClient,
	}, nil
}

// Run 运行应用程序
func (a *App) Run(ctx context.Context, mode string) error {
	// 创建带超时的上下文
	const shutdownTimeout = 30 * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	// 根据模式执行相应操作
	switch mode {
	case "smart":
		log.Println("执行智能处理模式...")
		return a.executeWithContext(timeoutCtx, a.natManager.SmartProcess)
	case "notify":
		log.Println("执行通知模式...")
		return a.executeWithContext(timeoutCtx, a.natManager.CheckAndNotify)
	case "cleanup":
		log.Println("执行清理模式...")
		return a.executeWithContext(timeoutCtx, a.natManager.CleanupExpired)
	default:
		return fmt.Errorf("无效的运行模式: %s", mode)
	}
}

// Close 关闭应用程序资源
func (a *App) Close() {
	if a.h3cClient != nil {
		a.h3cClient.Close()
	}
}

// executeWithContext 在上下文中执行函数
func (a *App) executeWithContext(ctx context.Context, fn func() error) error {
	done := make(chan error, 1)

	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}