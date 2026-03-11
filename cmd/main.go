package main

import (
	"fntv-proxy/internal/proxy"
	"log"
	"os"
)

func main() {
	// 从环境变量读取配置
	listenAddr := getEnv("LISTEN_ADDR", ":28005")
	targetAddr := getEnv("TARGET_ADDR", "http://127.0.0.1:8005")
	logLevel := getEnv("LOG_LEVEL", "info") // debug, info, warn
	logDir := getEnv("LOG_DIR", "./logs")


	// 创建代理服务器
	server, err := proxy.NewServer(listenAddr, targetAddr, logLevel, logDir)
	if err != nil {
		log.Fatalf("创建代理服务器失败: %v", err)
	}

	// 启动
	log.Printf("🚀 FNTV Proxy 启动")
	log.Printf("   监听: %s", listenAddr)
	log.Printf("   目标: %s", targetAddr)
	log.Printf("   日志级别: %s", logLevel)

	if err := server.Start(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
