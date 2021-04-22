package codec

import "io"

// Header 请求头
type Header struct {
	ServiceMethod string //服务名和方法名
	Seq           uint64 //请求的序号
	Error         string //错误信息
}

// Codec 对消息体进行编解码的接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error   //消息请求头
	ReadBody(interface{}) error //消息请求体
	Writer(*Header, interface{}) error
}

// NewCodecFunc Codec 构造函数
type NewCodecFunc func(closer io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

// NewCodecFuncMap 构造函数map
var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
