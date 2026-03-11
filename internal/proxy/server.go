package proxy

import (
	"bytes"
	"fntv-proxy/internal/cache"
	"fntv-proxy/internal/handler"
	"fntv-proxy/internal/logger"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Server 代理服务器
type Server struct {
	listenAddr string
	targetURL  *url.URL
	logger     *logger.Logger
	cache      *cache.Cache
	playbackHandler *handler.PlaybackHandler
	streamHandler *handler.StreamHandler
	proxy      *httputil.ReverseProxy
}

// NewServer 创建代理服务器
func NewServer(listenAddr, targetAddr, logLevel, logDir string) (*Server, error) {
	// 创建日志
	log := logger.New(logLevel, logDir)

	// 解析目标地址
	targetURL, err := url.Parse(targetAddr)
	if err != nil {
		return nil, err
	}

	// 创建缓存
	c := cache.New()

	// 创建处理器
	ph := handler.NewPlaybackHandler(c, log)
	sh := handler.NewStreamHandler(c, log)

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return &Server{
		listenAddr:    listenAddr,
		targetURL:     targetURL,
		logger:        log,
		cache:         c,
		playbackHandler: ph,
		streamHandler: sh,
		proxy:         proxy,
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	// 设置Director
	originalDirector := s.proxy.Director
	s.proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = s.targetURL.Host
	}

	// 设置ModifyResponse
	s.proxy.ModifyResponse = s.handleResponse

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    s.listenAddr,
		Handler: s.loggingMiddleware(s.proxy),
	}

	return server.ListenAndServe()
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
	return req.Method == "POST" &&
		len(req.URL.Path) > 0 &&
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

		// 检查是否是视频流请求
		if s.streamHandler.Handle(w, r) {
			return // 已处理，直接返回
		}

		// 继续处理
		next.ServeHTTP(w, r)
	})
}
