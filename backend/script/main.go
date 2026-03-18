package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	H3CMSR2600 RouterConfig `yaml:"h3c-msr2600"`
	Redis      RedisConfig  `yaml:"redis"`
	Cache      CacheConfig  `yaml:"cache"`
}

// RouterConfig 路由器配置
type RouterConfig struct {
	Host       string `yaml:"host"`
	User       string `yaml:"user"`
	Passwd     string `yaml:"passwd"`
	ExternalIP string `yaml:"external_ip"`
	ExpiryTime struct {
		Hour   int `yaml:"hour"`
		Minute int `yaml:"minute"`
	} `yaml:"expiry_time"`
	ReminderBeforeExpiration int `yaml:"Reminder_before_expiration"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	UpdateIntervalMinutes int  `yaml:"update_interval_minutes"`
	TestIntervalMinutes   int  `yaml:"test_interval_minutes"`
	UseTestInterval       bool `yaml:"use_test_interval"`
}

// RevertConfig revert配置结构
type RevertConfig struct {
	PortMappings map[int]int `yaml:"port_mappings"`
	Hosts        map[string]HostConfig `yaml:"hosts"`
}

// HostConfig 主机配置
type HostConfig struct {
	Env      string         `yaml:"env"`
	Services map[string]int `yaml:"services"`
}

// 全局配置变量
var config Config
var revertConfig RevertConfig
var routerConfig RouterConfig
var redisClient *redis.Client
var ctx = context.Background()

// Redis键名常量
const (
	REDIS_KEY_PORT_STATUS = "port_status"
	REDIS_KEY_LAST_UPDATE = "last_update"
)

// loadConfig 加载配置文件
func loadConfig() error {
	// 加载主配置文件
	data, err := ioutil.ReadFile("config/config.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	routerConfig = config.H3CMSR2600
	
	// 加载revert配置文件
	revertData, err := ioutil.ReadFile("config/revert.yaml")
	if err != nil {
		return fmt.Errorf("读取revert配置文件失败: %v", err)
	}

	if err := yaml.Unmarshal(revertData, &revertConfig); err != nil {
		return fmt.Errorf("解析revert配置文件失败: %v", err)
	}

	log.Printf("加载配置成功: 路由器地址 %s, 用户 %s, 外网IP %s", routerConfig.Host, routerConfig.User, routerConfig.ExternalIP)
	log.Printf("端口映射配置: %+v", revertConfig.PortMappings)
	log.Printf("Redis配置: %s:%d", config.Redis.Host, config.Redis.Port)
	return nil
}

// initRedis 初始化Redis连接
func initRedis() error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	// 测试连接
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis连接失败: %v", err)
	}

	log.Printf("Redis连接成功")
	return nil
}

func main() {
	// 加载配置文件
	if err := loadConfig(); err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 初始化Redis
	if err := initRedis(); err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}

	// 启动时立即更新一次缓存
	go updatePortStatusCache()

	// 启动定时更新任务
	go startCacheUpdateScheduler()

	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 端口切换API - 单个端口切换（保留兼容性）
	r.POST("/api/switch-port", switchPortHandler)
	
	// 批量配置应用API - 新增
	r.POST("/api/apply-config", applyConfigHandler)
	
	// 获取当前映射状态 - 从Redis获取
	r.GET("/api/port-status", getPortStatusFromCacheHandler)
	
	// 获取端口配置信息 - 新增
	r.GET("/api/port-config", getPortConfigHandler)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("Server starting on :8080")
	r.Run(":8080")
}

// SwitchPortRequest 切换端口请求
type SwitchPortRequest struct {
	CurrentInternalIP string `json:"current_internal_ip" binding:"required"`
	NewInternalIP     string `json:"new_internal_ip" binding:"required"`
	InternalPort      int    `json:"internal_port" binding:"required"`
}

// ApplyConfigRequest 批量配置应用请求
type ApplyConfigRequest struct {
	Configs []PortConfig `json:"configs" binding:"required"`
}

// PortConfig 端口配置
type PortConfig struct {
	InternalPort int    `json:"internal_port" binding:"required"`
	InternalIP   string `json:"internal_ip" binding:"required"`
}

// PortConfigResponse 端口配置响应
type PortConfigResponse struct {
	Ports []PortInfo `json:"ports"`
}

// PortInfo 端口信息
type PortInfo struct {
	InternalPort int        `json:"internal_port"`
	ExternalPort int        `json:"external_port"`
	ExternalIP   string     `json:"external_ip"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Options      []IPOption `json:"options"`
}

