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

# 安装 ca-certificates
RUN apk --no-cache add ca-certificates

# 复制二进制文件
COPY --from=builder /app/fntv-proxy /app/fntv-proxy

# 暴露端口
EXPOSE 28005

# 运行
ENTRYPOINT ["/app/fntv-proxy"]
