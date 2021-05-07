package geecache

import (
	"Gee/geecache/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 提供被其他节点访问的能力(基于http)

// 作为节点间通讯地址的前缀，默认是 /_geecache/
const defaultBasePath = "/_geecache/"
const defaultReplicas = 50

// HTTPPool
// 作为承载节点间http通信的数据结构
type HTTPPool struct {
	// 记录自己的地址
	self string
	// 节点通信地址的前缀
	basePath string

	mu sync.Mutex
	// 根据具体的key选择节点
	peers *consistenthash.Map
	// 每个节点的httpGetter
	httpGetters map[string]*httpGetter
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

// 继承了PeerGetter接口
type httpGetter struct {
	// 将要访问的远程节点的地址
	baseUrl string
}

// 实现PeerGetter接口
// 获取分布式节点的返回值，并转换成byte类型
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf("%v%v%v",
		h.baseUrl, url.QueryEscape(group), url.QueryEscape(key))
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned:%v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body:%v", err)
	}

	return bytes, nil
}

// 实例化一致性哈希算法，传入新节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 初始化哈希环
	p.peers = consistenthash.New(defaultReplicas, nil)
	// 将节点加入哈希环
	p.peers.Add(peers...)

	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		// 为每一个真实节点创建一个HTTP客户端httpGetter，用于返回节点数据
		p.httpGetters[peer] = &httpGetter{baseUrl: peer + p.basePath}
	}
}

// 返回节点对应的HTTP客户端
func (p *HTTPPool) PeerPicker(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}

	return nil, false
}
