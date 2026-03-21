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

	// 从缓存查找（先按MediaSourceId查，找不到再按ItemId查）
	source, found := h.findInCache(r, mediaSourceID)
	if !found {
		h.logger.Warn("❌ MediaSourceId %s 和 ItemId 都不在缓存中", mediaSourceID)
		return false
	}

	// 检查是否是.strm
	if !strings.HasSuffix(source.Path, ".strm") {
		h.logger.Info("ℹ️ 不是.strm文件，直接转发")
		return false
	}

	// 尝试从缓存获取直链
	if streamURL, found := h.cache.GetStreamURL(source.ID); found {
		h.logger.Info("✅ 从缓存获取直链: %s", streamURL.URL)
		w.Header().Set("Location", streamURL.URL)
		w.WriteHeader(http.StatusFound)
		return true
	}

	// 读取.strm文件
	strmURL, err := ReadStrmFile(source.Path)
	if err != nil {
		h.logger.Error("❌ 读取.strm失败: %v", err)
		return false
	}

	h.logger.Info("📄 strm内容: %s", strmURL)

	// 请求strm URL，获取最终地址（透传原始请求的UA）
	finalURL, err := h.resolveURL(strmURL, r)
	if err != nil {
		h.logger.Error("❌ 解析URL失败: %v", err)
		return false
	}

	h.logger.Info("✅ 最终地址: %s", finalURL)

	// 缓存直链
	h.cache.SetStreamURL(source.ID, finalURL)
	h.logger.Info("💾 已缓存直链到 MediaSourceId: %s", source.ID)

	// 返回302重定向到最终地址
	w.Header().Set("Location", finalURL)
	w.WriteHeader(http.StatusFound)
	return true
}

// resolveURL 请求URL，跟随重定向，返回最终地址
// 透传原始请求的UA
func (h *StreamHandler) resolveURL(urlStr string, originalReq *http.Request) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	// 透传原始请求的UA
	originalUA := originalReq.Header.Get("User-Agent")
	if originalUA != "" {
		req.Header.Set("User-Agent", originalUA)
		h.logger.Debug("透传UA: %s", originalUA)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.0.0")
	}

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

// findInCache 从缓存查找MediaSource（支持MediaSourceId和ItemId）
func (h *StreamHandler) findInCache(r *http.Request, mediaSourceID string) (cache.MediaSource, bool) {
	// 1. 先按MediaSourceId查找
	if mediaSourceID != "" {
		if source, found := h.cache.Get(mediaSourceID); found {
			h.logger.Info("✅ 通过 MediaSourceId 找到缓存: %s", mediaSourceID)
			return source, true
		}
	}

	// 2. 从URL路径提取ItemId查找
	itemID := extractItemID(r.URL.Path)
	if itemID != "" {
		if source, found := h.cache.GetByItemID(itemID); found {
			h.logger.Info("✅ 通过 ItemId 找到缓存: %s", itemID)
			return source, true
		}
	}

	return cache.MediaSource{}, false
}

// extractItemID 从URL路径提取ItemId
// 例如: /emby/videos/064d9e7c19cb41ed884bcf8e22e64f80/stream.MKV
func extractItemID(path string) string {
	// 移除前缀 /emby
	path = strings.TrimPrefix(path, "/emby")

	// 匹配 /videos/{itemId}/ 或 /Items/{itemId}/
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if (part == "videos" || part == "Items") && i+1 < len(parts) {
			itemID := parts[i+1]
			// 验证是32位十六进制
			if len(itemID) == 32 {
				return itemID
			}
		}
	}
	return ""
}

// isStreamRequest 检查是否是视频流请求
func isStreamRequest(r *http.Request) bool {
	path := strings.ToLower(r.URL.Path)
	return strings.Contains(path, "/stream.") ||
		strings.Contains(path, "/master.m3u8")
}
