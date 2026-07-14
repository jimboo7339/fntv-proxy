package handler

import (
"encoding/json"
"fntv-proxy/internal/cache"
"fntv-proxy/internal/logger"
"io"
"net/http"
"net/url"
"strings"
)

type StreamHandler struct {
cache      *cache.Cache
logger     *logger.Logger
client     *http.Client
targetAddr string
}

func NewStreamHandler(c *cache.Cache, l *logger.Logger, targetAddr string) *StreamHandler {
return &StreamHandler{
cache: c, logger: l, targetAddr: targetAddr,
client: &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }},
}
}

func (h *StreamHandler) Handle(w http.ResponseWriter, r *http.Request) bool {
if !isStreamRequest(r) { return false }
h.logger.Info("🎬 拦截到视频流请求: %s", r.URL.Path)
mediaSourceID := r.URL.Query().Get("MediaSourceId")
source, found := h.findInCache(r, mediaSourceID)
if !found {
itemID := extractItemID(r.URL.Path)
if itemID != "" { source, found = h.fetchFromBackend(itemID, mediaSourceID, r) }
}
if !found {
h.logger.Warn("❌ MediaSourceId %s 和 ItemId 都不在缓存中", mediaSourceID)
return false
}
if !strings.HasSuffix(source.Path, ".strm") {
h.logger.Info("ℹ️ 不是.strm文件，直接转发")
return false
}
if streamURL, found := h.cache.GetStreamURL(source.ID); found {
h.logger.Info("✅ 从缓存获取直链: %s", streamURL.URL)
h.logDirectLinkType(streamURL.URL)
w.Header().Set("Location", streamURL.URL)
w.WriteHeader(http.StatusFound)
return true
}
strmURL, err := ReadStrmFile(source.Path)
if err != nil { h.logger.Error("❌ 读取.strm失败: %v", err); return false }
h.logger.Info("📄 strm内容: %s", strmURL)
finalURL, err := h.resolveURL(strmURL, r)
if err != nil { h.logger.Error("❌ 解析URL失败: %v", err); return false }
h.logger.Info("✅ 最终地址: %s", finalURL)
h.logDirectLinkType(finalURL)
h.cache.SetStreamURL(source.ID, finalURL)
h.logger.Info("💾 已缓存直链到 MediaSourceId: %s", source.ID)
w.Header().Set("Location", finalURL)
w.WriteHeader(http.StatusFound)
return true
}

func (h *StreamHandler) logDirectLinkType(finalURL string) {
linkType := ClassifyDirectLink(finalURL)
h.logger.Info("📡 直链类型: %s → %s", linkType, DirectLinkMetaHint(linkType))
}

func (h *StreamHandler) resolveURL(urlStr string, originalReq *http.Request) (string, error) {
req, err := http.NewRequest("GET", urlStr, nil)
if err != nil { return "", err }
ua := originalReq.Header.Get("User-Agent")
if ua == "" { ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" }
req.Header.Set("User-Agent", ua)
resp, err := h.client.Do(req)
if err != nil { return "", err }
defer resp.Body.Close()
if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
if loc := resp.Header.Get("Location"); loc != "" { return loc, nil }
}
return urlStr, nil
}

func (h *StreamHandler) findInCache(r *http.Request, mediaSourceID string) (cache.MediaSource, bool) {
if mediaSourceID != "" {
if s, f := h.cache.Get(mediaSourceID); f { return s, true }
}
itemID := extractItemID(r.URL.Path)
if itemID != "" {
if s, f := h.cache.GetByItemID(itemID); f { return s, true }
}
return cache.MediaSource{}, false
}

func (h *StreamHandler) fetchFromBackend(itemID, mediaSourceID string, r *http.Request) (cache.MediaSource, bool) {
apiKey := GetAPIKey(r)
if apiKey == "" { return cache.MediaSource{}, false }
targetURL := h.targetAddr
if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
targetURL = "http://" + targetURL
}
parsedTarget, _ := url.Parse(targetURL)
queryPath := "/Items/" + itemID + "/PlaybackInfo"
if mediaSourceID != "" { queryPath += "?MediaSourceId=" + mediaSourceID }
u := parsedTarget.ResolveReference(&url.URL{Path: queryPath})
q := u.Query(); q.Set("api_key", apiKey); u.RawQuery = q.Encode()
resp, err := http.DefaultClient.Get(u.String())
if err != nil { return cache.MediaSource{}, false }
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
var result struct {
	ItemID       string `json:"ItemId"`
	MediaSources []struct {
		ID   string `json:"Id"`
		Path string `json:"Path"`
	} `json:"MediaSources"`
}
json.Unmarshal(body, &result)
for _, ms := range result.MediaSources {
if mediaSourceID != "" && ms.ID != mediaSourceID { continue }
source := cache.MediaSource{ID: ms.ID, ItemID: result.ItemID, Path: ms.Path}
h.cache.Set(ms.ID, source)
return source, true
}
return cache.MediaSource{}, false
}

func extractItemID(path string) string {
path = strings.TrimPrefix(path, "/emby")
parts := strings.Split(path, "/")
for i, part := range parts {
if (part == "videos" || part == "Items") && i+1 < len(parts) {
if len(parts[i+1]) == 32 { return parts[i+1] }
}
}
return ""
}

func isStreamRequest(r *http.Request) bool {
path := strings.ToLower(r.URL.Path)
return strings.Contains(path, "/stream.") || strings.Contains(path, "/master.m3u8")
}
