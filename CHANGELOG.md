# CHANGELOG

## [v0.0.6] - 2026-05-19

### 修复
- **端口映射配置不一致**：`revert.yaml` 中 `port_mappings` 的 key `7109` 与 `port_descriptions` 的 key `1883` 不匹配，导致前端缺少 36409（侦测mqtt）端口。统一为 `1883`。
- **前端 options 为 null 崩溃**：`/api/port-config` 返回的 1883 端口 `options` 为 `null`，前端 `[...null]` 展开抛出 TypeError，页面无法渲染。
  - 前端：`[...port.options]` 加 `|| []` 防御
  - 后端：`options` 用 `make` 初始化，确保 JSON 输出 `[]` 而非 `null`
- **Docker 构建失败**：Makefile 中 `$(cat VERSION)` 未正确执行 shell 命令，改为 `$(shell cat VERSION)`。

### 变更
- **操作日志字段优化**：`operation` 字段从硬编码 `"端口切换"` 改为动态读取配置文件中 `mappings.description`，使日志能区分具体操作项。
  - `config.go`：新增 `PortMappingDetail` 结构体，`PortDescription` 增加 `Mappings` 字段
  - `cmd/main.go`：构造 `portDescriptions` 时传递 Mappings 数据
  - `port_application_service.go`：新增 `GetMappingDescription()`、`GetDefaultMappingDescription()` 方法
  - `port_handler.go`：所有日志调用改用动态描述
  - `OperationLogs.vue`：标签 `操作类型` → `操作项`

## [v0.0.5] - 2026-05-13

### 新功能
- **一个内网端口映射多个外网端口**：`port_mappings` 的值从 `int` 改为 `[]int`，如 61002 可同时映射到 36401 和 36405，前端自动展开多端口卡片并编号区分。
- **external_port 校验**：`apply-config` 接口增加外网端口校验，只允许配置文件中定义的外网端口，校验不合法时返回错误并列出允许范围。
- **revert.yaml 配置扩展**：新增 `hosts` 和 `port_descriptions` 配置，支持多环境 IP 动态配置。
- **前端工程化拆分**：将单文件组件拆分为 `api/`、`composables/`、`views/` 分层架构，提升代码可维护性。
- **端口配置三级优先级**：环境变量 `PORT` > 配置文件 `server.port` > 默认值 8080，适配 Docker/K8s 部署。

### 重构
- 删除未使用的函数：`createNewMapping`、`UpdateInternalTarget`、`HasOption`、`TimeAgo`、`isStatusEqual`、`ApplyBatchConfig`、`switchNATMapping`、`min`
- 删除未使用的接口方法及实现：`FindByInternalTarget`、`FindByExternalPort`、`UpdateMapping`、`InvalidateCache`
- 清理未使用的 import（`nat_mapping.go` 中的 `time`）
- `validatePortConfig` 新增 `externalPort` 参数校验
- 移除 `node_modules` 版本跟踪，更新 README

## [v0.0.4] - 2026-04-24

### 变更
- **动态 IP 配置**：前端 IP 选项从配置文件中动态读取，按 IP 地址排序显示，不再硬编码。
- **新增杭实环境（zc-hangshi-tmp）**：为 61002/61100 端口添加杭实临时环境支持。
- 修正 README 端口数量描述，补充新 IP 配置说明。
- 简单 README 标识修复。

## [v0.0.3] - 2026-03-27

### 新功能
- **DDD 分层架构**：后端重构为 Domain/Application/Infrastructure/Interface 四层，职责清晰，依赖单向传递。
- **zg-test 环境支持**：天波珞光设备现支持 dev/zc-test/zg-test 三个环境切换。
- 完善项目构建和部署配置（Makefile、Dockerfile），添加项目 README 文档。

### 前端优化
- 端口配置卡片设置固定高度 280px，解决高度不一致问题。
- 端口选项添加内部滚动，避免影响整体布局。
- 修复日志组件，现在显示所有日志记录（不再限制条数）。
- 优化卡片内容布局，使用 flex 确保对齐。
- 修复端口配置环境顺序和描述显示。

### Bug 修复
- 修复 `BatchSwitchMappings` 中的空指针引用问题。
- 修复无旧映射时新增操作的空指针异常，正确处理新增场景。
- 修复日志显示限制，用户可查看完整操作历史。

### 技术改进
- 优化 NAT 映射描述格式：`时间戳 IP切换信息`（如 `2026-03-27-14:30 192.168.1.1 switch to 192.168.1.2`）。
- 完善错误处理和日志输出。
- 优化批量操作的描述信息生成。

## [v0.0.2] - 2026-03-25

### 前端优化
- 修复布局对齐问题，确保状态区和配置区宽度一致。
- 优化日志模块展示，修复折叠状态尺寸和样式问题。
- 调整端口状态显示为单行布局，提升视觉效果。
- 为天波珞光设备添加默认选中逻辑。
- 移除多余的白色背景，统一视觉风格。

### 后端重构
- 端口描述从硬编码改为配置文件定义。
- 新增 `port_descriptions` 配置段，支持灵活的端口管理。
- 重构 `GetPortConfig` 方法，从配置文件读取端口信息。
- 优化端口顺序配置，提升代码维护性。
- 为天波珞光设备添加默认 IP 映射逻辑。

### 技术改进
- 配置结构体增强，支持端口描述配置。
- 应用服务重构，提升配置化程度。
- 前端布局算法优化，确保响应式适配。

## [v0.0.1] - 2026-03-18

### 项目概述
基于 Vue3 + Go 的 H3C 路由器端口映射快速切换工具，支持多端口批量操作和实时状态监控。

### 核心功能
- 端口映射快速切换：支持 61002、61100、48080 三个端口的内网 IP 切换
- 批量操作优化：单 SSH 会话完成多端口切换，避免重复连接
- 实时状态监控：Redis 缓存 + 路由器实时查询的混合策略
- 操作日志记录：完整的操作轨迹追踪，包含操作者 IP、时间戳、操作详情
- 响应式界面：三层布局设计，状态区-配置区-操作区清晰分离

### 技术架构
- 前端：Vue3 + Element Plus + Vite
- 后端：Go + Gin + DDD 架构 + SSH 客户端
- 缓存：Redis (操作日志 + 状态缓存)
- 部署：前后端分离，支持内网环境

### DDD 分层架构
- Domain 层：实体、仓储接口、领域服务
- Application 层：应用服务、DTO、业务编排
- Infrastructure 层：SSH 实现、Redis 实现、配置管理
- Interface 层：HTTP 路由、处理器、中间件

### 关键优化
- SSH 连接池化：批量操作使用单会话
- 缓存策略：15 分钟生产 / 1 分钟测试的智能缓存
- 错误处理：完整的校验链和错误恢复机制
- 日志系统：操作审计和性能监控

### UI/UX 特性
- 紫色渐变主题，现代化卡片设计
- 可折叠日志面板，节省屏幕空间
- 实时状态指示器，直观的操作反馈
- 响应式布局，支持多设备访问

### 配置管理
- 动态端口配置：支持多环境 (dev/zc/dw)
- 路由器连接配置：SSH 认证和网络参数
- Redis 配置：缓存策略和连接参数
- 安全校验：仅允许配置文件定义的端口和 IP 操作