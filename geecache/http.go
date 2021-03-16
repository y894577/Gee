package geecache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// 提供被其他节点访问的能力(基于http)

// 作为节点间通讯地址的前缀，默认是 /_geecache/
const defaultBasePath = "/_geecache/"

// HTTPPool
// 作为承载节点间http通信的数据结构

type HTTPPool struct {
	// 记录自己的地址
	self string
	// 节点通信地址的前缀
	basePath string
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 不匹配则直接panic
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	// 根据/将数组进行分割
	// //参数n表示最多切分出几个子串，超出的部分将不再切分，最后一个n包含了所有剩下的不切分
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)

	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 获取路由组
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group："+groupName, http.StatusNotFound)
		return
	}

	// 从组里获取缓存数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 使用 w.Write() 将缓存值作为 httpResponse 的 body 返回
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())

}
