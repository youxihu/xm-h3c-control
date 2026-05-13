# 星目路由器端口快切工具

基于 Vue3 + Go 的路由器端口映射管理工具，采用 DDD 分层架构。

## 项目能做什么

- 通过 Web 界面快速切换路由器端口映射的内网目标IP
- 支持一个内网端口映射到多个外网端口
- 批量操作在单次 SSH 会话中完成，减少连接开销
- Redis 缓存端口状态，失效时自动从路由器获取并更新
- 操作日志审计，记录操作者、时间和详情
- 校验目标 IP 和端口是否在配置允许范围内，防止误操作
- 多环境 IP 切换支持（配置文件动态定义）
- 端口优先级：环境变量 PORT > 配置文件 server.port > 默认 8080

## 当前实现

### 端口对应映射
内网端口与外网端口一一对应，映射关系由配置文件定义，切换操作仅变更映射指向的内网目标 IP，不改变端口对应关系本身。所有操作经配置校验后才下发到路由器，确保只有预定义的 IP 和端口组合能被写入设备。

### 多外网端口映射
一个内网端口可同时映射到多个外网端口，映射关系通过 `revert.yaml` 的 `port_mappings` 字段定义，新增端口或修改映射只需编辑配置文件重启服务即可生效，无需改动代码。

### 配置驱动架构
所有业务参数（主机列表、端口描述、映射规则、环境标签）均由 YAML 配置文件定义，支持按需增删主机、扩展端口映射、调整缓存策略，配置变更后自动校验生效。

### 智能缓存机制
Redis 缓存端口映射状态，服务启动时预热、定时刷新、切换后即时更新；缓存失效时自动回源查询路由器并补写缓存，前端请求始终优先读取缓存，减少对路由器的 SSH 访问频率。

### 操作审计与日志
每次端口切换操作自动记录操作者来源 IP、切换详情、执行结果和时间戳，日志存储于 Redis List，支持保留上限与过期清理，便于事后追溯和合规审计。

### 部署灵活性
服务端口支持配置文件指定、环境变量覆盖、默认值兜底三级优先级，适配本地开发、测试环境与 Docker/K8s 容器化部署的不同端口需求。

### DDD 分层架构
后端严格遵循领域驱动设计分层：Domain 层承载核心业务规则与实体定义，Application 层编排业务流程与 DTO 转换，Infrastructure 层实现 SSH/Redis 等技术细节，Interface 层暴露 HTTP API。各层职责清晰，依赖单向传递，便于独立演进和替换实现。

## 当前映射条目

外网IP `117.149.14.2` 上的端口映射现状：

| 外网端口 | 内网端口 | 当前指向 | 可切换环境 | 服务描述 |
|---------|---------|---------|----------|---------|
| 36401, 36405 | 61002 | zc-hangshi-tmp (192.168.1.231) | dev / zc-test / zc-hangshi-tmp | 侦测无人机控制链路 |
| 36402, 36406 | 61100 | dev (192.168.1.218) | dev / zc-test / zc-hangshi-tmp | 侦测无人机数据链路 |
| 36404, 36407 | 62201 | zg-test (192.168.1.230) | dev / zc-test / zg-test | 天波珞光设备 |
| 36403 | 48080 | dev (192.168.1.218) | dev / dw-test | 东吴后端端口 |

> 映射条目由 `revert.yaml` 配置文件定义，增删端口或环境只需修改配置文件重启服务。

## 使用示例

### 批量配置应用（前端当前使用）
一次性切换多个端口到指定环境：
```bash
curl -X POST http://localhost:8080/api/apply-config \
  -H "Content-Type: application/json" \
  -d '{"configs":[
    {"internal_port":61002,"internal_ip":"192.168.1.228","external_port":36401},
    {"internal_port":61100,"internal_ip":"192.168.1.228","external_port":36402}
  ]}'
```

### 校验拦截
请求不在配置范围内的端口或IP时，服务会拒绝并返回允许值提示：
```bash
curl -X POST http://localhost:8080/api/apply-config \
  -H "Content-Type: application/json" \
  -d '{"configs":[{"internal_port":61002,"internal_ip":"192.168.1.228","external_port":32401}]}'
# 返回: "外网端口 32401 不属于内网端口 61002 的映射范围，允许的外网端口: [36401, 36405]"
```

### 端口状态查询
```bash
curl http://localhost:8080/api/port-status
```

### 操作日志追溯
```bash
curl http://localhost:8080/api/operation-logs
```

### Docker Compose 部署
```yaml
# docker/docker-compose.yaml
services:
  port-switch:
    image: registry.cn-hangzhou.xingmukeji.com/tools/xm-switch-port:v0.0.5
    container_name: xm-port-switch
    ports:
      - "25003:25003"
    volumes:
      - ./backend/config:/app-acc/config:ro
    environment:
      - TZ=Asia/Shanghai
      - PORT=25003
    restart: always
```

```bash
# 构建并部署
make build-backend    # 构建后端可执行文件
make build-docker     # 构建Docker镜像
cd docker && docker compose up -d
```

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3, Element Plus, Vite, Axios |
| 后端 | Go, Gin, golang.org/x/crypto/ssh, go-redis/v8, gopkg.in/yaml.v3 |
| 存储 | Redis（缓存 + 日志） |

## 架构

### 整体架构

