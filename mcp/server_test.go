package mcp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-core/module"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/redact"
	kitmcp "github.com/liujitcn/kratos-kit/transport/mcp"
	"github.com/mark3labs/mcp-go/client"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"google.golang.org/grpc"
)

// TestNewServerPolicyResolver 验证各传输配置均先于业务中间件注入当前实例的脱敏策略。
func TestNewServerPolicyResolver(t *testing.T) {
	for _, mode := range []configv1.Server_Mcp_Transport{configv1.Server_Mcp_HTTP, configv1.Server_Mcp_SSE, configv1.Server_Mcp_STDIO, configv1.Server_Mcp_IN_PROCESS} {
		t.Run(mode.String(), func(t *testing.T) {
			probe := &policyProbeModule{t: t}
			cfg := &configv1.Bootstrap{Server: &configv1.Server{Mcp: &configv1.Server_Mcp{Transport: mode}}}
			ctx := bootstrap.NewContextWithParam(t.Context(), nil, cfg, slog.Default())
			srv, cleanup, err := NewServer(ctx, module.Modules{probe}, probe)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(cleanup)
			var c *client.Client
			c, err = client.NewInProcessClient(srv.Server.MCPServer())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err = c.Close(); err != nil {
					t.Error(err)
				}
			})
			_, err = c.Initialize(t.Context(), mcpsdk.InitializeRequest{})
			if err != nil {
				t.Fatal(err)
			}
			req := mcpsdk.CallToolRequest{}
			req.Params.Name = "policy_probe"
			var result *mcpsdk.CallToolResult
			result, err = c.CallTool(t.Context(), req)
			if err != nil || result.IsError || !probe.called {
				t.Fatalf("工具未成功执行: result=%v err=%v", result, err)
			}
			if redact.PolicyResolverFromContext(context.Background()) != nil {
				t.Fatal("实例策略泄漏到其他请求")
			}
		})
	}
}

type policyProbeModule struct {
	t      *testing.T
	called bool
}

// RegisterGRPC 满足测试模块的 gRPC 注册契约。
func (*policyProbeModule) RegisterGRPC(grpc.ServiceRegistrar) {}

// RegisterHTTP 满足测试模块的 HTTP 注册契约。
func (*policyProbeModule) RegisterHTTP(*http.Server) {}

// RegisterMCP 验证模块中间件和工具处理器均能读取宿主策略。
func (p *policyProbeModule) RegisterMCP(srv *kitmcp.Server) {
	srv.MCPServer().Use(func(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
		return func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			if redact.PolicyResolverFromContext(ctx) != p {
				p.t.Error("模块中间件未读取到当前实例的策略")
			}
			return next(ctx, req)
		}
	})
	srv.MCPServer().AddTool(mcpsdk.NewTool("policy_probe"), func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		p.called = true
		if redact.PolicyResolverFromContext(ctx) != p {
			p.t.Error("工具处理器未读取到当前实例的策略")
		}
		return mcpsdk.NewToolResultText("ok"), nil
	})
}

// Resolve 满足脱敏策略解析器契约。
func (*policyProbeModule) Resolve(context.Context, string) (redact.FieldPolicy, bool) {
	return redact.FieldPolicy{}, false
}
