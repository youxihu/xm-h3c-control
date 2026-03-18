# 星目路由器端口快切工具

基于Vue3+Go的H3C路由器端口映射快速切换工具，支持多端口批量操作和实时状态监控。

## 🚀 项目概述

本工具专为H3C路由器端口映射管理而设计，提供直观的Web界面来快速切换端口映射的内网IP地址。支持批量操作、实时状态监控和完整的操作审计功能。

### 核心功能

- **端口映射快速切换**：支持61002、61100、48080三个端口的内网IP切换
- **批量操作优化**：单SSH会话完成多端口切换，避免重复连接开销
- **实时状态监控**：Redis缓存+路由器实时查询的混合策略
- **操作日志记录**：完整的操作轨迹追踪，包含操作者IP、时间戳、操作详情
- **响应式界面**：三层布局设计，状态区-配置区-操作区清晰分离

## 🏗️ 技术架构

### 前端技术栈
- **Vue 3** - 渐进式JavaScript框架
- **Element Plus** - Vue 3组件库
- **Vite** - 现代化构建工具
- **Axios** - HTTP客户端

### 后端技术栈
- **Go 1.21+** - 高性能编程语言
- **Gin** - Web框架
- **SSH** - 路由器连接
- **Redis** - 缓存和日志存储
- **YAML** - 配置文件格式

### 架构设计
采用DDD（领域驱动设计）分层架构：

```
backend/
├── cmd/                    # 应用入口
├── internal/
│   ├── domain/            # 领域层
│   │   ├── entity/        # 实体
│   │   ├── repository/    # 仓储接口
│   │   └── service/       # 领域服务
│   ├── application/       # 应用层
│   │   ├── dto/          # 数据传输对象
│   │   └── service/      # 应用服务
│   ├── infrastructure/    # 基础设施层
│   │   ├── config/       # 配置管理
│   │   ├── repository/   # 仓储实现
│   │   └── utils/        # 工具类
│   └── interface/         # 接口层
│       └── http/         # HTTP处理器
└── config/               # 配置文件
```

## 📦 快速开始

### 环境要求

- **Go 1.21+**
- **Node.js 16+**
- **Redis 6+**
- **H3C路由器** (支持SSH访问)

### 安装部署

1. **克隆项目**
```bash
git clone <repository-url>
cd xm-h3c-control
```

2. **配置后端**
```bash
cd backend
cp config/config.yaml.example config/config.yaml
# 编辑配置文件，填入路由器和Redis连接信息
```

3. **启动后端服务**
```bash
cd backend
go mod tidy
make run
# 或者 go run cmd/main.go
```

4. **配置前端**
```bash
cd frontend
npm install
```

5. **启动前端服务**
```bash
cd frontend
npm run dev
```

6. **访问应用**
```
前端界面: http://localhost:3000
后端API: http://localhost:8080
```

## ⚙️ 配置说明

### 后端配置 (backend/config/config.yaml)

```yaml
# 路由器连接配置
router:
  host: "192.168.1.1"        # 路由器IP地址
  user: "admin"              # SSH用户名
  passwd: "password"         # SSH密码
  external_ip: "112.149.14.2" # 外网IP

# Redis配置
redis:
  host: "192.168.1.218"      # Redis服务器地址
  port: 6379                 # Redis端口
  password: "ningzaichun"    # Redis密码
  db: 0                      # 数据库编号

# 端口映射配置
port_mappings:
  61002: 65530               # 内网端口: 外网端口
  61100: 65531
  48080: 65532

# 环境配置
environment: "dev"           # dev/test/prod
cache_ttl_minutes: 15        # 缓存过期时间(分钟)
```

### 端口配置说明

系统支持三个预定义端口的映射切换：

| 内网端口 | 外网端口 | 服务描述 | 支持的内网IP |
|---------|---------|----------|-------------|
| 61002 | 65530 | 侦测无人机控制链路端口 | 192.168.1.211, 192.168.1.221 |
| 61100 | 65531 | 侦测无人机数据链路端口 | 192.168.1.211, 192.168.1.221 |
| 48080 | 65532 | 东吴后端端口 | 192.168.1.218 |

## 🎨 界面预览

### 主界面布局
- **状态区**：显示当前端口映射状态，实时更新
- **配置区**：端口映射配置卡片，支持IP选择
- **操作区**：保存按钮和操作日志面板

### 特色功能
- **可折叠日志面板**：默认收起，点击展开查看详细日志
- **实时状态指示**：绿色表示已同步，黄色表示待保存
- **批量操作支持**：一次性切换多个端口映射
- **操作审计**：记录每次操作的详细信息

## 🔧 API接口

### 端口状态查询
```http
GET /api/port-status
```

### 端口配置查询
```http
GET /api/port-config
```

### 批量应用配置
```http
POST /api/apply-config
Content-Type: application/json

{
  "configs": [
    {
      "internal_port": 61002,
      "internal_ip": "192.168.1.221"
    }
  ]
}
```

### 操作日志查询
```http
GET /api/operation-logs?limit=10
```

## 🚀 性能优化

### SSH连接优化
- **批量操作**：多个端口切换使用单个SSH会话
- **连接复用**：避免频繁建立/断开连接
- **超时控制**：30秒连接超时，防止长时间等待

### 缓存策略
- **Redis缓存**：端口状态缓存15分钟（生产环境）
- **智能更新**：仅在数据变化时更新缓存
- **缓存穿透保护**：空值缓存防止频繁查询

### 前端优化
- **响应式设计**：适配多种屏幕尺寸
- **组件懒加载**：按需加载减少初始包大小
- **状态管理**：Vue 3 Composition API优化

## 🛠️ 开发指南

### 后端开发

```bash
# 运行测试
cd backend
go test ./...

# 构建
go build -o port-switch-backend cmd/main.go

# 热重载开发
make run
```

### 前端开发

```bash
# 开发模式
cd frontend
npm run dev

# 构建生产版本
npm run build

# 预览构建结果
npm run preview
```

### 代码规范
- **Go**: 遵循Go官方代码规范，使用gofmt格式化
- **Vue**: 遵循Vue 3官方风格指南
- **提交**: 使用Conventional Commits规范

## 📝 更新日志

### v1.0.0 (2026-03-18)
- ✨ 初始版本发布
- 🚀 支持H3C路由器端口映射切换
- 📊 实现操作日志和状态监控
- 🎨 响应式Web界面
- ⚡ 批量操作性能优化

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🆘 故障排除

### 常见问题

**Q: 连接路由器失败**
A: 检查路由器IP、用户名密码是否正确，确保SSH服务已启用

**Q: Redis连接失败**
A: 检查Redis服务是否运行，网络连接是否正常

**Q: 前端无法访问后端**
A: 检查后端服务是否启动，端口8080是否被占用

**Q: 端口切换失败**
A: 检查目标IP是否在配置的允许列表中

### 日志查看

```bash
# 后端日志
cd backend
make run

# 前端开发日志
cd frontend
npm run dev
```

## 📞 技术支持

如有问题或建议，请通过以下方式联系：

- 提交 Issue: [GitHub Issues](https://github.com/your-repo/issues)
- 邮箱: your-email@example.com

---

**星目路由器端口快切工具** - 让端口映射管理更简单高效 🚀