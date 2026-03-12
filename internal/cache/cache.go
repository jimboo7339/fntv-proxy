package cache

import (
	"sync"
	"time"
)

// MediaSource 媒体源信息
type MediaSource struct {
	ID       string
	Path     string
	Protocol string
}

// cacheItem 带过期时间的缓存项
type cacheItem struct {
	source MediaSource
	expire time.Time
}

// Cache MediaSource缓存
type Cache struct {
	items       map[string]cacheItem
	mutex       sync.RWMutex
	ttl         time.Duration
	stopCleaner chan struct{}
}

// New 创建缓存，默认1小时过期
func New() *Cache {
	return NewWithTTL(1 * time.Hour)
}

// NewWithTTL 创建指定过期时间的缓存
func NewWithTTL(ttl time.Duration) *Cache {
	c := &Cache{
		items:       make(map[string]cacheItem),
		ttl:         ttl,
		stopCleaner: make(chan struct{}),
	}
	// 启动清理协程
	go c.cleaner()
	return c
}

// Set 设置缓存
func (c *Cache) Set(id string, source MediaSource) {
	c.mutex.Lock()
	c.items[id] = cacheItem{
		source: source,
		expire: time.Now().Add(c.ttl),
	}
	c.mutex.Unlock()
}

// Get 获取缓存（自动检查过期）
func (c *Cache) Get(id string) (MediaSource, bool) {
	c.mutex.RLock()
	item, found := c.items[id]
	c.mutex.RUnlock()

	if !found {
		return MediaSource{}, false
	}

	// 检查是否过期
	if time.Now().After(item.expire) {
		c.Delete(id)
		return MediaSource{}, false
	}

	return item.source, true
}

// Delete 删除缓存
func (c *Cache) Delete(id string) {
	c.mutex.Lock()
	delete(c.items, id)
	c.mutex.Unlock()
}

// cleaner 定期清理过期缓存
func (c *Cache) cleaner() {
	ticker := time.NewTicker(c.ttl / 2) // 每半周期清理一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleaner:
			return
		}
	}
}

// cleanup 清理过期项
func (c *Cache) cleanup() {
	now := time.Now()
	c.mutex.Lock()
	for id, item := range c.items {
		if now.After(item.expire) {
			delete(c.items, id)
		}
	}
	c.mutex.Unlock()
}

// Stop 停止清理协程
func (c *Cache) Stop() {
	close(c.stopCleaner)
}
