package geerpc

import (
	"Gee/geerpc/codec"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
)

//| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
//| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|

const MagicNumber = 0x3bef5c

// Option 消息的编解码方式
type Option struct {
	MagicNumber int        // 魔数 标记这是 rpc 请求
	CodecType   codec.Type // 选择的编码方式
}

// DefaultOption 默认配置
var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents an RPC Server.
type Server struct {
	serviceMap sync.Map
}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer() //默认服务器

// Accept 接收监听器和服务器请求
func Accept(listener net.Listener) {
	DefaultServer.Accept(listener)
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(listener net.Listener) {
	// 一次连接中允许接收多个请求
	// for 无限制地等待请求的到来，直到发生错误
	for {
		conn, err := listener.Accept() // 等待socket建立连接
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn) // 协程并发执行请求
	}
}

// ServeConn 执行请求
// runs the server on a single connection.
// blocks, serving the connection until the client hangs up.
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	// 反序列化得到 Option 实例
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 并发执行请求
		go server.handleRequest(cc, req, sending, wg)
	}
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
	mtype        *methodType
	svc          *service
}

// 读取请求头RequestHeader
func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

// 读取请求体Request
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: header}
	req.svc, req.mtype, err = server.findService(header.ServiceMethod)
	if err != nil {
		return req, err
	}
	//通过 newArgv() 和 newReplyv() 两个方法创建出两个入参实例
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// 确保 argvi 是 pointer 类型
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	//通过 cc.ReadBody() 将请求报文反序列化为第一个入参 argv
	if err := cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv err:", err)
		return nil, err
	}
	return req, nil
}

// 返回Response
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	// 处理请求是并发的，但是回复请求的报文必须是逐个发送的
	// 并发容易导致多个回复报文交织在一起，客户端无法解析
	// 在这里使用锁(sending)保证
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Writer(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

// 处理请求
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	err := req.svc.call(req.mtype, req.argv, req.replyv)
	if err != nil {
		req.h.Error = err.Error()
		server.sendResponse(cc, req.h, invalidRequest, sending)
		return
	}
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

// Register 通过自定义 Server 注册
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)

	if _, loaded := server.serviceMap.LoadOrStore(s.name, s); loaded {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

// Register 通过 DefaultServer 注册
func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

// findService 发现查找服务，格式 string = "Service.Method"
func (server *Server) findService(serviceMethod string) (svc *service, mType *methodType, err error) {
	//先将其分割成 2 部分
	//第一部分是 Service 的名称
	//第二部分即方法名
	dot := strings.LastIndex(serviceMethod, ".")
	serviceName, serviceMethod := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mType = svc.method[serviceMethod]
	if mType == nil {
		err = errors.New("rpc server: can't find method " + serviceMethod)
	}
	return
}
