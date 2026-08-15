package mcp

import (
	"context"
	"errors"
	"net/url"

	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/pkg/module"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	mcpServer "github.com/liujitcn/kratos-kit/transport/mcp"
)

// Server 描述 Core MCP 服务及其传输模式。
type Server struct {
	// Server 是 Core 创建的底层 MCP 服务。
	Server *mcpServer.Server
	// InProcess 表示 MCP 是否挂载到 Core HTTP 服务。
	InProcess bool
}

var _ kratosTransport.Server = (*Server)(nil)
var _ kratosTransport.Endpointer = (*Server)(nil)

// NewServer 按 Server_Mcp 配置创建 MCP 服务，并注册全部模块工具。
func NewServer(ctx *bootstrap.Context, modules module.Modules) (*Server, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Mcp == nil {
		return nil, nil
	}
	inProcess := cfg.Server.Mcp.GetTransport() == bootstrapConfigv1.Server_Mcp_IN_PROCESS
	var server *mcpServer.Server
	var err error
	if inProcess {
		server, err = rpc.CreateMcpHandler(cfg)
	} else {
		server, err = rpc.CreateMcpServer(cfg)
	}
	if err != nil {
		return nil, err
	}
	if server != nil {
		modules.RegisterMCP(server)
	}
	return &Server{Server: server, InProcess: inProcess}, nil
}

// Start 启动独立 MCP 服务，进程内模式由 HTTP 宿主负责承载。
func (s *Server) Start(ctx context.Context) error {
	if s == nil || s.Server == nil || s.InProcess {
		return nil
	}
	return s.Server.Start(ctx)
}

// Stop 停止 MCP 服务并释放连接资源。
func (s *Server) Stop(ctx context.Context) error {
	if s == nil || s.Server == nil {
		return nil
	}
	return s.Server.Stop(ctx)
}

// Endpoint 返回独立 MCP 服务的注册端点。
func (s *Server) Endpoint() (*url.URL, error) {
	if s == nil || s.Server == nil {
		return nil, errors.New("MCP服务未初始化")
	}
	return s.Server.Endpoint()
}
