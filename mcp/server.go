package mcp

import (
	"context"
	"errors"
	"net/url"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/module"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/redact"
	servermcp "github.com/liujitcn/kratos-kit/server/mcp"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server 描述 Core MCP 服务及其传输模式。
type Server struct {
	// Server 是 Core 创建的底层 MCP 服务。
	Server *mcp.Server
	// InProcess 表示 MCP 是否挂载到 Core HTTP 服务。
	InProcess bool
}

var _ transport.Server = (*Server)(nil)
var _ transport.Endpointer = (*Server)(nil)

// NewServer 按配置创建 MCP 服务，为各传输模式注入实例策略并注册模块工具。
func NewServer(ctx *bootstrap.Context, modules module.Modules, policyResolver redact.PolicyResolver) (*Server, func(), error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Mcp == nil {
		return nil, func() {}, nil
	}
	inProcess := cfg.Server.Mcp.GetTransport() == configv1.Server_Mcp_IN_PROCESS
	var server *mcp.Server
	var err error
	if inProcess {
		server, err = servermcp.CreateMcpHandler(cfg)
	} else {
		server, err = servermcp.CreateMcpServer(cfg)
	}
	if err != nil {
		return nil, func() {}, err
	}
	runtime := &Server{Server: server, InProcess: inProcess}
	cleanup := func() {
		if runtime.Server == nil {
			return
		}
		if stopErr := runtime.Stop(context.Background()); stopErr != nil {
			log.Error("停止 MCP 服务失败", "error", stopErr)
		}
	}
	if server != nil {
		modules.RegisterMCP(server)
		// SDK 接收中间件统一覆盖独立 HTTP、SSE、STDIO 和挂载模式，且先于模块中间件执行。
		server.MCPServer().AddReceivingMiddleware(func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
			return func(ctx context.Context, method string, request mcpsdk.Request) (mcpsdk.Result, error) {
				return next(redact.WithPolicyResolver(ctx, policyResolver), method, request)
			}
		})
	}
	return runtime, cleanup, nil
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
