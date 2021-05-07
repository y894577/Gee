package main

import (
	"Gee/geerpc"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// 启动服务器
func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	geerpc.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	client, _ := geerpc.Dial("tcp", <-addr)
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {

		wg.Add(1)
		go func(i int) {
			defer func() { wg.Done() }()
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			err := client.Call(ctx, "Foo.Sum", args, &reply)
			if err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}
