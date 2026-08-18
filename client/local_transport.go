package client

import (
	"strings"

	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type localTransport struct {
	operation            string
	requestHeader        localHeader
	initialRequestHeader metadata.MD
	replyHeader          localHeader
	replyTrailer         metadata.MD
}

// newLocalTransport 创建进程内调用使用的 Kratos gRPC transport。
func newLocalTransport(operation string, requestHeader metadata.MD) *localTransport {
	return &localTransport{
		operation:            operation,
		requestHeader:        localHeader(requestHeader),
		initialRequestHeader: requestHeader.Copy(),
		replyHeader:          make(localHeader),
	}
}

// Kind 返回本地 transport 的协议类型。
func (t *localTransport) Kind() kratosTransport.Kind { return kratosTransport.KindGRPC }

// Endpoint 返回进程内 transport 的空地址。
func (t *localTransport) Endpoint() string { return "" }

// Operation 返回当前 RPC 的完整方法名。
func (t *localTransport) Operation() string { return t.operation }

// Method 返回本地 gRPC 服务端传输的方法名。
func (t *localTransport) Method() string { return t.operation }

// SetHeader 累积本地 unary gRPC 响应头。
func (t *localTransport) SetHeader(md metadata.MD) error {
	t.replyHeader = localHeader(metadata.Join(metadata.MD(t.replyHeader), md))
	return nil
}

// SendHeader 累积并发送本地 unary gRPC 响应头。
func (t *localTransport) SendHeader(md metadata.MD) error {
	return t.SetHeader(md)
}

// SetTrailer 累积本地 gRPC 响应尾部元数据。
func (t *localTransport) SetTrailer(md metadata.MD) error {
	t.replyTrailer = metadata.Join(t.replyTrailer, md)
	return nil
}

var _ grpc.ServerTransportStream = (*localTransport)(nil)

type localStreamTransport struct {
	stream    grpc.ServerStream
	operation string
}

// Method 返回本地 stream transport 的方法名。
func (t localStreamTransport) Method() string { return t.operation }

// SetHeader 转发本地 stream 响应头。
func (t localStreamTransport) SetHeader(md metadata.MD) error {
	return t.stream.SetHeader(md)
}

// SendHeader 转发本地 stream 响应头发送操作。
func (t localStreamTransport) SendHeader(md metadata.MD) error {
	return t.stream.SendHeader(md)
}

// SetTrailer 转发本地 stream 响应尾部元数据。
func (t localStreamTransport) SetTrailer(md metadata.MD) error {
	t.stream.SetTrailer(md)
	return nil
}

// RequestHeader 返回本地调用请求头。
func (t *localTransport) RequestHeader() kratosTransport.Header { return t.requestHeader }

// ReplyHeader 返回本地调用响应头。
func (t *localTransport) ReplyHeader() kratosTransport.Header { return t.replyHeader }

type localHeader metadata.MD

// Get 返回请求头的第一个值。
func (h localHeader) Get(key string) string {
	values := metadata.MD(h).Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Set 覆盖请求头的已有值。
func (h localHeader) Set(key, value string) {
	metadata.MD(h).Set(strings.ToLower(key), value)
}

// Add 追加请求头值。
func (h localHeader) Add(key, value string) {
	metadata.MD(h).Append(strings.ToLower(key), value)
}

// Keys 返回全部请求头名称。
func (h localHeader) Keys() []string {
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	return keys
}

// Values 返回请求头的全部值。
func (h localHeader) Values(key string) []string { return metadata.MD(h).Get(key) }
