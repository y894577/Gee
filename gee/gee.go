package gee

import (
	"net/http"
)

//提供给框架用户的，用来定义路由映射的处理方法
type HandlerFunc func(*Context)

//实现http.Handler接口
type Engine struct {
	router *router
}

func New() *Engine {
	//new new(T)会为T类型的新项目，分配被置零的存储，并且返回它的地址，一个类型为*T的值。
	//make(T, args)只用来创建slice，map和channel，并且返回一个初始化的(不是置零)，类型为T的值
	return &Engine{router: newRouter()}
}

func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)

}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}
