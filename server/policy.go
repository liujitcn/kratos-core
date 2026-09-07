package server

import (
	"context"

	"google.golang.org/grpc"
)

type policyServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回保留传输元数据、取消信号和实例策略的流上下文。
func (s *policyServerStream) Context() context.Context {
	return s.ctx
}
