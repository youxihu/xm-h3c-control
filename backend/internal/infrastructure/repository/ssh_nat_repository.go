package repository

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"port-switch-backend/internal/domain/entity"
	"port-switch-backend/internal/domain/repository"
	"port-switch-backend/internal/infrastructure/config"
)

// SSHNATRepository SSH NAT仓储实现
type SSHNATRepository struct {
	routerConfig *config.RouterConfig
}

// NewSSHNATRepository 创建SSH NAT仓储
func NewSSHNATRepository(cfg *config.RouterConfig) *SSHNATRepository {
	return &SSHNATRepository{
		routerConfig: cfg,
	}
}

// GetAllMappings 获取所有NAT映射
func (r *SSHNATRepository) GetAllMappings() ([]*entity.NATMapping, error) {
	client, err := r.connect()
	if err != nil {
		return nil, fmt.Errorf("连接路由器失败: %v", err)
	}
	defer func() {
		client.Close()
		log.Printf("SSH连接已关闭")
	}()

	return r.getCurrentNATMappings(client)
}

// FindByInternalTarget 根据内网目标查找映射
func (r *SSHNATRepository) FindByInternalTarget(internalIP string, internalPort int) (*entity.NATMapping, error) {
	mappings, err := r.GetAllMappings()
	if err != nil {
		return nil, err
	}

	for _, mapping := range mappings {
		if mapping.IsSameTarget(internalIP, internalPort) {
			return mapping, nil
		}
	}

	return nil, fmt.Errorf("未找到内网目标 %s:%d 的映射", internalIP, internalPort)
}

// FindByExternalPort 根据外网端口查找映射
func (r *SSHNATRepository) FindByExternalPort(externalPort int) (*entity.NATMapping, error) {
	mappings, err := r.GetAllMappings()
	if err != nil {
		return nil, err
	}

	for _, mapping := range mappings {
		if mapping.ExternalPort() == externalPort {
			return mapping, nil
		}
	}

	return nil, fmt.Errorf("未找到外网端口 %d 的映射", externalPort)
}

// CreateMapping 创建新映射
func (r *SSHNATRepository) CreateMapping(mapping *entity.NATMapping) error {
	client, err := r.connect()
	if err != nil {
		return fmt.Errorf("连接路由器失败: %v", err)
	}
	defer func() {
		client.Close()
		log.Printf("SSH连接已关闭")
	}()

	return r.createNATMapping(client, mapping)
}

// UpdateMapping 更新映射（删除旧的，创建新的）
func (r *SSHNATRepository) UpdateMapping(oldMapping, newMapping *entity.NATMapping) error {
	client, err := r.connect()
	if err != nil {
		return fmt.Errorf("连接路由器失败: %v", err)
	}
	defer func() {
		client.Close()
		log.Printf("SSH连接已关闭")
	}()

	return r.switchNATMapping(client, oldMapping, newMapping)
}

// DeleteMapping 删除映射
func (r *SSHNATRepository) DeleteMapping(mapping *entity.NATMapping) error {
	client, err := r.connect()
	if err != nil {
		return fmt.Errorf("连接路由器失败: %v", err)
	}
	defer func() {
		client.Close()
		log.Printf("SSH连接已关闭")
	}()

	return r.deleteNATMapping(client, mapping)
}

// connect 连接路由器
func (r *SSHNATRepository) connect() (*ssh.Client, error) {
	log.Printf("正在连接路由器 %s，用户: %s", r.routerConfig.Host, r.routerConfig.User)

	sshConfig := &ssh.ClientConfig{
		User: r.routerConfig.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(r.routerConfig.Passwd),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	address := fmt.Sprintf("%s:22", r.routerConfig.Host)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		log.Printf("SSH连接失败: %v", err)
		return nil, err
	}

	log.Printf("SSH连接成功，连接地址: %s", address)
	return client, nil
}

