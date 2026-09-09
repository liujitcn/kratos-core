package mcp_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-core/mcp"
	"github.com/liujitcn/kratos-core/module"
	coreserver "github.com/liujitcn/kratos-core/server"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/redact"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type testPolicyResolver string

type policyTestModule struct {
	t        *testing.T
	resolver redact.PolicyResolver
}

// Resolve 返回可区分应用实例的测试策略。
func (r testPolicyResolver) Resolve(context.Context, string) (redact.FieldPolicy, bool) {
	return redact.FieldPolicy{Mode: redact.PolicyModeApplyRule, Transform: func(any) any { return string(r) }}, true
}

// RegisterGRPC 保持测试模块的 gRPC 能力为空。
func (policyTestModule) RegisterGRPC(grpc.ServiceRegistrar) {}

// RegisterHTTP 保持测试模块的 HTTP 能力为空。
func (policyTestModule) RegisterHTTP(*kratosHTTP.Server) {}

// RegisterMCP 注册验证请求策略的模块中间件和工具。
func (m policyTestModule) RegisterMCP(server *mcpserver.Server) {
	server.MCPServer().AddReceivingMiddleware(func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(ctx context.Context, method string, request mcpsdk.Request) (mcpsdk.Result, error) {
			if redact.PolicyResolverFromContext(ctx) != m.resolver {
				m.t.Errorf("MCP 模块中间件策略错误: method=%s got=%v want=%v", method, redact.PolicyResolverFromContext(ctx), m.resolver)
			}
			return next(ctx, method, request)
		}
	})
	server.MCPServer().AddTool(&mcpsdk.Tool{Name: "policy", InputSchema: map[string]any{"type": "object"}}, func(ctx context.Context, request *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if redact.PolicyResolverFromContext(ctx) != m.resolver {
			m.t.Errorf("MCP 工具策略错误: got=%v want=%v", redact.PolicyResolverFromContext(ctx), m.resolver)
		}
		message := wrapperspb.String("phone=13800138000")
		redact.ApplyWith(ctx, nil, message)
		return &mcpsdk.CallToolResult{Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: message.Value}}}, nil
	})
}

// TestMCPPolicyContext 验证所有 MCP 传输模式的模块中间件和工具可读取隔离的实例策略。
func TestMCPPolicyContext(t *testing.T) {
	for _, mode := range []configv1.Server_Mcp_Transport{
		configv1.Server_Mcp_HTTP, configv1.Server_Mcp_SSE, configv1.Server_Mcp_STDIO, configv1.Server_Mcp_IN_PROCESS,
	} {
		for _, resolver := range []redact.PolicyResolver{testPolicyResolver("instance-a"), testPolicyResolver("instance-b"), nil} {
			name := "static"
			want := redact.SanitizeText("phone=13800138000")
			if resolver != nil {
				name = string(resolver.(testPolicyResolver))
				want = name
			}
			t.Run(mode.String()+"/"+name, func(t *testing.T) {
				t.Parallel()
				cfg := &configv1.Bootstrap{Server: &configv1.Server{
					Http: &configv1.Server_Http{},
					Mcp:  &configv1.Server_Mcp{Transport: mode, EnableKeepalive: new(bool)},
				}}
				bootstrapContext := bootstrap.NewContextWithParam(context.Background(), nil, cfg, nil)
				server, cleanup, err := mcp.NewServer(bootstrapContext, module.Modules{policyTestModule{t: t, resolver: resolver}}, resolver)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(cleanup)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				var clientTransport mcpsdk.Transport
				switch mode {
				case configv1.Server_Mcp_STDIO:
					// 使用与 STDIO 相同的逐行 JSON 传输，避免替换测试进程的标准输入输出。
					clientConn, serverConn := net.Pipe()
					clientTransport = &mcpsdk.IOTransport{Reader: clientConn, Writer: clientConn}
					var session *mcpsdk.ServerSession
					session, err = server.Server.MCPServer().Connect(redact.WithPolicyResolver(ctx, testPolicyResolver("upstream")), &mcpsdk.IOTransport{Reader: serverConn, Writer: serverConn}, nil)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						err := session.Close()
						if err != nil {
							t.Errorf("关闭 MCP 服务会话失败: %v", err)
						}
					})
				default:
					var handler http.Handler
					if mode == configv1.Server_Mcp_IN_PROCESS {
						var runtime transport.Server
						runtime, err = coreserver.NewHTTPServer(bootstrapContext, nil, nil, nil, nil, nil, nil, server, nil, testPolicyResolver("http-parent"))
						if err != nil {
							t.Fatal(err)
						}
						handler = runtime.(http.Handler)
						t.Cleanup(func() {
							err := runtime.Stop(context.Background())
							if err != nil {
								t.Errorf("停止 HTTP 宿主失败: %v", err)
							}
						})
					} else if mode == configv1.Server_Mcp_SSE {
						handler, err = server.Server.SSEHandler()
					} else {
						handler, err = server.Server.HTTPHandler()
					}
					if err != nil {
						t.Fatal(err)
					}
					httpServer := httptest.NewServer(handler)
					t.Cleanup(httpServer.Close)
					endpoint := httpServer.URL
					if mode == configv1.Server_Mcp_IN_PROCESS {
						endpoint += mcpserver.DefaultMCPHandlerPath
					}
					if mode == configv1.Server_Mcp_SSE {
						clientTransport = &mcpsdk.SSEClientTransport{Endpoint: endpoint}
					} else {
						clientTransport = &mcpsdk.StreamableClientTransport{Endpoint: endpoint}
					}
				}
				client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "policy-test", Version: "1"}, nil)
				var session *mcpsdk.ClientSession
				session, err = client.Connect(ctx, clientTransport, nil)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					err := session.Close()
					if err != nil {
						t.Errorf("关闭 MCP 客户端会话失败: %v", err)
					}
				})
				_, err = session.ListTools(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				var result *mcpsdk.CallToolResult
				result, err = session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "policy", Arguments: map[string]any{}})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError || len(result.Content) != 1 {
					t.Fatalf("MCP 返回异常结果: %+v", result)
				}
				text, ok := result.Content[0].(*mcpsdk.TextContent)
				if !ok || text.Text != want {
					t.Fatalf("MCP 响应错误: got=%v want=%q", result.Content, want)
				}
			})
		}
	}
}

// TestMCPPolicyDisabled 验证未启用 MCP 时仍允许显式传入 nil 解析器。
func TestMCPPolicyDisabled(t *testing.T) {
	ctx := bootstrap.NewContextWithParam(context.Background(), nil, nil, nil)
	server, cleanup, err := mcp.NewServer(ctx, nil, nil)
	if err != nil || server != nil {
		t.Fatalf("未启用 MCP 时应跳过创建: server=%v err=%v", server, err)
	}
	cleanup()
}
