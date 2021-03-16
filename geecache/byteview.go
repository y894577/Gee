package geecache

type ByteView struct {
	// 存储真实的缓存
	// 选择 byte 类型是为了能够支持任意的数据类型的存储，例如字符串、图片等
	b []byte
}

// 实现lru.Value中的接口
func (v ByteView) Len() int {
	return len(v.b)
}

func cloneByte(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func (v ByteView) String() string {
	return string(v.b)
}

// 返回拷贝值 防止缓存被外部修改
func (v ByteView) ByteSlice() []byte {
	return cloneByte(v.b)
}
