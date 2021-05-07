package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 一致性哈希算法
//在新增/删除节点时，只需要重新定位该节点附近的一小部分数据，而不需要重新定位所有的节点

// 一致性哈希算法将 key 映射到 2^32 的空间中，将这个数字首尾相连，形成一个环。
// 计算节点/机器(通常使用节点的名称、编号和 IP 地址)的哈希值，放置在环上。
// 计算 key 的哈希值，放置在环上，顺时针寻找到的第一个节点，就是应选取的节点/机器。

type Hash func(data []byte) uint32

// 如果服务器的节点过少，容易引起 key 的倾斜。
// 虚拟节点扩充了节点的数量，解决了节点较少的情况下数据容易倾斜的问题。

type Map struct {
	// 哈希函数
	hash Hash
	// 虚拟节点倍数
	replicas int
	// 哈希环
	keys []int
	// 虚拟节点与真实节点映射表
	hashMap map[int]string
}

// 依赖注入hash函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点/机器的 Add() 方法
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		//对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
		for i := 0; i < m.replicas; i++ {
			//计算虚拟节点的 Hash 值，放置在环上
			//假设 1 个真实节点对应 3 个虚拟节点
			//那么 peer1 对应的虚拟节点是 peer1-1、 peer1-2、 peer1-3
			//strconv.Itoa()方法将数字转换成对应的字符串类型的数字
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			//增加虚拟节点和真实节点的映射关系
			m.hashMap[hash] = key
		}
		// 对map的keys哈希环进行排序
		sort.Ints(m.keys)
	}
}

func (m *Map) Get(key string) string {
	// 不存在真实节点
	if len(m.keys) == 0 {
		return ""
	}

	// 计算key的哈希值
	hash := int(m.hash([]byte(key)))

	// 二分查找哈希环
	//顺时针找到第一个匹配的虚拟节点的下标idx
	idx := sort.Search(len(m.keys), func(i int) bool {
		// 从m.keys中获取到对应的哈希值
		return m.keys[i] >= hash
	})

	// 通过hashmap映射到真实节点
	// idx%len(m.keys) 为最近节点对应的下标
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
