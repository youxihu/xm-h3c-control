package main

import (
	"flag"
	"log"

	"h3c-nat-manager/internal/application/service"
	"h3c-nat-manager/internal/infrastructure/config"
	"h3c-nat-manager/internal/infrastructure/description"
	"h3c-nat-manager/internal/infrastructure/notification"
	"h3c-nat-manager/internal/infrastructure/router"
)

func main() {
	// 解析命令行参数
	mode := flag.String("mode", "smart", "运行模式: smart(智能处理), notify(仅通知), cleanup(仅清理)")
	configFile := flag.String("configs", "configs/config.yaml", "配置文件路径")
	descFile := flag.String("desc", "configs/description.yaml", "描述映射文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 创建描述映射器
	descMapper := description.NewMapper()
	if err := descMapper.LoadMappings(*descFile); err != nil {
		log.Fatalf("加载描述映射失败: %v", err)
	}

	// 创建H3C客户端
	h3cClient := router.NewH3CClientWithExpiryTime(
		cfg.Router.Host,
		cfg.Router.User,
		cfg.Router.Passwd,
		cfg.Router.ExpiryTime.Hour,
		cfg.Router.ExpiryTime.Minute,
	)

	// 创建钉钉通知服务
	dingTalkSvc := notification.NewDingTalkService(&cfg.DingTalk)

	// 创建NAT管理服务
	natManager := service.NewNATManagerService(
		h3cClient,
		dingTalkSvc,
		descMapper,
		cfg,
	)

	// 根据模式执行相应操作
	switch *mode {
	case "smart":
		log.Println("执行智能处理模式...")
		if err := natManager.SmartProcess(); err != nil {
			log.Fatalf("执行智能处理失败: %v", err)
		}
	case "notify":
		log.Println("执行通知模式...")
		if err := natManager.CheckAndNotify(); err != nil {
			log.Fatalf("执行通知失败: %v", err)
		}
	case "cleanup":
		log.Println("执行清理模式...")
		if err := natManager.CleanupExpired(); err != nil {
			log.Fatalf("执行清理失败: %v", err)
		}
	default:
		log.Fatalf("无效的运行模式: %s", *mode)
	}

	log.Println("程序执行完成")
}