# FNTV Proxy

飞牛影视代理工具 - 自动解析 .strm 文件并重定向

已测试OpenList挂载的夸克TV生成的strm，在CapyPlayer播放器正常播放，理论上其他存储的strm也支持，如有问题请提issue

## 功能

- ✅ 透明代理飞牛影视服务
- ✅ 自动缓存 PlaybackInfo 中的 .strm MediaSource
- ✅ 拦截视频流请求，返回 302 重定向到真实 URL
- ✅ **配置文件热重载** - 修改配置无需重启
- ✅ 支持日志级别配置
- ✅ 缓存过期时间可配置
- ✅ 优雅关闭

## 快速开始

## 配置文件

创建 `config.yaml`：

```yaml
# 代理监听地址
listen: ":28005"

# 飞牛影视服务地址
target: "http://127.0.0.1:8005"

# 日志级别: debug / info / warn / error
log_level: "info"

# 日志目录（debug级别时写入，info级别可省略）
log_dir: "./logs"

# 缓存过期时间（分钟）
cache_ttl: 60
```

### 热重载

修改 `config.yaml` 后**自动生效**，无需重启容器：

```bash
# 修改配置
echo "log_level: debug" > config.yaml

# 1秒后自动生效，查看日志确认
docker-compose logs -f
# 📝 配置文件发生变化: /app/config.yaml
# ✅ 配置已热重载
```

## Docker Compose 配置

```yaml
services:
  fntv-proxy:
    image: jimboo7339/fntv-proxy:latest
    container_name: fntv-proxy
    ports:
      - "28005:28005"
    volumes:
      # 挂载strm文件目录（根据实际路径修改）
      - /vol00:/vol00:ro
      # 挂载配置文件（用于热重载）
      - ./config.yaml:/app/config.yaml:ro
    environment:
      # 配置文件路径
      - CONFIG=/app/config.yaml
    restart: unless-stopped
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `CONFIG` | 配置文件路径 | `./config.yaml` |
| `TZ` | 时区 | `Asia/Shanghai` |

## 日志级别说明

| 级别 | 输出位置 | 说明 |
|------|---------|------|
| `trace` | 文件 | **最详细**，记录完整请求/响应头、体（排查问题用） |
| `debug` | 控制台 + 文件 | 记录所有请求和响应 |
| `info` | 控制台 | 只输出关键信息（推荐生产环境） |
| `warn` | 控制台 | 只输出警告和错误 |
| `error` | 控制台 | 只输出错误 |

### trace 级别使用示例

```yaml
log_level: "trace"
log_dir: "./logs"
```

日志文件将包含：
```
=== REQUEST ===
Method: GET
URL: /Items/xxx/PlaybackInfo?MediaSourceId=yyy
Headers:
  User-Agent: xxx
  Authorization: xxx
Body: {...}
===============
=== RESPONSE ===
Request: GET /Items/xxx/PlaybackInfo
Status: 200
Headers:
  Content-Type: application/json
================
```

## 目录结构

```
fntv-proxy/
├── cmd/
│   └── main.go                 # 入口
├── internal/
│   ├── config/
│   │   └── config.go           # 配置管理（热重载）
│   ├── proxy/
│   │   └── server.go           # 代理服务器
│   ├── handler/
│   │   ├── playback.go         # PlaybackInfo处理
│   │   └── stream.go           # Stream处理
│   ├── cache/
│   │   └── cache.go            # MediaSource缓存
│   └── logger/
│       └── logger.go           # 日志
├── config.yaml                 # 配置文件
├── docker-compose.yml
├── Dockerfile
└── README.md
```

## 工作原理

```
1. 播放器 → PlaybackInfo → 代理缓存 MediaSource（含.strm路径）
                        ↓
2. 播放器 → stream.mp4/stream.MOV → 代理
                        ↓
3. 代理查缓存 → 读取.strm → 请求获取真实URL
                        ↓
4. 代理返回 302 → 播放器 → 真实URL播放
```

## 常见问题

### Q: 修改配置后需要重启吗？
**A:** 不需要！保存 `config.yaml` 后 1 秒内自动热重载。

### Q: 支持哪些视频格式？
**A:** 支持 `stream.mp4`、`stream.MOV` 等所有格式。

### Q: 缓存多久过期？
**A:** 默认 60 分钟，可通过 `cache_ttl` 配置。

### Q: 如何查看详细日志？
**A:** 修改 `log_level: debug`，会自动输出到 `./logs` 目录。

**strm路径一定要挂载到docker容器中，不然会播放失败，找不到strm路径**


## 声明

1. 本项目仅针对 **夸克网盘** 在 **openlist**的**夸克TV驱动**挂载下，实现302
2. 只要strm文件中的地址能正常下载文件，就可以通过本工具实现第三方播放器播放
3. 经测试 **CapyPlayer** **Vidhub** **爆米花** 下播放器正常播放
