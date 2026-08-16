package client

import (
	"strings"

	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	"google.golang.org/grpc/metadata"
)

type localTransport struct {
	operation            string
	requestHeader        localHeader
	initialRequestHeader metadata.MD
	replyHeader          localHeader
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
