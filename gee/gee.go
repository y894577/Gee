package gee

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

//提供给框架用户的，用来定义路由映射的处理方法
type HandlerFunc func(*Context)

// 实现http.Handler接口
// Engine is the uni handler for all requests
type Engine struct {
	router *router

	// embeded type 嵌套类型 类似java继承 Go语言的嵌套在其他语言中类似于继承
	// Engine继承了RouterGroup的分组功能, 同时还有其他的Run, ServeHTTP等接口功能
	*RouterGroup                // engine指向group
	groups       []*RouterGroup // store all groups 存储所有groups

	//将所有的模板加载进内存
	htmlTemplates *template.Template
	//所有的自定义模板渲染函数
	funcMap template.FuncMap
}

type RouterGroup struct {
	prefix      string        //前缀
	middlewares []HandlerFunc //中间件
	parent      *RouterGroup  //当前分组的父节点
	engine      *Engine       //group指向engine，通过engine间接地访问各种接口
}

func New() *Engine {
	//new new(T)会为T类型的新项目，分配被置零的存储，并且返回它的地址，一个类型为*T的值。
	//make(T, args)只用来创建slice，map和channel，并且返回一个初始化的(不是置零)，类型为T的值
	engine := &Engine{router: newRouter()}

	engine.RouterGroup = &RouterGroup{engine: engine}
	//将RouterGroup添加进groups中，此Group为全局的Group
	engine.groups = []*RouterGroup{engine.RouterGroup}

	return engine
}

func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	//将新建的Group添加到全局请求处理器Engine的groups表中
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

//原本为engine类函数，改为group，由group控制访问，通过group调用engine
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// add middleware to the group
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

// create static handler
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	//绝对路径
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(context *Context) {
		file := context.Param("filepath")
		// 检查路径是否存在
		if _, err := fs.Open(file); err != nil {
			context.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(context.Writer, context.Req)
	}
}

// 服务器静态资源
// relativePath 相对路径 root 磁盘文件夹目录
// r.Static("/assets", "/usr/web/blog/static")
// 或相对路径 r.Static("/assets", "./static")
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//接收到请求后，应查找所有应作用于该路由的中间件，保存在Context中，依次进行调用
	//中间件不仅作用在处理流程前，也可以作用在处理流程后
	//即在用户定义的 Handler 处理完毕后，还可以执行剩下的操作

	//当我们接收到一个具体请求时，要判断该请求适用于哪些中间件
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		//简单通过 URL 的前缀来判断属于哪个分组
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			//得到中间件列表后，赋值给c.handlers
			middlewares = append(middlewares, group.middlewares...)
		}
	}

	c := newContext(w, req)
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c)
}

//设置自定义渲染函数funcMap和加载模板
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

//加载html文件
func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}
