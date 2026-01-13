FROM alpine:latest

RUN mkdir -p /app-acc/configs

ENV WORKDIR /app-acc

RUN echo -e  "http://mirrors.aliyun.com/alpine/v3.4/main\nhttp://mirrors.aliyun.com/alpine/v3.4/community" >  /etc/apk/repositories \
    && apk update && apk add tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Shanghai/Asia" > /etc/timezone \
    && apk del tzdata

WORKDIR $WORKDIR

COPY ./bin/xm-h3c-control $WORKDIR/xm-h3c-control

RUN chmod +x $WORKDIR/xm-h3c-control

EXPOSE 25003
# start
CMD ["./xm-h3c-control"]