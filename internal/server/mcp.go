package server

import (
	"github.com/liujitcn/kratos-core/pkg/module"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

// MCPRuntime 描述 Core MCP 服务及其传输模式。
type MCPRuntime struct {
	// Server 是 Core 创建的底层 MCP 服务。
	Server *mcpserver.Server
	// InProcess 表示 MCP 是否挂载到 Core HTTP 服务。
	InProcess bool
}

// NewMCPServer 按 Server_Mcp 配置创建 MCP 服务，并注册全部模块工具。
func NewMCPServer(ctx *bootstrap.Context, modules module.Modules) (*MCPRuntime, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Mcp == nil {
		return nil, nil
	}
	inProcess := cfg.Server.Mcp.GetTransport() == bootstrapConfigv1.Server_Mcp_IN_PROCESS
	var server *mcpserver.Server
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
	return &MCPRuntime{Server: server, InProcess: inProcess}, nil
}
