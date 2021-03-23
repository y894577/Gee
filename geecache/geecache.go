package geecache

import (
	"Gee/geecache/singleflight"
	"fmt"
	"log"
	"sync"
)

// 回调函数
// 当缓存不存在的时候调用这个函数获得源数据
type Getter interface {
	Get(key string) ([]byte, error)
}

// 接口型函数
// 接口型函数只能应用于接口内部只定义了一个方法的情况
type GetterFunc func(key string) ([]byte, error)

// GetterFunc实现了接口Getter
// 可以将这个结构体封装作为参数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 负责与外部交互，控制缓存存储和获取的主流程。
// 一个 Group 可以认为是一个缓存的命名空间。
// 比如可以创建三个 Group。
// 缓存学生的成绩命名为scores，缓存学生信息的命名为info，缓存学生课程的命名为courses。
type Group struct {
	name string // group的唯一名称

	getter    Getter // 缓存未命中时获取源数据的回调
	mainCache cache  // 缓存

	peers PeerPicker // 每个group对应的分布式节点

	loader *singleflight.Group
}

// 全局变量
var (
	// 读写锁
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheByte: cacheBytes},
		loader:    &singleflight.Group{},
	}
	// 储存在全局变量中
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	// 只读锁 不涉及写操作
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

// 将数据添加在Group的缓存中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	// 从源数据中得到数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	// 获取源数据的拷贝
	value := ByteView{b: cloneByte(bytes)}
	// 将数据加入到缓存中
	g.populateCache(key, value)
	return value, nil
}

// 加载数据
func (g *Group) load(key string) (value ByteView, err error) {
	// 确保并发环境下针对相同的key，load过程只调用一次
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 选择节点
		if g.peers != nil {
			if peer, ok := g.peers.PeerPicker(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 从缓存中查找数据
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	// 缓存未命中 调用load方法获取源数据
	return g.load(key)
}

// 将实现了PeerPicker接口的HTTPPool的分布式节点peers注入到Group中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 从远程分布式节点获取
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, err
}
