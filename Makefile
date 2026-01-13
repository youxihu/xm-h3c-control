version := $(shell cat VERSION)
.PHONY: build docker docker_run docker_push

build:
	rm -rf ./bin
	mkdir -p bin/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o ./bin/xm-h3c-control ./cmd/main.go
	upx  ./bin/* && du -sh ./bin/*

docker:
	docker build -t "harbor-hz-xmkj.com/infra/xm-h3c-control:$(version)" .

docker_run:
	docker run -di \
            --name xm-h3c-control-v0.0.1 \
            -p 25003:25003 \
            -v /home/youxihu/mywork/myproject/xm-h3c-control/docker:/app-acc/configs \
            harbor-hz-xmkj.com/infra/xm-h3c-control:$(version)

docker_push:
	harbor-hz-xmkj.com/infra/xm-h3c-control:$(version)