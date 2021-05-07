package singleflight

import "sync"

// call 代表正在进行中，或已经结束的请求。
type call struct {
	wg  sync.WaitGroup //WaitGroup并发，避免锁重入
	val interface{}
	err error
}

// 管理不同的key请求
type Group struct {
	mu sync.Mutex       // 保护 Group 的成员变量 m 不被并发读写而加上的锁。
	m  map[string]*call // 不同 key 的请求
}

// 对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次。
// 等待 fn 调用结束了，返回返回值或错误。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()

	if g.m == nil {
		g.m = make(map[string]*call) // 初始化call
	}

	//wg.Add(1) 锁加1。
	//wg.Wait() 阻塞，直到锁被释放。
	//wg.Done() 锁减1。

	// 查询对应的key是否有请求
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         // 如果请求正在进行中，则等待
		return c.val, c.err // 请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)  // 发起请求前加锁
	g.m[key] = c // 添加到g.m，表示kye已经有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn() // 调用fn，发起请求
	c.wg.Done()         // 执行完成，请求结束

	delete(g.m, key) // 更新g.m

	g.mu.Unlock()

	return c.val, c.err
}
