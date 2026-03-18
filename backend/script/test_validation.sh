#!/bin/bash

# 测试端口映射校验功能
# 使用前请先启动服务器: make run

BASE_URL="http://localhost:8080"

echo "=== 端口映射校验测试 ==="
echo

# 1. 测试合法的端口切换请求
echo "1. 测试合法请求 - 端口61002切换到192.168.1.211"
curl -X POST "${BASE_URL}/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.221",
    "new_internal_ip": "192.168.1.211", 
    "internal_port": 61002
  }' | jq .
echo -e "\n"

# 2. 测试非法端口
echo "2. 测试非法端口 - 端口8080（不在配置中）"
curl -X POST "${BASE_URL}/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.211",
    "new_internal_ip": "192.168.1.221",
    "internal_port": 8080
  }' | jq .
echo -e "\n"

# 3. 测试非法IP
echo "3. 测试非法IP - 192.168.1.100（不在配置中）"
curl -X POST "${BASE_URL}/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.211",
    "new_internal_ip": "192.168.1.100",
    "internal_port": 61002
  }' | jq .
echo -e "\n"

# 4. 测试IP与端口不匹配
echo "4. 测试IP与端口不匹配 - 192.168.1.221不支持48080端口"
curl -X POST "${BASE_URL}/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.211",
    "new_internal_ip": "192.168.1.221",
    "internal_port": 48080
  }' | jq .
echo -e "\n"

# 5. 测试批量配置 - 包含合法和非法请求
echo "5. 测试批量配置 - 混合合法和非法请求"
curl -X POST "${BASE_URL}/api/apply-config" \
  -H "Content-Type: application/json" \
  -d '{
    "configs": [
      {
        "internal_port": 61002,
        "internal_ip": "192.168.1.211"
      },
      {
        "internal_port": 8080,
        "internal_ip": "192.168.1.211"
      },
      {
        "internal_port": 48080,
        "internal_ip": "192.168.1.221"
      }
    ]
  }' | jq .
echo -e "\n"

# 6. 测试获取端口配置
echo "6. 获取端口配置信息"
curl -X GET "${BASE_URL}/api/port-config" | jq .
echo -e "\n"

# 7. 测试获取端口状态
echo "7. 获取当前端口状态"
curl -X GET "${BASE_URL}/api/port-status" | jq .
echo -e "\n"

echo "=== 测试完成 ==="