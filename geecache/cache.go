package geecache

import (
	"Gee/geecache/lru"
	"sync"
)

// 并发控制
// 使用mutex互斥锁
type cache struct {
	mu        sync.Mutex
	lru       *lru.Cache
	cacheByte int64
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 延迟初始化（懒加载）
	// 对象的创建将会延迟至第一次使用该对象时
	if c.lru == nil {
		c.lru = lru.New(c.cacheByte, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		// 将value强转成ByteValue类型
		return v.(ByteView), true
	}

	return

}