// IPOption IP选项
type IPOption struct {
	IP          string `json:"ip"`
	Environment string `json:"environment"`
	Description string `json:"description"`
}

// NATEntry NAT映射条目
type NATEntry struct {
	Protocol     string
	ExternalIP   string
	ExternalPort int
	InternalIP   string
	InternalPort int
	Description  string
}

// getPortConfigHandler 获取端口配置处理器
func getPortConfigHandler(c *gin.Context) {
	// 定义固定的端口顺序，确保前端页面布局稳定
	portOrder := []struct {
		InternalPort int
		Name         string
		Description  string
	}{
		{61002, "drone_control_link", "侦测无人机控制链路端口"},
		{61100, "drone_data_link", "侦测无人机数据链路端口"},
		{48080, "dongwu_backend", "东吴后端端口"},
	}
	
	var ports []PortInfo
	
	// 按照固定顺序构建端口信息
	for _, portDef := range portOrder {
		// 从配置文件获取对应的外网端口
		externalPort, exists := revertConfig.PortMappings[portDef.InternalPort]
		if !exists {
			log.Printf("警告: 配置文件中未找到内网端口 %d 的映射", portDef.InternalPort)
			continue
		}
		
		var options []IPOption
		
		// 从hosts配置中找到支持该服务的IP
		for hostIP, hostConfig := range revertConfig.Hosts {
			for serviceName, servicePort := range hostConfig.Services {
				if servicePort == portDef.InternalPort && serviceName == portDef.Name {
					var envDesc string
					switch hostConfig.Env {
					case "dev":
						envDesc = "开发环境"
					case "zc-test":
						envDesc = "测试环境"
					case "dw-test":
						envDesc = "东吴环境"
					default:
						envDesc = hostConfig.Env
					}
					
					options = append(options, IPOption{
						IP:          hostIP,
						Environment: hostConfig.Env,
						Description: envDesc,
					})
				}
			}
		}
		
		// 按IP地址排序，确保选项顺序也是固定的
		// 简单排序：dev环境(.211) -> zc-test环境(.221) -> dw-test环境(.218)
		sortedOptions := make([]IPOption, 0, len(options))
		
		// 先添加dev环境
		for _, opt := range options {
			if opt.Environment == "dev" {
				sortedOptions = append(sortedOptions, opt)
			}
		}
		// 再添加zc-test环境
		for _, opt := range options {
			if opt.Environment == "zc-test" {
				sortedOptions = append(sortedOptions, opt)
			}
		}
		// 最后添加dw-test环境
		for _, opt := range options {
			if opt.Environment == "dw-test" {
				sortedOptions = append(sortedOptions, opt)
			}
		}
		
		ports = append(ports, PortInfo{
			InternalPort: portDef.InternalPort,
			ExternalPort: externalPort,
			ExternalIP:   routerConfig.ExternalIP,
			Name:         portDef.Name,
			Description:  portDef.Description,
			Options:      sortedOptions,
		})
	}
	
	response := PortConfigResponse{
		Ports: ports,
	}
	
	log.Printf("返回端口配置，共 %d 个端口，顺序固定", len(ports))
	c.JSON(200, response)
}

