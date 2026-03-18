# 项目配置
APP_NAME = h3c-port-manager
BUILD_DIR = build
MAIN_PATH = cmd/main.go

.PHONY: run
run:
	go run $(MAIN_PATH)

.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)