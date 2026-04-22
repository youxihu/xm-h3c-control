# 星目路由器端口快切工具 - 统一构建脚本
# 项目配置
APP_NAME = xm-port-switch
BACKEND_DIR = backend
FRONTEND_DIR = frontend
BACKEND_MAIN = $(BACKEND_DIR)/cmd/main.go
BACKEND_BIN = ./bin/$(APP_NAME)
FRONTEND_DIST = $(FRONTEND_DIR)/dist

# 默认目标
.DEFAULT_GOAL := help

# 帮助信息
.PHONY: help
help:
	@echo "星目路由器端口快切工具 - 构建脚本"
	@echo ""
	@echo "可用命令:"
	@echo "  make run-backend     - 运行后端服务"
	@echo "  make run-frontend    - 运行前端开发服务"
	@echo "  make run-all         - 同时运行前后端服务"
	@echo ""
	@echo "  make build-backend   - 构建后端可执行文件"
	@echo "  make build-frontend  - 构建前端静态文件"
	@echo ""
	@echo "  make clean-frontend         - 清理前端构建产物"
	@echo "  make clean-backend          - 清理后端构建产物"
	@echo ""

# ==================== 运行命令 ====================

# 运行后端服务
.PHONY: run-backend
run-backend:
	@echo "🚀 启动后端服务..."
	cd $(BACKEND_DIR) && go run cmd/main.go

# 运行前端开发服务
.PHONY: run-frontend
run-frontend:
	@echo "🚀 启动前端开发服务..."
	cd $(FRONTEND_DIR) && npm run dev

# 同时运行前后端（需要两个终端）
.PHONY: run-all
run-all:
	@echo "🚀 启动前后端服务..."
	@echo "注意：这将在后台启动后端，前端在前台运行"
	@echo "按 Ctrl+C 停止前端，后端需要手动停止"
	cd $(BACKEND_DIR) && go run cmd/main.go & \
	sleep 3 && \
	cd $(FRONTEND_DIR) && npm run dev

# ==================== 构建命令 ====================

# 构建后端
.PHONY: build-backend
build-backend:
	@echo "🔨 构建后端..."
	cd $(BACKEND_DIR) && go mod tidy
	cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BACKEND_BIN) cmd/main.go
	cd $(BACKEND_DIR) && upx -9 $(BACKEND_BIN)
	echo "upx 压缩完成"
	@echo "✅ 后端构建完成: $(BACKEND_BIN)"
	@ls -lh $(BACKEND_DIR)/$(BACKEND_BIN)

# 构建前端
.PHONY: build-frontend
build-frontend:
	@echo "🔨 构建前端..."
	cd $(FRONTEND_DIR) && npm run build
	@echo "✅ 前端构建完成: $(FRONTEND_DIST)"
	cd $(FRONTEND_DIR) && zip -r switch-port-frontend.zip dist/
	@ls -la $(FRONTEND_DIST)

.PHONY: build-docker
build-docker:
	@echo "🔨 构建后端Docker镜像..."
	docker build -f docker/Dockerfile -t registry.cn-hangzhou.xingmukeji.com/tools/xm-switch-port:v0.0.4 .

# ==================== 依赖管理 ====================

# 清理后端构建产物
.PHONY: clean-backend
clean-backend:
	@echo "🧹 清理后端构建产物..."
	cd $(BACKEND_DIR) && rm -rf bin/
	rm -f $(BACKEND_BIN)

# 清理前端构建产物
.PHONY: clean-frontend
clean-frontend:
	@echo "🧹 清理前端构建产物..."
	cd $(FRONTEND_DIR) && rm -rf dist/