// applyConfigHandler 批量配置应用处理器
func applyConfigHandler(c *gin.Context) {
	var req ApplyConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	log.Printf("开始批量应用配置，共 %d 个端口", len(req.Configs))

	// 获取当前映射状态
	currentStatus, err := getCurrentPortStatus()
	if err != nil {
		log.Printf("获取当前状态失败: %v", err)
		c.JSON(500, gin.H{"error": "获取当前状态失败: " + err.Error()})
		return
	}

	// 批量处理配置变更
	var results []map[string]interface{}
	var hasError bool

	for _, config := range req.Configs {
		result := map[string]interface{}{
			"internal_port": config.InternalPort,
			"target_ip":     config.InternalIP,
		}

		// 检查当前映射状态
		currentIP := ""
		// 根据内网端口找到对应的外网端口，然后构建键名
		if externalPort, exists := revertConfig.PortMappings[config.InternalPort]; exists {
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

		// 执行端口切换
		if err := performPortSwitch(currentIP, config.InternalIP, config.InternalPort); err != nil {
			log.Printf("端口 %d 切换失败: %v", config.InternalPort, err)
			result["status"] = "error"
			result["message"] = err.Error()
			hasError = true
		} else {
			result["status"] = "success"
			result["message"] = "切换成功"
		}

		results = append(results, result)
	}

	// 返回结果
	if hasError {
		// 即使有错误，也尝试更新缓存以保持一致性
		log.Printf("配置应用过程中有错误，但仍尝试更新缓存")
		updatePortStatusCache()
		
		c.JSON(500, gin.H{
			"message": "部分配置应用失败",
			"results": results,
		})
	} else {
		// 配置应用成功后立即同步更新缓存
		log.Printf("所有配置应用成功，立即更新缓存")
		updatePortStatusCache()
		
		c.JSON(200, gin.H{
			"message": "所有配置应用成功",
			"results": results,
		})
	}
}

// switchPortHandler 切换端口处理器
func switchPortHandler(c *gin.Context) {
	var req SwitchPortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	log.Printf("开始切换端口 %d 到内网IP %s", req.InternalPort, req.NewInternalIP)

	// 执行端口切换
	if err := performPortSwitch(req.CurrentInternalIP, req.NewInternalIP, req.InternalPort); err != nil {
		log.Printf("Port switch failed: %v", err)
		
		// 检查是否是"已经映射"的情况
		if strings.Contains(err.Error(), "已经映射到") {
			c.JSON(200, gin.H{
				"message": err.Error(),
				"status": "already_mapped",
				"internal_port": req.InternalPort,
				"current_internal_ip": req.NewInternalIP,
			})
			return
		}
		
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 端口切换成功后立即同步更新缓存
	log.Printf("单个端口切换成功，立即更新缓存")
	updatePortStatusCache()

	c.JSON(200, gin.H{
		"message": "端口切换成功",
		"status": "switched",
		"internal_port": req.InternalPort,
		"new_internal_ip": req.NewInternalIP,
	})
}

// getPortStatusHandler 获取端口状态处理器
func getPortStatusHandler(c *gin.Context) {
	status, err := getCurrentPortStatus()
	if err != nil {
		log.Printf("Get port status failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, status)
}
// performPortSwitch 执行端口切换 - 基于md文档的逻辑
func performPortSwitch(currentInternalIP, newInternalIP string, internalPort int) error {
	log.Printf("开始切换端口 %s:%d 到内网IP %s:%d", currentInternalIP, internalPort, newInternalIP, internalPort)

	// 1. 连接路由器
	client, err := connectRouter()
	if err != nil {
		return fmt.Errorf("连接路由器失败: %v", err)
	}
	defer client.Close()

	// 2. 获取当前NAT映射
	currentMappings, err := getCurrentNATMappings(client)
	if err != nil {
		return fmt.Errorf("获取当前NAT映射失败: %v", err)
	}

	// 3. 找到当前内网IP:端口对应的外网映射
	var targetMapping *NATEntry
	for _, mapping := range currentMappings {
		if mapping.InternalIP == currentInternalIP && mapping.InternalPort == internalPort {
			targetMapping = mapping
			break
		}
	}

	if targetMapping == nil {
		log.Printf("未找到内网 %s:%d 的映射，将创建新映射", currentInternalIP, internalPort)
		
		// 从配置文件获取外网端口
		externalPort, exists := revertConfig.PortMappings[internalPort]
		if !exists {
			return fmt.Errorf("配置文件中未找到内网端口 %d 对应的外网端口", internalPort)
		}
		
		// 创建新映射
		newMapping := &NATEntry{
			Protocol:     "tcp", // 默认使用TCP
			ExternalIP:   routerConfig.ExternalIP, // 使用配置的外网IP
			ExternalPort: externalPort, // 使用配置的外网端口
			InternalIP:   newInternalIP,
			InternalPort: internalPort,
			Description:  fmt.Sprintf("new-mapping-%s-%d", newInternalIP, internalPort),
		}

		if err := createNATMappingInOneSession(client, newMapping); err != nil {
			return fmt.Errorf("创建新映射失败: %v", err)
		}

		log.Printf("新映射创建成功: %s:%d -> %s:%d", newMapping.ExternalIP, newMapping.ExternalPort, newInternalIP, internalPort)
		return nil
	}

	// 4. 检查是否需要切换
	if targetMapping.InternalIP == newInternalIP {
		log.Printf("端口 %s:%d 已经映射到 %s，无需切换", currentInternalIP, internalPort, newInternalIP)
		return fmt.Errorf("端口已经映射到 %s", newInternalIP)
	}

	log.Printf("当前映射: %s:%d -> %s:%d, 准备切换到: %s:%d -> %s:%d", 
		targetMapping.ExternalIP, targetMapping.ExternalPort, currentInternalIP, internalPort,
		targetMapping.ExternalIP, targetMapping.ExternalPort, newInternalIP, internalPort)

	// 5. 在同一个SSH会话中删除旧映射并创建新映射
	newMapping := &NATEntry{
		Protocol:     targetMapping.Protocol,
		ExternalIP:   targetMapping.ExternalIP,
		ExternalPort: targetMapping.ExternalPort, // 保持外网端口不变
		InternalIP:   newInternalIP,
		InternalPort: internalPort,
		Description:  fmt.Sprintf("switched-to-%s-%d", newInternalIP, internalPort),
	}

	if err := switchNATMappingInOneSession(client, targetMapping, newMapping); err != nil {
		return fmt.Errorf("切换映射失败: %v", err)
	}

	log.Printf("端口切换成功: %s:%d -> %s:%d", targetMapping.ExternalIP, targetMapping.ExternalPort, newInternalIP, internalPort)
	return nil
}

// connectRouter 连接路由器
func connectRouter() (*ssh.Client, error) {
	log.Printf("正在连接路由器 %s，用户: %s", routerConfig.Host, routerConfig.User)
	
	config := &ssh.ClientConfig{
		User: routerConfig.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(routerConfig.Passwd),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second, // 连接超时30秒
	}

	address := fmt.Sprintf("%s:22", routerConfig.Host)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		log.Printf("SSH连接失败: %v", err)
		return nil, err
	}

	log.Printf("SSH连接成功，连接地址: %s", address)
	return client, nil
}

// executeCommand 执行SSH命令
func executeCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// getCurrentNATMappings 获取当前NAT映射 - 基于md文档
func getCurrentNATMappings(client *ssh.Client) ([]*NATEntry, error) {
	log.Printf("开始获取NAT映射配置")
	
	// 在一个session中连续执行命令
	log.Printf("执行命令: screen-length disable 然后 display nat server")
	output, err := executeCommand(client, "screen-length disable\ndisplay nat server")
	if err != nil {
		log.Printf("获取NAT配置失败: %v", err)
		return nil, fmt.Errorf("获取NAT配置失败: %v", err)
	}

	log.Printf("NAT配置输出长度: %d 字节", len(output))
	if len(output) > 0 {
		log.Printf("NAT配置输出前500字符: %s", output[:min(500, len(output))])
	}

	// 解析输出
	mappings := parseNATOutput(output)
	log.Printf("解析到 %d 个NAT映射", len(mappings))
	
	for _, mapping := range mappings {
		log.Printf("映射: %s:%d -> %s:%d (%s)", 
			mapping.ExternalIP, mapping.ExternalPort, 
			mapping.InternalIP, mapping.InternalPort, 
			mapping.Protocol)
	}
	
	return mappings, nil
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseNATOutput 解析NAT输出
func parseNATOutput(output string) []*NATEntry {
	var entries []*NATEntry
	lines := strings.Split(output, "\n")
	
	var currentEntry *NATEntry
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 匹配接口行
		if strings.HasPrefix(line, "Interface:") {
			if currentEntry != nil {
				entries = append(entries, currentEntry)
			}
			currentEntry = &NATEntry{}
		}
		
		if currentEntry == nil {
			continue
		}
		
		// 匹配协议行
		if strings.HasPrefix(line, "Protocol:") {
			protocolStr := strings.TrimSpace(strings.TrimPrefix(line, "Protocol:"))
			if strings.Contains(protocolStr, "TCP") {
				currentEntry.Protocol = "tcp"
			} else if strings.Contains(protocolStr, "UDP") {
				currentEntry.Protocol = "udp"
			}
		}
		
		// 匹配全局IP/端口行
		if strings.HasPrefix(line, "Global IP/port:") {
			globalAddr := strings.TrimSpace(strings.TrimPrefix(line, "Global IP/port:"))
			parseAddress(globalAddr, &currentEntry.ExternalIP, &currentEntry.ExternalPort)
		}
		
		// 匹配本地IP/端口行
		if strings.HasPrefix(line, "Local IP/port") {
			localAddr := strings.TrimSpace(strings.Split(line, ":")[1])
			parseAddress(localAddr, &currentEntry.InternalIP, &currentEntry.InternalPort)
		}
		
		// 匹配描述行
		if strings.HasPrefix(line, "Description") {
			currentEntry.Description = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}
	
	// 处理最后一个条目
	if currentEntry != nil {
		entries = append(entries, currentEntry)
	}
	
	return entries
}

// parseAddress 解析IP地址和端口
func parseAddress(addr string, ip *string, port *int) error {
	re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)/(\d+)`)
	matches := re.FindStringSubmatch(addr)
	
	if len(matches) != 3 {
		return fmt.Errorf("无效的地址格式: %s", addr)
	}
	
	*ip = matches[1]
	
	portNum, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("无效的端口号: %s", matches[2])
	}
	
	*port = portNum
	return nil
}
// createNATMappingInOneSession 在一个SSH会话中创建新映射
func createNATMappingInOneSession(client *ssh.Client, entry *NATEntry) error {
	log.Printf("在单个会话中创建映射: %s:%d -> %s:%d", 
		entry.ExternalIP, entry.ExternalPort, entry.InternalIP, entry.InternalPort)

	// 在一个session中连续执行：system-view -> interface -> nat server -> quit -> quit
	createCmd := fmt.Sprintf(`system-view
interface GigabitEthernet0/0
nat server protocol %s global %s %d inside %s %d description %s
quit
quit`,
		entry.Protocol, entry.ExternalIP, entry.ExternalPort, 
		entry.InternalIP, entry.InternalPort, entry.Description,
	)

	log.Printf("执行创建命令序列: %s", strings.ReplaceAll(createCmd, "\n", " -> "))
	
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(createCmd)
	if err != nil {
		log.Printf("创建命令执行失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("创建命令执行失败: %v", err)
	}

	log.Printf("创建命令执行成功，输出: %s", string(output))
	return nil
}

// switchNATMappingInOneSession 在一个SSH会话中完成删除和创建操作
func switchNATMappingInOneSession(client *ssh.Client, oldEntry, newEntry *NATEntry) error {
	log.Printf("在单个会话中切换映射: %s:%d -> %s:%d 到 %s:%d -> %s:%d", 
		oldEntry.ExternalIP, oldEntry.ExternalPort, oldEntry.InternalIP, oldEntry.InternalPort,
		newEntry.ExternalIP, newEntry.ExternalPort, newEntry.InternalIP, newEntry.InternalPort)

	// 在一个session中连续执行：system-view -> interface -> undo -> nat server -> quit -> quit
	switchCmd := fmt.Sprintf(`system-view
interface GigabitEthernet0/0
undo nat server protocol %s global %s %d
nat server protocol %s global %s %d inside %s %d description %s
quit
quit`,
		oldEntry.Protocol, oldEntry.ExternalIP, oldEntry.ExternalPort,
		newEntry.Protocol, newEntry.ExternalIP, newEntry.ExternalPort, 
		newEntry.InternalIP, newEntry.InternalPort, newEntry.Description,
	)

	log.Printf("执行切换命令序列: %s", strings.ReplaceAll(switchCmd, "\n", " -> "))
	
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(switchCmd)
	if err != nil {
		log.Printf("切换命令执行失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("切换命令执行失败: %v", err)
	}

	log.Printf("切换命令执行成功，输出: %s", string(output))
	return nil
}

// getCurrentPortStatus 获取当前端口状态
func getCurrentPortStatus() (map[string]string, error) {
	// 连接路由器
	client, err := connectRouter()
	if err != nil {
		return nil, fmt.Errorf("连接路由器失败: %v", err)
	}
	defer client.Close()

	// 获取当前映射
	mappings, err := getCurrentNATMappings(client)
	if err != nil {
		return nil, fmt.Errorf("获取NAT映射失败: %v", err)
	}

	// 构建状态响应
	status := make(map[string]string)
	
	// 根据配置文件中的端口映射查找对应的内网IP
	for internalPort, externalPort := range revertConfig.PortMappings {
		portKey := fmt.Sprintf("port_%d", externalPort) // 使用外网端口作为键名
		status[portKey] = "" // 默认为空
		
		// 在NAT映射中查找对应的外网端口
		for _, mapping := range mappings {
			if mapping.ExternalPort == externalPort {
				status[portKey] = mapping.InternalIP
				log.Printf("找到映射: 内网端口%d -> 外网端口%d -> 内网IP%s", internalPort, externalPort, mapping.InternalIP)
				break
			}
		}
		
		if status[portKey] == "" {
			log.Printf("未找到映射: 内网端口%d -> 外网端口%d", internalPort, externalPort)
		}
	}

	return status, nil
}
// updatePortStatusCache 更新端口状态缓存
func updatePortStatusCache() {
	log.Printf("开始更新端口状态缓存")
	
	status, err := getCurrentPortStatusDirect()
	if err != nil {
		log.Printf("获取端口状态失败: %v", err)
		return
	}

	// 将状态序列化为JSON
	statusJSON, err := json.Marshal(status)
	if err != nil {
		log.Printf("序列化状态失败: %v", err)
		return
	}

	// 存储到Redis
	err = redisClient.Set(ctx, REDIS_KEY_PORT_STATUS, statusJSON, 0).Err()
	if err != nil {
		log.Printf("存储到Redis失败: %v", err)
		return
	}

	// 更新最后更新时间
	err = redisClient.Set(ctx, REDIS_KEY_LAST_UPDATE, time.Now().Unix(), 0).Err()
	if err != nil {
		log.Printf("更新时间戳失败: %v", err)
		return
	}

	log.Printf("端口状态缓存更新成功: %+v", status)
}

// startCacheUpdateScheduler 启动缓存更新调度器
func startCacheUpdateScheduler() {
	var interval time.Duration
	
	if config.Cache.UseTestInterval {
		interval = time.Duration(config.Cache.TestIntervalMinutes) * time.Minute
		log.Printf("使用测试间隔: %d分钟", config.Cache.TestIntervalMinutes)
	} else {
		interval = time.Duration(config.Cache.UpdateIntervalMinutes) * time.Minute
		log.Printf("使用生产间隔: %d分钟", config.Cache.UpdateIntervalMinutes)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			updatePortStatusCache()
		}
	}
}

// getPortStatusFromCacheHandler 从Redis缓存获取端口状态
func getPortStatusFromCacheHandler(c *gin.Context) {
	// 从Redis获取缓存的状态
	statusJSON, err := redisClient.Get(ctx, REDIS_KEY_PORT_STATUS).Result()
	if err != nil {
		if err == redis.Nil {
			log.Printf("缓存中没有端口状态，直接查询路由器")
			// 缓存中没有数据，直接查询
			getPortStatusHandler(c)
			return
		}
		log.Printf("从Redis获取状态失败: %v", err)
		c.JSON(500, gin.H{"error": "获取缓存状态失败"})
		return
	}

	// 反序列化状态
	var status map[string]string
	err = json.Unmarshal([]byte(statusJSON), &status)
	if err != nil {
		log.Printf("反序列化状态失败: %v", err)
		c.JSON(500, gin.H{"error": "解析缓存状态失败"})
		return
	}

	// 获取最后更新时间
	lastUpdateStr, err := redisClient.Get(ctx, REDIS_KEY_LAST_UPDATE).Result()
	if err == nil {
		if lastUpdate, err := strconv.ParseInt(lastUpdateStr, 10, 64); err == nil {
			updateTime := time.Unix(lastUpdate, 0)
			log.Printf("返回缓存的端口状态，最后更新时间: %s", updateTime.Format("2006-01-02 15:04:05"))
		}
	}

	c.JSON(200, status)
}

// getCurrentPortStatusDirect 直接从路由器获取端口状态（不使用缓存）
func getCurrentPortStatusDirect() (map[string]string, error) {
	return getCurrentPortStatus()
}