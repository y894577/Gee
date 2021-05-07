package lru

import "container/list"

type Cache struct {
	// 允许使用最大内存
	maxBytes int64
	// 当前已使用内存
	nbytes int64
	// 双向链表
	ll *list.List
	// 字典，存储键和值的映射关系。
	// 这样根据某个键(key)查找对应的值(value)的复杂是O(1)
	// 在字典中插入一条记录的复杂度也是O(1)。
	cache map[string]*list.Element

	OnEvicted func(key string, value Value)
}

func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted}
}

// 第一步是从字典中找到对应的双向链表的节点，第二步，将该节点移动到队尾。
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 链表中的节点ele移动到队尾
		c.ll.MoveToFront(ele)
		// 获取ele节点的值
		// list 存储的是任意类型，使用时需要类型转换
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// 缓存淘汰，即移除最近最少访问的节点（队首）
func (c *Cache) RemoveOldest() {
	// 取到队首节点
	ele := c.ll.Back()
	if ele != nil {
		// 从链表中删除
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从字典中 c.cache 删除该节点的映射关系
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增/修改
func (c Cache) Add(key string, value Value) {
	// 如果节点存在在队列
	if ele, ok := c.cache[key]; ok {
		// 将节点移动到队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 更新已使用内存
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// 将节点添加到队尾
		ele := c.ll.PushFront(&entry{key, value})
		// 更新缓存
		c.cache[key] = ele
		// 更新已使用内存
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 缓存淘汰
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// 获取添加了多少条数据
func (c *Cache) Len() int {
	return c.ll.Len()
}

// 双向链表节点的数据类型
type entry struct {
	key   string
	value Value
}

// value接口
type Value interface {
	Len() int
}
