package cache

import (
	"sync"
	"time"
)

// MediaSource 媒体源信息
type MediaSource struct {
	ID       string
	ItemID   string // 关联的ItemId
	Path     string
	Protocol string
}

// StreamURL 视频流直链缓存
type StreamURL struct {
	URL        string
	MediaSrcID string // 关联的 MediaSourceId
}

// cacheItem 带过期时间的缓存项
type cacheItem struct {
	source MediaSource
	expire time.Time
}

// Cache 缓存
type Cache struct {
	items       map[string]cacheItem       // MediaSourceId -> MediaSource（有过期时间）
	itemIndex   map[string]string          // ItemId -> MediaSourceId
	streamURLs  map[string]streamCacheItem // MediaSourceId -> StreamURL（直链缓存，有过期时间）
	mutex       sync.RWMutex
	ttl         time.Duration              // MediaSource 和直链缓存共用同一个 TTL
	stopCleaner chan struct{}
}

// streamCacheItem 带过期时间的直链缓存项
type streamCacheItem struct {
	url    StreamURL
	expire time.Time
}

// New 创建缓存，默认1小时过期
func New() *Cache {
	return NewWithTTL(1 * time.Hour)
}

// NewWithStreamTTL 创建缓存，指定直链 TTL（MediaSource 不过期）
func NewWithStreamTTL(streamTTL time.Duration) *Cache {
	c := &Cache{
		items:       make(map[string]cacheItem),
		itemIndex:   make(map[string]string),
		streamURLs:  make(map[string]streamCacheItem),
		ttl:         streamTTL,
		stopCleaner: make(chan struct{}),
	}
	// 启动清理协程
	go c.cleaner()
	return c
}

// NewWithTTL 创建指定过期时间的缓存（兼容旧接口，ttl 用于直链缓存）
func NewWithTTL(ttl time.Duration) *Cache {
	return NewWithStreamTTL(ttl)
}

// Set 设置 MediaSource 缓存（使用同样的 TTL）
func (c *Cache) Set(id string, source MediaSource) {
	c.mutex.Lock()
	c.items[id] = cacheItem{
		source: source,
		expire: time.Now().Add(c.ttl),
	}
	// 建立ItemId索引
	if source.ItemID != "" {
		c.itemIndex[source.ItemID] = id
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

// GetByItemID 通过ItemId查找MediaSource
func (c *Cache) GetByItemID(itemID string) (MediaSource, bool) {
	c.mutex.RLock()
	mediaSourceID, found := c.itemIndex[itemID]
	c.mutex.RUnlock()

	if !found {
		return MediaSource{}, false
	}

	return c.Get(mediaSourceID)
}

// Delete 删除缓存
func (c *Cache) Delete(id string) {
	c.mutex.Lock()
	if item, ok := c.items[id]; ok && item.source.ItemID != "" {
		delete(c.itemIndex, item.source.ItemID)
	}
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

// SetStreamURL 设置直链缓存
func (c *Cache) SetStreamURL(mediaSourceID string, url string) {
	c.mutex.Lock()
	c.streamURLs[mediaSourceID] = streamCacheItem{
		url: StreamURL{
			URL:        url,
			MediaSrcID: mediaSourceID,
		},
		expire: time.Now().Add(c.ttl),
	}
	c.mutex.Unlock()
}

// GetStreamURL 获取直链缓存（自动检查过期）
func (c *Cache) GetStreamURL(mediaSourceID string) (StreamURL, bool) {
	c.mutex.RLock()
	item, found := c.streamURLs[mediaSourceID]
	c.mutex.RUnlock()

	if !found {
		return StreamURL{}, false
	}

	// 检查是否过期
	if time.Now().After(item.expire) {
		c.DeleteStreamURL(mediaSourceID)
		return StreamURL{}, false
	}

	return item.url, true
}

// DeleteStreamURL 删除直链缓存
func (c *Cache) DeleteStreamURL(mediaSourceID string) {
	c.mutex.Lock()
	delete(c.streamURLs, mediaSourceID)
	c.mutex.Unlock()
}

// cleanup 清理过期项（清理 MediaSource 和直链缓存）
func (c *Cache) cleanup() {
	now := time.Now()
	c.mutex.Lock()
	// 清理 MediaSource 缓存
	for id, item := range c.items {
		if now.After(item.expire) {
			if item.source.ItemID != "" {
				delete(c.itemIndex, item.source.ItemID)
			}
			delete(c.items, id)
		}
	}
	// 清理直链缓存
	for id, item := range c.streamURLs {
		if now.After(item.expire) {
			delete(c.streamURLs, id)
		}
	}
	c.mutex.Unlock()
}

// Stop 停止清理协程
func (c *Cache) Stop() {
	close(c.stopCleaner)
}