```
前端 (Vue3)  ──HTTP──>  后端 (Go/Gin)  ──SSH──>  H3C路由器
                         │
                         └─Redis──> 缓存 + 日志
```

### 后端分层架构 (DDD)

```
backend/
├── cmd/
│   └── main.go                             # 入口：依赖注入、启动调度
├── internal/
│   ├── domain/                             # 领域层 — 核心业务规则
│   │   ├── entity/
│   │   │   ├── nat_mapping.go              # NAT映射实体（端口、IP、协议、描述）
│   │   │   ├── port_config.go              # 端口配置实体（选项、描述）
│   │   │   └── operation_log.go            # 操作日志实体（操作者、时间、详情）
│   │   ├── repository/
│   │   │   ├── nat_repository.go           # NAT仓储接口（查询、创建、删除、批量切换）
│   │   │   ├── cache_repository.go         # 缓存仓储接口（读写端口状态）
│   │   │   └── operation_log_repository.go # 日志仓储接口（保存、查询、清理）
│   │   └── service/
│   │       └── nat_service.go              # NAT领域服务（缓存更新、批量切换编排）
│   ├── application/                        # 应用层 — 业务流程编排
│   │   ├── dto/
│   │   │   ├── port_dto.go                 # 端口请求/响应DTO
│   │   │   └── operation_dto.go            # 操作日志DTO
│   │   └── service/
│   │       └── port_application_service.go # 端口应用服务（校验、切换、缓存管理）
│   ├── infrastructure/                     # 基础设施层 — 技术实现
│   │   ├── config/
│   │   │   └── config.go                   # 配置加载（YAML解析、结构定义）
│   │   ├── repository/
│   │   │   ├── ssh_nat_repository.go       # SSH仓储实现（连接路由器、执行NAT命令）
│   │   │   ├── redis_cache_repository.go   # Redis缓存实现（端口状态读写、过期管理）
│   │   │   └── redis_operation_log_repository.go # Redis日志实现（保存、查询、清理）
│   │   └── utils/
│   │       └── ip_utils.go                 # IP工具函数
│   └── interface/                          # 接口层 — HTTP API
│       └── http/
│           ├── handler/
│           │   └── port_handler.go          # 端口处理器（状态查询、切换、日志）
│           └── router/
│               └── router.go                # 路由配置（API路由、健康检查）
├── config/
│   ├── config.yaml                          # 主配置（路由器SSH、Redis、缓存、服务端口）
│   └── revert.yaml                          # 映射配置（端口映射、描述、主机列表）
├── docker/
│   ├── Dockerfile                           # 镜像构建（Alpine + 时区 + 可执行文件）
│   └── docker-compose.yaml                  # 部署编排（端口、挂载、环境变量）
└── script/
    └── main.go                              # 旧版独立脚本（参考）
```

### 依赖方向

```
Interface → Application → Domain ← Infrastructure
```

- Domain 层不依赖任何外层，只定义接口和业务规则
- Infrastructure 层实现 Domain 层定义的仓储接口（依赖倒置）
- Application 层编排 Domain 服务，协调业务流程
- Interface 层调用 Application 服务，暴露 HTTP API

### 前端分层架构

```
frontend/src/
├── api/
│   └── index.js                 # API层：axios实例、客户端IP识别、HTTP请求函数
├── composables/
│   ├── useTheme.js              # 主题切换（亮色/暗色，localStorage持久化）
│   └── usePortConfig.js         # 端口配置逻辑（fetch、save、变更检测、展开折叠）
├── components/
│   └── OperationLogs.vue        # 操作日志面板组件
├── router/
│   └── index.js                 # vue-router 路由配置
├── views/
│   └── PortManagement.vue       # 端口管理页面（主界面）
├── App.vue                      # 根组件（仅挂载 router-view）
└── main.js                      # 入口（Vue实例、ElementPlus、Router注册）
```

- API 层封装所有 HTTP 通信，自动识别开发/生产环境后端地址
- Composables 层按业务逻辑拆分，每个 composable 封装一类可复用逻辑
- Views 层对应路由页面，新增页面只需在 `views/` 和 `router/` 中添加
- Components 层是独立 UI 组件，可跨页面复用

## 快速开始

```bash
# 后端（通过Makefile启动）
make run-backend              # 默认 8080
make run-backend   # 环境变量指定端口

# 前端
make run-frontend
```

## 配置说明

所有端口映射、主机环境、端口描述均通过 YAML 配置文件定义，无需改代码即可扩展：

- `config.yaml` — 路由器 SSH、Redis、缓存策略、服务端口
- `revert.yaml` — 端口映射规则、主机列表、端口描述

配置加载后自动校验，不在配置范围内的 IP / 端口请求将被拒绝。

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/port-status | 获取端口状态（缓存优先） |
| GET | /api/port-config | 获取端口配置与可切换选项 |
| POST | /api/apply-config | 批量应用配置（主用） |
| POST | /api/switch-port | 单端口切换（备用） |
| GET | /api/operation-logs | 操作日志 |
| GET | /health | 健康检查 |

## 业务流程

**切换流程**：校验合法性 → 获取当前状态 → SSH 执行映射变更 → 更新缓存 → 记录日志

**缓存流程**：启动预热 → 定时更新 → 切换后立即刷新 → 失效时自动从路由器获取