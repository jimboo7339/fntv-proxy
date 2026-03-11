# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制源码
COPY . .

# 编译
RUN go build -o fntv-proxy ./cmd/main.go

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装 ca-certificates 和 tzdata（时区数据）
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为 Asia/Shanghai（东八区）
ENV TZ=Asia/Shanghai
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 复制二进制文件
COPY --from=builder /app/fntv-proxy /app/fntv-proxy

# 暴露端口
EXPOSE 28005

# 运行
ENTRYPOINT ["/app/fntv-proxy"]
