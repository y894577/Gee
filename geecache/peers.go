package geecache

// 分布式节点
type PeerPicker interface {
	// 根据传入的key选择相应节点PeerGetter
	PeerPicker(key string) (peer PeerGetter, ok bool)
}

// 从对应 group 查找缓存值
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