// executeCommand 执行SSH命令
func (r *SSHNATRepository) executeCommand(client *ssh.Client, command string) (string, error) {
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

// getCurrentNATMappings 获取当前NAT映射
func (r *SSHNATRepository) getCurrentNATMappings(client *ssh.Client) ([]*entity.NATMapping, error) {
	log.Printf("开始获取NAT映射配置")

	output, err := r.executeCommand(client, "screen-length disable\ndisplay nat server")
	if err != nil {
		log.Printf("获取NAT配置失败: %v", err)
		return nil, fmt.Errorf("获取NAT配置失败: %v", err)
	}

	log.Printf("NAT配置输出长度: %d 字节", len(output))
	//if len(output) > 0 {
	//	log.Printf("NAT配置输出前500字符: %s", output[:min(500, len(output))])
	//}

	mappings := r.parseNATOutput(output)
	log.Printf("解析到 %d 个NAT映射", len(mappings))

	//for _, mapping := range mappings {
	//	log.Printf("映射: %s", mapping.String())
	//}

	return mappings, nil
}

// createNATMapping 创建NAT映射
func (r *SSHNATRepository) createNATMapping(client *ssh.Client, mapping *entity.NATMapping) error {
	log.Printf("在单个会话中创建映射: %s", mapping.String())

	createCmd := fmt.Sprintf(`system-view
interface GigabitEthernet0/0
nat server protocol %s global %s %d inside %s %d description %s
quit
quit`,
		mapping.Protocol(), mapping.ExternalIP(), mapping.ExternalPort(),
		mapping.InternalIP(), mapping.InternalPort(), mapping.Description(),
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

// switchNATMapping 切换NAT映射
func (r *SSHNATRepository) switchNATMapping(client *ssh.Client, oldMapping, newMapping *entity.NATMapping) error {
	log.Printf("=== 开始端口切换操作 ===")
	log.Printf("删除旧映射: %s", oldMapping.String())
	log.Printf("创建新映射: %s", newMapping.String())

	switchCmd := fmt.Sprintf(`system-view
interface GigabitEthernet0/0
undo nat server protocol %s global %s %d
nat server protocol %s global %s %d inside %s %d description %s
quit
quit`,
		oldMapping.Protocol(), oldMapping.ExternalIP(), oldMapping.ExternalPort(),
		newMapping.Protocol(), newMapping.ExternalIP(), newMapping.ExternalPort(),
		newMapping.InternalIP(), newMapping.InternalPort(), newMapping.Description(),
	)

	log.Printf("执行切换命令序列:")
	log.Printf("1. system-view")
	log.Printf("2. interface GigabitEthernet0/0")
	log.Printf("3. undo nat server protocol %s global %s %d (删除旧映射)",
		oldMapping.Protocol(), oldMapping.ExternalIP(), oldMapping.ExternalPort())
	log.Printf("4. nat server protocol %s global %s %d inside %s %d description %s (创建新映射)",
		newMapping.Protocol(), newMapping.ExternalIP(), newMapping.ExternalPort(),
		newMapping.InternalIP(), newMapping.InternalPort(), newMapping.Description())
	log.Printf("5. quit")
	log.Printf("6. quit")

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(switchCmd)
	if err != nil {
		log.Printf("❌ 切换命令执行失败: %v", err)
		log.Printf("命令输出: %s", string(output))
		return fmt.Errorf("切换命令执行失败: %v", err)
	}

	log.Printf("✅ 切换命令执行成功")
	log.Printf("命令输出: %s", string(output))
	log.Printf("=== 端口切换操作完成 ===")
	return nil
}

// deleteNATMapping 删除NAT映射
func (r *SSHNATRepository) deleteNATMapping(client *ssh.Client, mapping *entity.NATMapping) error {
	deleteCmd := fmt.Sprintf(`system-view
interface GigabitEthernet0/0
undo nat server protocol %s global %s %d
quit
quit`,
		mapping.Protocol(), mapping.ExternalIP(), mapping.ExternalPort(),
	)

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(deleteCmd)
	if err != nil {
		log.Printf("删除命令执行失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("删除命令执行失败: %v", err)
	}

	log.Printf("删除命令执行成功，输出: %s", string(output))
	return nil
}

// parseNATOutput 解析NAT输出
func (r *SSHNATRepository) parseNATOutput(output string) []*entity.NATMapping {
	var mappings []*entity.NATMapping
	lines := strings.Split(output, "\n")

	var protocol, externalIP, internalIP, description string
	var externalPort, internalPort int

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 匹配协议行
		if strings.HasPrefix(line, "Protocol:") {
			protocolStr := strings.TrimSpace(strings.TrimPrefix(line, "Protocol:"))
			if strings.Contains(protocolStr, "TCP") {
				protocol = "tcp"
			} else if strings.Contains(protocolStr, "UDP") {
				protocol = "udp"
			}
		}

		// 匹配全局IP/端口行
		if strings.HasPrefix(line, "Global IP/port:") {
			globalAddr := strings.TrimSpace(strings.TrimPrefix(line, "Global IP/port:"))
			r.parseAddress(globalAddr, &externalIP, &externalPort)
		}

		// 匹配本地IP/端口行
		if strings.HasPrefix(line, "Local IP/port") {
			localAddr := strings.TrimSpace(strings.Split(line, ":")[1])
			r.parseAddress(localAddr, &internalIP, &internalPort)
		}

		// 匹配描述行
		if strings.HasPrefix(line, "Description") {
			description = strings.TrimSpace(strings.Split(line, ":")[1])

			// 创建映射实体
			if protocol != "" && externalIP != "" && internalIP != "" {
				mapping, err := entity.NewNATMapping(
					protocol, externalIP, externalPort,
					internalIP, internalPort, description,
				)
				if err == nil {
					mappings = append(mappings, mapping)
				}
			}

			// 重置变量
			protocol, externalIP, internalIP, description = "", "", "", ""
			externalPort, internalPort = 0, 0
		}
	}

	return mappings
}

// parseAddress 解析IP地址和端口
func (r *SSHNATRepository) parseAddress(addr string, ip *string, port *int) error {
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

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BatchSwitchMappings 批量切换映射（在单个SSH会话中完成）
func (r *SSHNATRepository) BatchSwitchMappings(operations []repository.SwitchOperation) error {
	if len(operations) == 0 {
		return nil
	}

	client, err := r.connect()
	if err != nil {
		return fmt.Errorf("连接路由器失败: %v", err)
	}
	defer func() {
		client.Close()
		log.Printf("SSH连接已关闭")
	}()

	log.Printf("=== 开始批量端口切换操作，共 %d 个操作 ===", len(operations))

	// 构建批量命令
	var commands []string
	commands = append(commands, "system-view")
	commands = append(commands, "interface GigabitEthernet0/0")

	for i, op := range operations {
		var oldMappingStr, newMappingStr string
		
		if op.OldMapping != nil {
			oldMappingStr = op.OldMapping.String()
		} else {
			oldMappingStr = "无旧映射"
		}
		
		if op.NewMapping != nil {
			newMappingStr = op.NewMapping.String()
		} else {
			newMappingStr = "无新映射"
		}
		
		log.Printf("操作 %d: 删除 %s -> 创建 %s", i+1, oldMappingStr, newMappingStr)

		// 如果有旧映射，先删除
		if op.OldMapping != nil {
			deleteCmd := fmt.Sprintf("undo nat server protocol %s global %s %d",
				op.OldMapping.Protocol(), op.OldMapping.ExternalIP(), op.OldMapping.ExternalPort())
			commands = append(commands, deleteCmd)
		}

		// 创建新映射
		if op.NewMapping != nil {
			createCmd := fmt.Sprintf("nat server protocol %s global %s %d inside %s %d description %s",
				op.NewMapping.Protocol(), op.NewMapping.ExternalIP(), op.NewMapping.ExternalPort(),
				op.NewMapping.InternalIP(), op.NewMapping.InternalPort(), op.NewMapping.Description())
			commands = append(commands, createCmd)
		}
	}

	commands = append(commands, "quit")
	commands = append(commands, "quit")

	// 执行批量命令
	batchCmd := strings.Join(commands, "\n")
	log.Printf("执行批量命令序列:")
	for i, cmd := range commands {
		log.Printf("%d. %s", i+1, cmd)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(batchCmd)
	if err != nil {
		log.Printf("❌ 批量命令执行失败: %v", err)
		log.Printf("命令输出: %s", string(output))
		return fmt.Errorf("批量命令执行失败: %v", err)
	}

	log.Printf("✅ 批量命令执行成功")
	log.Printf("命令输出: %s", string(output))
	log.Printf("=== 批量端口切换操作完成 ===")
	return nil
}
