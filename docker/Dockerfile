FROM alpine:latest

# Layer 1: 环境初始化 + 时区配置（合并所有系统操作）
RUN mkdir -p /app-acc/config \
    && echo -e "http://mirrors.aliyun.com/alpine/v3.19/main\nhttp://mirrors.aliyun.com/alpine/v3.19/community" > /etc/apk/repositories \
    && apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del --purge tzdata \
    && rm -rf /var/cache/apk/*

# Layer 2: 设置工作目录
ENV WORKDIR=/app-acc
WORKDIR $WORKDIR

# Layer 3: 复制文件 + 赋权（合并复制和权限操作）
COPY backend/bin/xm-port-switch $WORKDIR/xm-port-switch
COPY backend/config/* $WORKDIR/config/
RUN chmod +x $WORKDIR/xm-port-switch

# Layer 4: 启动
EXPOSE 25003
CMD ["./xm-port-switch"]