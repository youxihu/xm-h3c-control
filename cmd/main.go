package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"h3c-nat-manager/internal/application"
)

const (
	ExitSuccess = 0
	ExitFailure = 1
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

	// 创建应用程序
	app, err := application.NewApp(&application.Config{
		Mode:       *mode,
		ConfigFile: *configFile,
		DescFile:   *descFile,
	})
	if err != nil {
		log.Printf("创建应用程序失败: %v", err)
		os.Exit(ExitFailure)
	}
	defer app.Close()

	// 运行应用程序
	if err := app.Run(ctx, *mode); err != nil {
		log.Printf("程序执行失败: %v", err)
		os.Exit(ExitFailure)
	}

	log.Println("程序执行完成")
	os.Exit(ExitSuccess)
}