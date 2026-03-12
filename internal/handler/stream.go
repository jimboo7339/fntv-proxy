package handler

import (
	"fntv-proxy/internal/cache"
	"fntv-proxy/internal/logger"
	"net/http"
	"strings"
)

// StreamHandler 处理视频流请求
type StreamHandler struct {
	cache  *cache.Cache
	logger *logger.Logger
	client *http.Client
}

// NewStreamHandler 创建处理器
func NewStreamHandler(c *cache.Cache, l *logger.Logger) *StreamHandler {
	return &StreamHandler{
		cache:  c,
		logger: l,
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// 不自动跟随重定向，返回最后一个响应
				return http.ErrUseLastResponse
			},
		},
	}
}

// Handle 处理stream.mp4请求
func (h *StreamHandler) Handle(w http.ResponseWriter, r *http.Request) bool {
	// 检查是否是视频流请求
	if !isStreamRequest(r) {
		return false
	}

	h.logger.Info("🎬 拦截到视频流请求: %s", r.URL.Path)

	// 获取MediaSourceId
	mediaSourceID := r.URL.Query().Get("MediaSourceId")
	if mediaSourceID == "" {
		h.logger.Warn("❌ 缺少MediaSourceId参数")
		return false
	}

	// 从缓存查找
	source, found := h.cache.Get(mediaSourceID)
	if !found {
		h.logger.Warn("❌ MediaSourceId %s 不在缓存中", mediaSourceID)
		return false
	}

	// 检查是否是.strm
	if !strings.HasSuffix(source.Path, ".strm") {
		h.logger.Info("ℹ️ 不是.strm文件，直接转发")
		return false
	}

	// 读取.strm文件
	strmURL, err := ReadStrmFile(source.Path)
	if err != nil {
		h.logger.Error("❌ 读取.strm失败: %v", err)
		return false
	}

	h.logger.Info("📄 strm内容: %s", strmURL)

	// 请求strm URL，获取最终地址
	finalURL, err := h.resolveURL(strmURL)
	if err != nil {
		h.logger.Error("❌ 解析URL失败: %v", err)
		return false
	}

	h.logger.Info("✅ 最终地址: %s", finalURL)

	// 返回302重定向到最终地址
	w.Header().Set("Location", finalURL)
	w.WriteHeader(http.StatusFound)
	return true
}

// resolveURL 请求URL，跟随重定向，返回最终地址
func (h *StreamHandler) resolveURL(urlStr string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	// 设置请求头（模拟浏览器）
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 如果是302/301，获取Location头
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		location := resp.Header.Get("Location")
		if location != "" {
			return location, nil
		}
	}

	// 如果不是重定向，返回原始URL
	return urlStr, nil
}

// isStreamRequest 检查是否是视频流请求
func isStreamRequest(r *http.Request) bool {
	path := strings.ToLower(r.URL.Path)
	return strings.Contains(path, "/stream.") ||
		strings.Contains(path, "/master.m3u8")
}
