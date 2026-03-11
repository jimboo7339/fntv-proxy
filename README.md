# FNTV Proxy

飞牛影视代理工具 - 自动解析 .strm 文件并重定向

## 功能

- 透明代理飞牛影视服务
- 自动缓存 PlaybackInfo 中的 .strm MediaSource
- 拦截视频流请求，返回 302 重定向到真实 URL
- 支持日志级别配置

## 快速开始

### Docker Compose（推荐）

```bash
# 克隆项目
git clone <repo>
cd fntv-proxy

# 修改 docker-compose.yml 中的挂载路径
# - /vol00:/vol00:ro  ← 改成你的strm文件实际路径

# 启动
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 环境变量

| 变量 | 说明 | 默认值 |
|------|--------|--------|
| `LISTEN_ADDR` | 代理监听地址 | `:28005` |
| `TARGET_ADDR` | 飞牛影视地址 | `http://127.0.0.1:8005` |
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | `info` |
| `LOG_DIR` | 日志目录，空字符串表示不写文件 | `./logs` |

### 日志级别说明

- `debug`: 记录所有请求和响应到文件
- `info`: 只输出关键信息到控制台，不写文件
- `warn`: 只输出警告和错误
- `error`: 只输出错误

## 目录结构

```
fntv-proxy/
├── cmd/
│   └── main.go              # 入口
├── internal/
│   ├── proxy/
│   │   └── server.go        # 代理服务器
│   ├── handler/
│   │   ├── playback.go      # PlaybackInfo处理
│   │   └── stream.go        # Stream处理
│   ├── cache/
│   │   └── cache.go         # MediaSource缓存
│   └── logger/
│       └── logger.go        # 日志
├── configs/
│   └── docker-compose.yml     # Docker配置
├── Dockerfile
└── go.mod
```

## 使用示例

### 生产环境（推荐）

```yaml
# docker-compose.yml
version: '3.8'

services:
  fntv-proxy:
    build: .
    container_name: fntv-proxy
    ports:
      - "28005:28005"
    volumes:
      - /vol00:/vol00:ro        # strm文件目录
    environment:
      - TARGET_ADDR=http://192.168.1.100:8005
      - LOG_LEVEL=info          # 生产环境用info
    restart: unless-stopped
```

### 调试模式

```yaml
environment:
  - LOG_LEVEL=debug
  - LOG_DIR=/app/logs
volumes:
  - /vol00:/vol00:ro
  - ./logs:/app/logs
```

## 工作原理

1. **PlaybackInfo 拦截**: 缓存所有 .strm 的 MediaSource
2. **Stream 拦截**: 根据 MediaSourceId 查找缓存
3. **重定向**: 读取 .strm 文件，返回 302 到真实 URL

## 声明
1.本项目仅针对 **夸克网盘** 在 **openlist**的**夸克TV驱动**挂载下，实现302
2.经测试**CapyPlayer**下播放器正常播放
