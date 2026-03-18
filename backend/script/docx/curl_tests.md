# 端口映射校验测试命令

## 启动服务器
```bash
make run
```

## 测试命令

### 1. 合法请求 - 端口61002切换到192.168.1.211
```bash
curl -X POST "http://localhost:8080/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.221",
    "new_internal_ip": "192.168.1.211", 
    "internal_port": 61002
  }'
```

### 2. 非法端口测试 - 端口8080（不在配置中）
```bash
curl -X POST "http://localhost:8080/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.211",
    "new_internal_ip": "192.168.1.221",
    "internal_port": 8080
  }'
```

### 3. 非法IP测试 - 192.168.1.100（不在配置中）
```bash
curl -X POST "http://localhost:8080/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.211",
    "new_internal_ip": "192.168.1.100",
    "internal_port": 61002
  }'
```

### 4. IP与端口不匹配测试 - 192.168.1.221不支持48080端口
```bash
curl -X POST "http://localhost:8080/api/switch-port" \
  -H "Content-Type: application/json" \
  -d '{
    "current_internal_ip": "192.168.1.211",
    "new_internal_ip": "192.168.1.221",
    "internal_port": 48080
  }'
```

### 5. 批量配置测试 - 混合合法和非法请求
```bash
curl -X POST "http://localhost:8080/api/apply-config" \
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
  }'
```

### 6. 获取端口配置
```bash
curl -X GET "http://localhost:8080/api/port-config"
```

### 7. 获取端口状态
```bash
curl -X GET "http://localhost:8080/api/port-status"
```

## 预期结果

### 合法请求应该返回：
```json
{
  "message": "端口切换成功",
  "status": "switched",
  "internal_port": 61002,
  "new_internal_ip": "192.168.1.211"
}
```

### 非法端口应该返回：
```json
{
  "error": "端口 8080 不在允许的配置范围内，允许的端口: [61002 61100 48080]"
}
```

### 非法IP应该返回：
```json
{
  "error": "IP地址 192.168.1.100 不在允许的配置范围内，允许的IP: [192.168.1.211 192.168.1.221 192.168.1.218]"
}
```

### IP与端口不匹配应该返回：
```json
{
  "error": "IP地址 192.168.1.221 不支持端口 48080 的服务，该IP支持的端口: [61002 61100]"
}
```