# XM H3C NAT 映射管理工具

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![Version](https://img.shields.io/badge/version-v0.0.1-green.svg)](./VERSION)

一个专为 XM公司环境下的 H3C MSR2600 路由器设计的 NAT 端口映射自动化管理工具，支持过期提醒和自动清理功能。

## 功能特性

- **自动监控**: 定期检查 NAT 映射条目的过期状态
- **智能通知**: 根据服务器分组发送钉钉过期提醒到不同群组
- **自动清理**: 清理已过期的 NAT 映射条目
- **删除通知**: 删除条目时自动发送钉钉通知
- **中文支持**: 通过描述映射表解决路由器 CLI 中文乱码问题
- **容器化部署**: 支持 Docker 容器化运行
- **DDD 架构**: 采用领域驱动设计，代码结构清晰易维护

## 目录结构

```
xm-h3c-control/
├── cmd/                    # 应用程序入口
│   └── main.go
├── internal/               # 内部代码（DDD架构）
│   ├── application/        # 应用服务层
│   │   └── service/
│   ├── domain/            # 领域层
│   │   ├── nat/           # NAT映射领域
│   │   └── notification/  # 通知领域
│   └── infrastructure/    # 基础设施层
│       ├── config/        # 配置管理
│       ├── description/   # 描述映射
│       ├── notification/  # 钉钉通知
│       └── router/        # H3C路由器客户端
├── config/                # 配置文件
├── docker/                # Docker相关配置
├── bin/                   # 编译输出目录
├── description.yaml       # 中文描述映射表
├── Dockerfile            # Docker镜像构建文件
├── Makefile              # 构建脚本
└── README.md             # 项目说明文档
```

## 安装部署

### 环境要求

- Go 1.23+
- Docker (可选)
- H3C MSR2600 路由器
- 钉钉机器人 Webhook

### 本地编译

```bash
# 克隆项目
git clone <repository-url>
cd xm-h3c-control

# 安装依赖
go mod tidy

# 编译
make build

# 运行
./bin/xm-h3c-control --mode=notify
```

### Docker 部署

```bash
# 构建镜像
make docker

# 运行容器
make docker_run

# 推送镜像
make docker_push
```

## 配置说明

### 主配置文件 (config/config.yaml)

```yaml
# H3C路由器配置
h3c-msr2600:
  host: 192.168.1.1                    # 路由器IP地址
  user: admin                          # SSH用户名
  passwd: password                     # SSH密码
  Reminder_before_expiration: 10       # 过期前提醒天数
  # 过期时间设置 (24小时制)
  expiry_time:
    hour: 21    # 过期小时 (0-23)
    minute: 30  # 过期分钟 (0-59)

# 钉钉通知配置
dingtalk:
  # 默认通知群（兜底）
  default:
    webhook: "https://oapi.dingtalk.com/robot/send?access_token=xxx"
    secret: "SECxxx"
    name: "默认通知群"
  
  # 服务器分组通知配置
  groups:
    # 巡检相关服务器
    inspection:
      webhook: "https://oapi.dingtalk.com/robot/send?access_token=xxx"
      secret: "SECxxx"
      name: "巡检项目组"
      servers:
        - "192.168.1.112"  # 巡检测试演示服务器
        - "192.168.1.150"  # 巡检RTX4090服务器
```

### 描述映射文件 (description.yaml)

用于解决路由器 CLI 返回中文乱码问题：

```yaml
# NAT端口映射描述配置文件
mappings:
  # 外网IP:端口 -> 中文描述
  "117.149.14.2:7935": "巡检测试演示服务器-无人机视频流"
  "117.149.14.2:51280": "巡检测试演示服务器-测试环境Web端"
  # ... 更多映射

# 默认过期时间（天）
default_expiry_days: 365
```

## 使用方法

### 命令行参数

```bash
./xm-h3c-control [选项]

选项:
  --mode string        运行模式: smart(智能处理), notify(仅通知), cleanup(仅清理) (默认 "smart")
  --configs string     配置文件路径 (默认 "configs/config.yaml")
  --desc string        描述映射文件路径 (默认 "description.yaml")
```

### 运行模式

#### 1. 智能处理模式 (smart) - 默认模式
自动根据条目状态决定操作：
- 已过期的条目：发送删除通知后自动删除
- 即将过期的条目：发送钉钉提醒

```bash
./xm-h3c-control
# 或者显式指定
./xm-h3c-control --mode=smart
```

#### 2. 通知模式 (notify)
仅检查即将过期的 NAT 映射条目并发送钉钉提醒：

```bash
./xm-h3c-control --mode=notify
```

#### 3. 清理模式 (cleanup)
仅删除已过期的 NAT 映射条目（删除前会发送通知）：

```bash
./xm-h3c-control --mode=cleanup
```

### 定时任务配置

建议通过 crontab 设置定时任务：

```bash
# 编辑 crontab
crontab -e

# 每天上午9点执行智能处理（推荐）
0 9 * * * /path/to/xm-h3c-control

# 或者分别设置通知和清理任务
# 每天上午9点检查并发送过期提醒
0 9 * * * /path/to/xm-h3c-control --mode=notify

# 每天凌晨2点清理过期条目
0 2 * * * /path/to/xm-h3c-control --mode=cleanup
```

## 工作原理

### NAT 条目过期机制

工具通过解析 NAT 条目描述中的 `vp=YYMMDD` 格式来确定过期时间：

- `vp=260112` 表示 2026年01月12日过期
- 过期时间默认为当天的 21:30:00（可在配置文件中自定义）
- 没有 `vp` 标记的条目默认不过期

### 智能分组通知

根据服务器 IP 地址自动选择对应的钉钉群组：

1. 提取 NAT 条目的本地服务器 IP
2. 匹配配置文件中的服务器分组
3. 发送通知到对应群组
4. 未匹配的服务器使用默认群组
5. 不同群组收到各自服务器的通知，无需在消息中显示群组名称

### 通知消息格式

#### 过期提醒通知
```markdown
## [通知] 端口映射即将过期

**消息来源：** H3c-MSR2600
**外网地址端口：** 117.149.14.2:7935
**内网地址端口：** 192.168.1.112:7935
**协议类型：** TCP
**描述：** 巡检测试演示服务器-无人机视频流
**到期时间：** 2026年01月12日 21:30:00
**通知时间：** 2026年01月10日 09:00:00

---
[查看内外网映射关系表](https://alidocs.dingtalk.com/i/nodes/xxx)
```

#### 删除通知
```markdown
## [通知] 端口映射条目删除

**消息来源：** H3c-MSR2600
**外网地址端口：** 117.149.14.2:7935
**内网地址端口：** 192.168.1.112:7935
**协议类型：** TCP
**描述：** 巡检测试演示服务器-无人机视频流
**到期时间：** 2026年01月12日 21:30:00
**删除时间：** 2026年01月13日 02:00:00

---
[查看内外网映射关系表](https://alidocs.dingtalk.com/i/nodes/xxx)
```

## 架构设计

项目采用 DDD（领域驱动设计）架构：

### 领域层 (Domain)
- **NAT 实体**: 定义 NAT 映射条目的业务逻辑
- **通知实体**: 定义过期通知和删除通知的数据结构和格式化逻辑

### 应用服务层 (Application)
- **NAT 管理服务**: 协调各个领域服务，实现业务流程

### 基础设施层 (Infrastructure)
- **H3C 客户端**: SSH 连接和命令执行
- **钉钉通知服务**: 消息发送和群组路由
- **配置管理**: 配置文件加载和解析
- **描述映射器**: 中文描述映射管理

## 开发指南

### 添加新的服务器分组

1. 在 `config/config.yaml` 中添加新的群组配置
2. 在 `description.yaml` 中添加对应服务器的端口描述
3. 重启服务或重新部署

### 扩展通知渠道

1. 在 `internal/domain/notification/` 中定义新的通知服务接口
2. 在 `internal/infrastructure/notification/` 中实现具体的通知服务
3. 在应用服务层中注入新的通知服务

### 日志示例

#### 智能处理模式
```
2026/01/13 16:02:49 执行智能处理模式...
2026/01/13 16:02:49 开始智能处理NAT映射条目...
正在连接路由器 192.168.1.1...
SSH连接成功，创建会话...
执行NAT查询命令...
命令执行成功，输出长度: 12773 字节
服务器 192.168.1.218 匹配到群组: inspection
发送通知到群组: 巡检项目组 (服务器: 192.168.1.218)
2026/01/13 16:02:57 已发送过期提醒 [117.149.14.2:21]
2026/01/13 16:02:57 已删除过期条目 [117.149.14.2:8080]
2026/01/13 16:02:57 智能处理完成，共发送 1 条过期提醒，删除 1 个过期条目
2026/01/13 16:02:57 程序执行完成
```

## 故障排除

### 常见问题

1. **SSH 连接失败**
   - 检查路由器 IP 地址和端口
   - 验证用户名和密码
   - 确认网络连通性

2. **钉钉通知发送失败**
   - 检查 webhook 地址和 secret
   - 验证机器人权限设置
   - 查看钉钉群组配置

3. **中文描述显示乱码**
   - 更新 `description.yaml` 映射表
   - 确认外网地址端口格式正确
