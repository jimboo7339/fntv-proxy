# FNTV Proxy

飞牛影视代理工具 - 自动解析 .strm 文件并重定向

已测试OpenList挂载的夸克TV生成的strm，在CapyPlayer播放器正常播放，理论上其他存储的strm也支持，如有问题请提issue

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
services:
  fntv-proxy:
    image: jimboo7339/fntv-proxy:latest
    container_name: fntv-proxy
    ports:
      - "28005:28005"
    volumes:
      # 挂载strm文件目录（根据实际路径修改）
      - /vol00:/vol00:ro
      # 可选：挂载日志目录（debug模式）
      # - ./logs:/app/logs
    environment:
      # 飞牛影视地址
      - TARGET_ADDR=http://127.0.0.1:8005
      # 日志级别: debug/info/warn/error
      - LOG_LEVEL=info
      # strm缓存过期时间 默认1小时
      - CACHE_TL=60
      # 日志目录（debug模式需要，info模式可省略）
      # - LOG_DIR=/app/logs
      # 或 America/New_York, Europe/London 等
      # - TZ=Asia/Shanghai
    restart: unless-stopped
    # 使用host网络可以访问本机8005
    # network_mode: host
```
**strm路径一定要挂载到docker容器中，不然会播放失败，找不到strm路径**

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

1. 本项目仅针对 **夸克网盘** 在 **openlist**的**夸克TV驱动**挂载下，实现302
2. 只要strm文件中的地址能正常下载文件，就可以通过本工具实现第三方播放器播放
3. 经测试 **CapyPlayer** **Vidhub** **爆米花** 下播放器正常播放
