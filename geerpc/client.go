package geerpc

import (
	"Gee/geerpc/codec"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

// Call 承载一次RPC调用所需要的信息
type Call struct {
	Seq           uint64      // 序列号
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // 方法入参
	Reply         interface{} // 方法回调结果
	Error         error
	Done          chan *Call // strobes 根存 记录调用完成
}

// 支持异步调用，调用结束会调用此方法通知调用方
func (call *Call) done() {
	call.Done <- call
}

// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type Client struct {
	cc       codec.Codec
	opt      *Option
	sending  sync.Mutex   //互斥锁，保证请求有序发送
	header   codec.Header //每个消息请求头
	mu       sync.Mutex
	seq      uint64           //请求编号
	pending  map[uint64]*Call //存储未完成的请求，key-编号，value-Call实例
	closing  bool             // 用户主动关闭 Client
	shutdown bool             // Client 发生错误
}

// NewClient 新建客户端
func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	codecFunc := codec.NewCodecFuncMap[opt.CodecType]
	if codecFunc == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	err := json.NewEncoder(conn).Encode(opt)
	if err != nil {
		return nil, err
	}
	return newClientCodec(codecFunc(conn), opt), nil
}

// 初始化客户端
func newClientCodec(cc codec.Codec, opt *Option) *Client {
	// 新建 Client
	client := &Client{
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
		seq:     1,
	}
	// 协程开启接收方法进行for循环接收响应
	go client.receive()
	return client
}

// parseOptions 设置编码方式
func parseOptions(opts ...*Option) (*Option, error) {
	// if opts is nil or pass nil as parameter
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) == 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}

// Dial 用户传入服务端地址，创建 Client 实例，建立网络连接
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	// 建立网络连接
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return NewClient(conn, opt)
}

// 发送请求
func (client *Client) send(call *Call) {
	// make sure that the client will send a complete request
	client.sending.Lock()
	defer client.sending.Unlock()

	// register this call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// prepare request header
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = call.Seq
	client.header.Error = ""

	// encode and send the request
	if err := client.cc.Writer(&client.header, call.Args); err != nil {
		client.removeCall(seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go 异步接口，返回 Call 实例
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}

// Call 异步接口 Go 的封装
func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call))
	select {
	// context 当前程序单元（goroutine）的执行状态
	// 协程结束
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	// 调用完成
	case call := <-call.Done:
		return call.Error
	}
}

// Close 关闭 Client
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return rpc.ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

var _ io.Closer = (*Client)(nil)

// IsAvailable 客户端是否可用
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

// 将参数 call 添加到 client.pending 中，并更新 client.seq
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, rpc.ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// 根据 seq，从 client.pending 中移除对应的 Call，并返回
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// 服务端或客户端发生错误时调用
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	// 将 shutdown 设置为 true
	client.shutdown = true
	// 将错误信息通知所有 pending 状态的 call
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// 接收 Client 响应
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		//call 不存在，可能是请求没有发送完整，或者因为其他原因被取消
		//但是服务端仍旧处理了
		case call == nil:
			client.cc.ReadBody(nil)
		//call 存在，但服务端处理出错，即 h.Error 不为空
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}

// 超时处理的外壳
type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)

func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	options, err := parseOptions()
	if err != nil {
		return
	}
	// 建立网络连接
	conn, err := net.DialTimeout(network, address, options.ConnectTimeout)
	if err != nil {
		return
	}
	// client为空，关闭连接
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult)
	// 子协程执行 NewClient
	go func() {
		client, err := f(conn, options)
		// 执行完成后则通过信道 ch 发送结果
		ch <- clientResult{
			client: client,
			err:    err,
		}
	}()
	if options.HandleTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	select {
	// NewClient执行超时
	case <-time.After(options.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", options.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}
