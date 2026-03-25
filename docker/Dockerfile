FROM alpine:latest

# 设置时区为中国上海
RUN mkdir -p /app-acc/config \
    && echo -e  "http://mirrors.aliyun.com/alpine/v3.4/main\nhttp://mirrors.aliyun.com/alpine/v3.4/community" >  /etc/apk/repositories \
    && apk update && apk add tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Shanghai/Asia" > /etc/timezone \
    && apk del tzdata


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