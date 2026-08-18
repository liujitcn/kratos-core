package module

import (
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Module 定义宿主业务模块向 Core 提供的协议注册能力。
type Module interface {
	// RegisterGRPC 将模块 gRPC 服务注册到宿主。
	RegisterGRPC(grpc.ServiceRegistrar)
	// RegisterHTTP 将模块 HTTP 服务注册到宿主。
	RegisterHTTP(*http.Server)
	// RegisterMCP 将模块 MCP 工具注册到宿主。
	RegisterMCP(*mcp.Server)
}

// Modules 聚合多个宿主业务模块，并统一转发协议注册。
type Modules []Module

// RegisterGRPC 将全部模块 gRPC 服务注册到宿主。
func (modules Modules) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	for _, module := range modules {
		if module != nil {
			module.RegisterGRPC(registrar)
		}
	}
}

// RegisterHTTP 将全部模块 HTTP 路由注册到宿主。
func (modules Modules) RegisterHTTP(server *http.Server) {
	for _, module := range modules {
		if module != nil {
			module.RegisterHTTP(server)
		}
	}
}

// RegisterMCP 将全部模块 MCP 工具注册到宿主。
func (modules Modules) RegisterMCP(server *mcp.Server) {
	for _, module := range modules {
		if module != nil {
			module.RegisterMCP(server)
		}
	}
}
