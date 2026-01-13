package router

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"h3c-nat-manager/internal/domain/nat"
)

// H3CClient H3C路由器SSH客户端
type H3CClient struct {
	host       string
	username   string
	password   string
	expiryHour int // 过期小时
	expiryMin  int // 过期分钟
}

// NewH3CClient 创建H3C客户端
func NewH3CClient(host, username, password string) *H3CClient {
	return &H3CClient{
		host:       host,
		username:   username,
		password:   password,
		expiryHour: 21, // 默认21点
		expiryMin:  30, // 默认30分
	}
}

// NewH3CClientWithExpiryTime 创建带过期时间配置的H3C客户端
func NewH3CClientWithExpiryTime(host, username, password string, expiryHour, expiryMin int) *H3CClient {
	return &H3CClient{
		host:       host,
		username:   username,
		password:   password,
		expiryHour: expiryHour,
		expiryMin:  expiryMin,
	}
}

// GetAllEntries 获取所有NAT映射条目
func (c *H3CClient) GetAllEntries() ([]*nat.NATEntry, error) {
	fmt.Printf("正在连接路由器 %s...\n", c.host)
	
	conn, err := c.connect()
	if err != nil {
		return nil, fmt.Errorf("连接路由器失败: %v", err)
	}
	defer conn.Close()

	fmt.Println("SSH连接成功，创建会话...")
	
	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	fmt.Println("执行NAT查询命令...")
	
	// 直接使用Output方法执行命令
	output, err := session.Output("screen-length disable\ndisplay nat server")
	if err != nil {
		return nil, fmt.Errorf("执行命令失败: %v", err)
	}

	fmt.Printf("命令执行成功，输出长度: %d 字节\n", len(output))
	
	return c.parseNATOutput(string(output))
}

// DeleteEntry 删除NAT映射条目
func (c *H3CClient) DeleteEntry(entry *nat.NATEntry) error {
	conn, err := c.connect()
	if err != nil {
		return fmt.Errorf("连接路由器失败: %v", err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	// 构建删除命令
	protocol := strings.ToLower(entry.Protocol)
	deleteCmd := fmt.Sprintf("system-view\ninterface %s\nundo nat server protocol %s global %s %d\nquit\nquit\n",
		entry.Interface, protocol, entry.GlobalIP, entry.GlobalPort)

	// 执行删除命令
	_, err = session.Output(deleteCmd)
	if err != nil {
		return fmt.Errorf("删除NAT条目失败: %v", err)
	}

	return nil
}

// connect 建立SSH连接
func (c *H3CClient) connect() (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: c.username,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	conn, err := ssh.Dial("tcp", c.host+":22", config)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// parseNATOutput 解析NAT命令输出
func (c *H3CClient) parseNATOutput(output string) ([]*nat.NATEntry, error) {
	var entries []*nat.NATEntry
	
	// 按行分割输出
	lines := strings.Split(output, "\n")
	
	var currentEntry *nat.NATEntry
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 匹配接口行
		if strings.HasPrefix(line, "Interface:") {
			if currentEntry != nil {
				// 使用配置的过期时间解析
				currentEntry.ParseExpiryDateWithTime(c.expiryHour, c.expiryMin)
				entries = append(entries, currentEntry)
			}
			
			currentEntry = &nat.NATEntry{}
			currentEntry.Interface = strings.TrimSpace(strings.TrimPrefix(line, "Interface:"))
		}
		
		if currentEntry == nil {
			continue
		}
		
		// 匹配协议行
		if strings.HasPrefix(line, "Protocol:") {
			protocolStr := strings.TrimSpace(strings.TrimPrefix(line, "Protocol:"))
			if strings.Contains(protocolStr, "TCP") {
				currentEntry.Protocol = "TCP"
			} else if strings.Contains(protocolStr, "UDP") {
				currentEntry.Protocol = "UDP"
			}
		}
		
		// 匹配全局IP/端口行
		if strings.HasPrefix(line, "Global IP/port:") {
			globalAddr := strings.TrimSpace(strings.TrimPrefix(line, "Global IP/port:"))
			if err := c.parseAddress(globalAddr, &currentEntry.GlobalIP, &currentEntry.GlobalPort); err != nil {
				continue
			}
		}
		
		// 匹配本地IP/端口行
		if strings.HasPrefix(line, "Local IP/port") {
			localAddr := strings.TrimSpace(strings.Split(line, ":")[1])
			if err := c.parseAddress(localAddr, &currentEntry.LocalIP, &currentEntry.LocalPort); err != nil {
				continue
			}
		}
		
		// 匹配描述行
		if strings.HasPrefix(line, "Description") {
			currentEntry.Description = strings.TrimSpace(strings.Split(line, ":")[1])
		}
		
		// 匹配状态行
		if strings.HasPrefix(line, "Config status") {
			currentEntry.Status = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}
	
	// 处理最后一个条目
	if currentEntry != nil {
		currentEntry.ParseExpiryDateWithTime(c.expiryHour, c.expiryMin)
		entries = append(entries, currentEntry)
	}
	
	return entries, nil
}

// parseAddress 解析IP地址和端口
func (c *H3CClient) parseAddress(addr string, ip *string, port *int) error {
	// 匹配 IP/端口 格式
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