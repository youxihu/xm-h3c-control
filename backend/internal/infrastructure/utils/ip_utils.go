package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

// GetClientIP 获取客户端IP（优先获取真实外网IP）
// GetClientExternalIP 获取客户端外网IP（从请求头获取）
// GetClientExternalIP 获取客户端外网IP（直接调用第三方服务）
// GetClientExternalIP 获取客户端外网IP（从请求头获取，由前端提供）
// GetClientIP 获取客户端IP
func GetClientIP(r *http.Request) string {
	// 1. 检查 X-Forwarded-For 头
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" && net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// 2. 检查 X-Real-IP 头
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" && net.ParseIP(xRealIP) != nil {
		return xRealIP
	}

	// 3. 使用 RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// GetClientInternalIP 获取客户端内网IP（从请求头获取）
func GetClientInternalIP(r *http.Request) string {
	// 检查自定义头 X-Client-Internal-IP（前端发送的客户端内网IP）
	clientInternalIP := r.Header.Get("X-Client-Internal-IP")
	if clientInternalIP != "" {
		return clientInternalIP
	}

	// 如果没有自定义头，尝试从其他头获取
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		// 查找私有IP
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" && isPrivateIP(ip) {
				return ip
			}
		}
	}

	// 最后使用RemoteAddr（如果是私有IP）
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	
	if isPrivateIP(ip) {
		return ip
	}
	
	return "unknown"
}

// GetServerExternalIP 获取服务器外网IP
func GetServerExternalIP() string {
	// 尝试多个服务获取外网IP
	services := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ipecho.net/plain",
	}

	for _, service := range services {
		if ip := getIPFromService(service); ip != "" {
			return ip
		}
	}

	return "unknown"
}

// getIPFromService 从指定服务获取IP
func getIPFromService(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) != nil {
		return ip
	}

	return ""
}

// isPrivateIP 检查是否为私有IP
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 检查是否为私有网络地址
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// GetLocalIP 获取本机内网IP
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "unknown"
}

// FormatIPInfo 格式化IP信息
func FormatIPInfo(clientIP, serverExternalIP string) string {
	return fmt.Sprintf("客户端:%s 服务器:%s", clientIP, serverExternalIP)
}