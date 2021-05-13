package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

type Context struct {
	//原始结构
	Writer http.ResponseWriter
	Req    *http.Request
	//request信息
	Path   string
	Method string
	//提供对路由参数的访问
	Params map[string]string
	//response信息
	StatusCode int
	//middlewares中间件
	handlers []HandlerFunc
	//记录当前执行到第几个中间件，初始为-1
	index int

	//可以通过 Context 访问 Engine 中的 HTML 模板
	engine *Engine
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,

		index: -1,
	}
}

// Next 调用context的中间件
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

func (c *Context) Fail(code int, err string) {
	c.index = len(c.handlers)
	c.JSON(code, H{"message": err})
}

// PostForm 获取URL中?后面的请求参数
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

//解析GET请求中的参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

//设置响应status代码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

//设置响应头
func (c *Context) setHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

//string返回格式
func (c *Context) String(code int, format string, values ...interface{}) {
	c.setHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

//json返回格式
func (c *Context) JSON(code int, obj interface{}) {
	c.setHeader("Content-Type", "application/json")
	c.Status(code)
	encode := json.NewEncoder(c.Writer)
	if err := encode.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

//data返回格式
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

//html返回格式
func (c *Context) HTML(code int, name string, data interface{}) {
	c.setHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(500, err.Error())
	}
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}
