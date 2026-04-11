package proxy

import (
	"bytes"
	"context"
	"fntv-proxy/internal/cache"
	"fntv-proxy/internal/config"
	"fntv-proxy/internal/handler"
	"fntv-proxy/internal/logger"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Server 代理服务器
type Server struct {
	config          *config.Config
	logger          *logger.Logger
	cache           *cache.Cache
	playbackHandler *handler.PlaybackHandler
	streamHandler   *handler.StreamHandler
	proxy           *httputil.ReverseProxy
	httpServer      *http.Server
}

// NewServer 创建代理服务器
func NewServer(cfg *config.Config) (*Server, error) {
	// 创建日志
	log := logger.New(cfg.GetLogLevel(), cfg.LogDir)

	// 解析目标地址
	targetURL, err := url.Parse(cfg.GetTargetAddr())
	if err != nil {
		return nil, err
	}

	// 创建缓存（直链缓存使用 cache_ttl，MediaSource 不过期）
	c := cache.NewWithStreamTTL(cfg.GetCacheTTL())

	// 创建处理器
	ph := handler.NewPlaybackHandler(c, log)
	sh := handler.NewStreamHandler(c, log)

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return &Server{
		config:          cfg,
		logger:          log,
		cache:           c,
		playbackHandler: ph,
		streamHandler:   sh,
		proxy:           proxy,
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	// 设置Director
	originalDirector := s.proxy.Director
	s.proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = s.config.GetTargetAddr()
	}

	// 设置ModifyResponse
	s.proxy.ModifyResponse = s.handleResponse

	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:    s.config.GetListenAddr(),
		Handler: s.loggingMiddleware(s.proxy),
	}

	return s.httpServer.ListenAndServe()
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Reload 重新加载配置
func (s *Server) Reload() {
	// 更新日志级别
	s.logger.SetLevel(s.config.GetLogLevel())
	s.logger.Info("配置已重载，新日志级别: %s", s.config.GetLogLevel())
}

// handleResponse 处理响应
func (s *Server) handleResponse(resp *http.Response) error {
	// 只处理 PlaybackInfo，其他响应直接透传（避免读取大文件到内存）
	if !s.isPlaybackInfoRequest(resp.Request) {
		return nil
	}

	// 读取响应体（PlaybackInfo 数据量小，可以安全读取）
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// trace级别：记录响应体
	if s.logger.GetLevel() <= logger.TraceLevel {
		s.logResponseBody(resp, body)
	}

	// 处理 PlaybackInfo
	newBody, err := s.playbackHandler.Handle(resp, body)
	if err != nil {
		s.logger.Error("处理PlaybackInfo失败: %v", err)
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(newBody))
	return nil
}

// logResponseBody 记录响应体（trace级别）
func (s *Server) logResponseBody(resp *http.Response, body []byte) {
	s.logger.Trace("=== RESPONSE BODY ===")
	s.logger.Trace("Request: %s %s", resp.Request.Method, resp.Request.URL.Path)
	s.logger.Trace("Status: %d", resp.StatusCode)
	s.logger.Trace("Headers:")
	for name, values := range resp.Header {
		for _, v := range values {
			s.logger.Trace("  %s: %s", name, v)
		}
	}
	s.logger.Trace("Body: %s", truncate(string(body), 10000))
	s.logger.Trace("=====================")
}

// isPlaybackInfoRequest 检查是否是PlaybackInfo
func (s *Server) isPlaybackInfoRequest(req *http.Request) bool {
	// 支持 GET 和 POST 请求
	if req.Method != "POST" && req.Method != "GET" {
		return false
	}
	return len(req.URL.Path) > 0 &&
		contains(req.URL.Path, "/PlaybackInfo")
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "... (truncated)"
	}
	return s
}

// loggingMiddleware 日志中间件
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录请求（debug级别）
		s.logger.Debug("请求: %s %s", r.Method, r.URL.Path)

		// trace级别：记录完整请求信息（包括body）
		if s.logger.GetLevel() <= logger.TraceLevel {
			s.logRequest(r)
		}

		// 包装ResponseWriter以捕获响应
		wrapped := &responseRecorder{ResponseWriter: w, statusCode: 200}

		// 检查是否是视频流请求
		if s.streamHandler.Handle(wrapped, r) {
			return // 已处理，直接返回
		}

		// 继续处理
		next.ServeHTTP(wrapped, r)
	})
}

// logRequest 记录完整请求（trace级别），返回读取的body用于恢复
func (s *Server) logRequest(r *http.Request) []byte {
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Trace("读取请求体失败: %v", err)
	}
	// 恢复请求体，供后续处理使用
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// 记录请求详情
	s.logger.Trace("=== REQUEST ===")
	s.logger.Trace("Method: %s", r.Method)
	s.logger.Trace("URL: %s", r.URL.String())
	s.logger.Trace("Headers:")
	for name, values := range r.Header {
		for _, v := range values {
			s.logger.Trace("  %s: %s", name, v)
		}
	}
	if len(body) > 0 {
		s.logger.Trace("Body: %s", truncate(string(body), 10000))
	}
	s.logger.Trace("===============")

	return body
}

// responseRecorder 包装ResponseWriter以捕获状态码
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rec *responseRecorder) WriteHeader(code int) {
	if !rec.written {
		rec.statusCode = code
		rec.written = true
		rec.ResponseWriter.WriteHeader(code)
	}
}
