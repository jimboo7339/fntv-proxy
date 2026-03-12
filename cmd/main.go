package main

import (
	"fntv-proxy/internal/config"
	"fntv-proxy/internal/proxy"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 加载配置文件（从环境变量 CONFIG 获取路径，默认当前目录）
	configPath := os.Getenv("CONFIG")
	if err := config.Load(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 监听配置变化
	config.Watch(func() {
		log.Println("🔄 配置已更新")
		// 这里可以通知其他组件重新加载配置
	})

	// 创建代理服务器
	server, err := proxy.NewServer(config.Global)
	if err != nil {
		log.Fatalf("创建代理服务器失败: %v", err)
	}

	// 启动
	log.Printf("🚀 FNTV Proxy 启动")
	log.Printf("   监听: %s", config.Global.GetListenAddr())
	log.Printf("   目标: %s", config.Global.GetTargetAddr())
	log.Printf("   日志级别: %s", config.Global.GetLogLevel())
	log.Printf("   缓存TTL: %v", config.Global.GetCacheTTL())

	// 优雅关闭
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("🛑 正在关闭...")
		server.Stop()
	}()

	if err := server.Start(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
