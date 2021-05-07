package gee

import (
	"net/http"
	"strings"
)

//路由表

// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']

type router struct {
	//存储每种请求方式的Trie树根节点
	roots map[string]*node
	//储存每种请求方式的HandlerFunc
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 分割pattern
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	//将新增的路由进行分割
	parts := parsePattern(pattern)

	//eg GET-/v1/hello/:lang
	key := method + "-" + pattern
	_, ok := r.roots[method]
	//没有route，新建node
	if !ok {
		r.roots[method] = &node{}
	}
	//将新增的路由插入到roots表中对应的node
	r.roots[method].insert(pattern, parts, 0)
	//将处理器handler加入到handlers表中
	r.handlers[key] = handler
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	//分割访问的URL
	searchParts := parsePattern(path)
	params := make(map[string]string)
	//查找roots表是否有该请求方法
	root, ok := r.roots[method]
	//如果没有直接返回nil
	if !ok {
		return nil, nil
	}

	//查找是否匹配成功
	n := root.search(searchParts, 0)

	//n不为空，查找成功
	if n != nil {
		//分割pattern，即路由规则
		parts := parsePattern(n.pattern)
		//遍历parts，填充params
		for index, part := range parts {
			//模糊匹配
			if part[0] == ':' {
				//params为URL的对应切片
				params[part[1:]] = searchParts[index]
			}
			//通配符匹配
			if part[0] == '*' && len(part) > 1 {
				//后面所有的URL切片合并为一个参数，并跳出循环
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}
	//n为空，返回nil
	return nil, nil
}

func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + c.Path
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}
