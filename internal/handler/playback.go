package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fntv-proxy/internal/cache"
	"fntv-proxy/internal/logger"
	"io"
	"net/http"
	"os"
	"strings"
)

// PlaybackInfoResponse PlaybackInfo响应结构
type PlaybackInfoResponse struct {
	ItemID       string `json:"ItemId"` // 添加ItemId字段
	MediaSources []struct {
		ID       string `json:"Id"`
		Path     string `json:"Path"`
		Protocol string `json:"Protocol"`
	} `json:"MediaSources"`
}

// PlaybackHandler 处理PlaybackInfo
type PlaybackHandler struct {
	cache  *cache.Cache
	logger *logger.Logger
}

// NewPlaybackHandler 创建处理器
func NewPlaybackHandler(c *cache.Cache, l *logger.Logger) *PlaybackHandler {
	return &PlaybackHandler{
		cache:  c,
		logger: l,
	}
}

// Handle 处理PlaybackInfo响应
func (h *PlaybackHandler) Handle(resp *http.Response, body []byte) ([]byte, error) {
	h.logger.Info("🎯 拦截到 PlaybackInfo 接口")

	// 解压gzip
	displayBody := body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		if decompressed, err := decompressGzip(body); err == nil {
			displayBody = decompressed
		}
	}

	// 解析JSON
	var playbackInfo PlaybackInfoResponse
	if err := json.Unmarshal(displayBody, &playbackInfo); err != nil {
		h.logger.Warn("JSON解析失败: %v", err)
		return body, nil
	}

	// 缓存所有.strm的MediaSource
	strmCount := 0
	for _, source := range playbackInfo.MediaSources {
		if strings.HasSuffix(source.Path, ".strm") {
			h.cache.Set(source.ID, cache.MediaSource{
				ID:       source.ID,
				ItemID:   playbackInfo.ItemID,
				Path:     source.Path,
				Protocol: source.Protocol,
			})
			strmCount++
			h.logger.Info("📄 缓存.strm: MediaSourceId=%s, ItemId=%s", source.ID, playbackInfo.ItemID)
		}
	}

	if strmCount > 0 {
		h.logger.Info("✅ 已缓存 %d 个.strm MediaSource", strmCount)
	}

	return body, nil
}

// ReadStrmFile 读取.strm文件
func ReadStrmFile(path string) (string, error) {
	// 统一路径分隔符
	cleanPath := strings.ReplaceAll(path, "\\", "/")

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}

	url := strings.TrimSpace(string(content))
	if url == "" {
		return "", os.ErrInvalid
	}

	return url, nil
}

func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
