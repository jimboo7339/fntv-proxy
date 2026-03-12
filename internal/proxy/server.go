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

	// 创建缓存（使用配置中的TTL）
	c := cache.NewWithTTL(cfg.GetCacheTTL())

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
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// 检查是否是PlaybackInfo
	if s.isPlaybackInfoRequest(resp.Request) {
		newBody, err := s.playbackHandler.Handle(resp, body)
		if err != nil {
			s.logger.Error("处理PlaybackInfo失败: %v", err)
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(newBody))
		return nil
	}

	// 恢复原响应
	resp.Body = io.NopCloser(bytes.NewBuffer(body))
	return nil
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
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// loggingMiddleware 日志中间件
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录请求（debug级别）
		s.logger.Debug("请求: %s %s", r.Method, r.URL.Path)

		// trace级别：记录完整请求信息
		s.logRequest(r)

		// 包装ResponseWriter以捕获响应
		wrapped := &responseRecorder{ResponseWriter: w, statusCode: 200}

		// 检查是否是视频流请求
		if s.streamHandler.Handle(wrapped, r) {
			// trace级别：记录响应
			s.logResponse(r, wrapped)
			return // 已处理，直接返回
		}

		// 继续处理
		next.ServeHTTP(wrapped, r)

		// trace级别：记录响应
		s.logResponse(r, wrapped)
	})
}

// logRequest 记录完整请求（trace级别）
func (s *Server) logRequest(r *http.Request) {
	// 读取请求体
	body, _ := io.ReadAll(r.Body)
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
		s.logger.Trace("Body: %s", string(body))
	}
	s.logger.Trace("===============")
}

// logResponse 记录完整响应（trace级别）
func (s *Server) logResponse(r *http.Request, rec *responseRecorder) {
	s.logger.Trace("=== RESPONSE ===")
	s.logger.Trace("Request: %s %s", r.Method, r.URL.Path)
	s.logger.Trace("Status: %d", rec.statusCode)
	s.logger.Trace("Headers:")
	for name, values := range rec.Header() {
		for _, v := range values {
			s.logger.Trace("  %s: %s", name, v)
		}
	}
	s.logger.Trace("================")
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
