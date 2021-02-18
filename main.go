package main

import (
	"Gee/gee"
	"log"
	"net/http"
	"time"
)

func onlyForV2() gee.HandlerFunc {
	return func(c *gee.Context) {
		// Start timer
		t := time.Now()
		// if a server error occurred
		c.Fail(500, "Internal Server Error")
		// Calculate resolution time
		log.Printf("[%d] %s in %v for group v2", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

func main() {
	r := gee.New()

	// add middlewares to the group
	r.Use(gee.Logger()) // global middleware

	// add router
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})

	// add group 分组控制
	v2 := r.Group("/v2") //r为engine，调用RouterGroup中的Group方法（engine继承了RouterGroup）
	// 为v2分组添加中间件
	v2.Use(onlyForV2()) // v2 group middleware
	{
		//为v2组添加路由
		v2.GET("/hello/:name", func(c *gee.Context) {
			// expect /hello/geektutu
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
		})
	}

	r.Run(":9999")
}
