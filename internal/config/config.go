package config

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Config 配置结构
type Config struct {
	ListenAddr string        `mapstructure:"listen"`
	TargetAddr string        `mapstructure:"target"`
	LogLevel   string        `mapstructure:"log_level"`
	LogDir     string        `mapstructure:"log_dir"`
	CacheTTL   time.Duration `mapstructure:"cache_ttl"`
	mutex      sync.RWMutex
}

// Global 全局配置实例
var Global = &Config{
	ListenAddr: ":28005",
	TargetAddr: "http://127.0.0.1:8005",
	LogLevel:   "info",
	LogDir:     "./logs",
	CacheTTL:   60 * time.Minute,
}

// Load 加载配置
func Load(configPath string) error {
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/app/configs/")
		viper.AddConfigPath("/etc/fntv-proxy/")
	}

	// 设置默认值
	viper.SetDefault("listen", ":28005")
	viper.SetDefault("target", "http://127.0.0.1:8005")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("log_dir", "./logs")
	viper.SetDefault("cache_ttl", 60)

	// 环境变量覆盖
	viper.SetEnvPrefix("FNTV")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		log.Println("⚠️ 未找到配置文件，使用默认配置")
	}

	// 解析到结构体
	if err := viper.Unmarshal(Global); err != nil {
		return err
	}

	// 转换 cache_ttl 为 Duration
	Global.CacheTTL = time.Duration(viper.GetInt("cache_ttl")) * time.Minute

	log.Printf("✅ 配置加载完成: %s", viper.ConfigFileUsed())
	return nil
}

// Watch 监听配置变化（支持 Docker 卷挂载）
func Watch(onChange func()) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		log.Println("⚠️ 未找到配置文件，跳过监听")
		return
	}

	// 使用轮询方式检测文件变化（兼容 Docker bind mount）
	go pollConfigChanges(configFile, onChange)
}

// pollConfigChanges 轮询检测配置文件变化
func pollConfigChanges(configFile string, onChange func()) {
	var lastModTime time.Time

	// 获取初始修改时间
	if info, err := os.Stat(configFile); err == nil {
		lastModTime = info.ModTime()
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		info, err := os.Stat(configFile)
		if err != nil {
			continue
		}

		// 检测到修改
		if info.ModTime().After(lastModTime) {
			lastModTime = info.ModTime()
			handleConfigChange(configFile, onChange)
		}
	}
}

// handleConfigChange 处理配置变更
func handleConfigChange(configFile string, onChange func()) {
	log.Printf("📝 配置文件发生变化: %s", configFile)

	// 重新读取配置文件
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("❌ 读取配置文件失败: %v", err)
		return
	}

	// 重新加载到结构体
	Global.mutex.Lock()
	if err := viper.Unmarshal(Global); err != nil {
		Global.mutex.Unlock()
		log.Printf("❌ 解析配置失败: %v", err)
		return
	}
	Global.CacheTTL = time.Duration(viper.GetInt("cache_ttl")) * time.Minute
	Global.mutex.Unlock()

	log.Println("✅ 配置已热重载")

	if onChange != nil {
		onChange()
	}
}

// GetListenAddr 获取监听地址（线程安全）
func (c *Config) GetListenAddr() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.ListenAddr
}

// GetTargetAddr 获取目标地址
func (c *Config) GetTargetAddr() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.TargetAddr
}

// GetLogLevel 获取日志级别
func (c *Config) GetLogLevel() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.LogLevel
}

// GetCacheTTL 获取缓存TTL
func (c *Config) GetCacheTTL() time.Duration {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.CacheTTL
}

// SetLogLevel 设置日志级别（热重载用）
func (c *Config) SetLogLevel(level string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.LogLevel = level
}
