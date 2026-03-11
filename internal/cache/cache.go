package cache

import (
	"sync"
)

// MediaSource 媒体源信息
type MediaSource struct {
	ID       string
	Path     string
	Protocol string
}

// Cache MediaSource缓存
type Cache struct {
	sources map[string]MediaSource // MediaSourceId -> MediaSource
	mutex   sync.RWMutex
}

// New 创建缓存
func New() *Cache {
	return &Cache{
		sources: make(map[string]MediaSource),
	}
}

// Set 设置缓存
func (c *Cache) Set(id string, source MediaSource) {
	c.mutex.Lock()
	c.sources[id] = source
	c.mutex.Unlock()
}

// Get 获取缓存
func (c *Cache) Get(id string) (MediaSource, bool) {
	c.mutex.RLock()
	source, found := c.sources[id]
	c.mutex.RUnlock()
	return source, found
}

// Delete 删除缓存
func (c *Cache) Delete(id string) {
	c.mutex.Lock()
	delete(c.sources, id)
	c.mutex.Unlock()
}
