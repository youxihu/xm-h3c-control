package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"h3c-nat-manager/internal/application/service"
	"h3c-nat-manager/internal/infrastructure/config"
	"h3c-nat-manager/internal/infrastructure/description"
	"h3c-nat-manager/internal/infrastructure/notification"
	"h3c-nat-manager/internal/infrastructure/router"
)

const (
	// 程序退出码
	ExitSuccess = 0
	ExitFailure = 1
	
	// 优雅关闭超时时间
	ShutdownTimeout = 30 * time.Second
)

func main() {
	// 解析命令行参数
	mode := flag.String("mode", "smart", "运行模式: smart(智能处理), notify(仅通知), cleanup(仅清理)")
	configFile := flag.String("configs", "configs/config.yaml", "配置文件路径")
	descFile := flag.String("desc", "configs/description.yaml", "描述映射文件路径")
	flag.Parse()

	// 设置优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("收到信号 %v，开始优雅关闭...", sig)
		cancel()
	}()

	// 运行主程序
	if err := run(ctx, *mode, *configFile, *descFile); err != nil {
		log.Printf("程序执行失败: %v", err)
		os.Exit(ExitFailure)
	}

	log.Println("程序执行完成")
	os.Exit(ExitSuccess)
}

func run(ctx context.Context, mode, configFile, descFile string) error {
	// 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}
	log.Println("配置文件加载成功")

	// 创建描述映射器
	descMapper := description.NewMapper()
	if err := descMapper.LoadMappings(descFile); err != nil {
		return err
	}
	log.Println("描述映射文件加载成功")

	// 创建H3C客户端
	h3cClient := router.NewH3CClientWithExpiryTime(
		cfg.Router.Host,
		cfg.Router.User,
		cfg.Router.Passwd,
		cfg.Router.ExpiryTime.Hour,
		cfg.Router.ExpiryTime.Minute,
	)
	defer h3cClient.Close()

	// 创建钉钉通知服务
	dingTalkSvc := notification.NewDingTalkService(&cfg.DingTalk)

	// 创建NAT管理服务
	natManager := service.NewNATManagerService(
		h3cClient,
		dingTalkSvc,
		descMapper,
		cfg,
	)

	// 创建带超时的上下文
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, ShutdownTimeout)
	defer timeoutCancel()

	// 根据模式执行相应操作
	switch mode {
	case "smart":
		log.Println("执行智能处理模式...")
		return executeWithContext(timeoutCtx, natManager.SmartProcess)
	case "notify":
		log.Println("执行通知模式...")
		return executeWithContext(timeoutCtx, natManager.CheckAndNotify)
	case "cleanup":
		log.Println("执行清理模式...")
		return executeWithContext(timeoutCtx, natManager.CleanupExpired)
	default:
		return fmt.Errorf("无效的运行模式: %s", mode)
	}
}

// executeWithContext 在上下文中执行函数
func executeWithContext(ctx context.Context, fn func() error) error {
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