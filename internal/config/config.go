package config

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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

// Watch 监听配置变化
func Watch(onChange func()) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("📝 配置文件发生变化: %s", e.Name)

		// 重新加载
		if err := viper.Unmarshal(Global); err != nil {
			log.Printf("❌ 重新加载配置失败: %v", err)
			return
		}

		Global.CacheTTL = time.Duration(viper.GetInt("cache_ttl")) * time.Minute

		log.Println("✅ 配置已热重载")

		if onChange != nil {
			onChange()
		}
	})
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
