package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratosGRPC "github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-core/module"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/redact"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const policyTestText = "phone=13800138000"

type testPolicyResolver string

type policyTestModule struct {
	http func(*kratosHTTP.Server)
	grpc func(grpc.ServiceRegistrar)
}

type policyTestService struct {
	t        *testing.T
	resolver redact.PolicyResolver
}

// Resolve 将测试字符串替换为实例标识，以识别跨实例策略串用。
func (r testPolicyResolver) Resolve(context.Context, string) (redact.FieldPolicy, bool) {
	return redact.FieldPolicy{Mode: redact.PolicyModeApplyRule, Transform: func(any) any { return string(r) }}, true
}

// RegisterHTTP 注册测试 HTTP 路由。
func (m policyTestModule) RegisterHTTP(server *kratosHTTP.Server) {
	if m.http != nil {
		m.http(server)
	}
}

// RegisterGRPC 注册测试 gRPC 服务。
func (m policyTestModule) RegisterGRPC(server grpc.ServiceRegistrar) {
	if m.grpc != nil {
		m.grpc(server)
	}
}

// RegisterMCP 保持测试模块的 MCP 能力为空。
func (policyTestModule) RegisterMCP(*mcp.Server) {}

// checkContext 验证业务处理器拿到实例策略以及原有传输元数据。
func (s *policyTestService) checkContext(ctx context.Context, operation string) {
	s.t.Helper()
	if redact.PolicyResolverFromContext(ctx) != s.resolver {
		s.t.Errorf("请求策略未隔离: got=%v want=%v", redact.PolicyResolverFromContext(ctx), s.resolver)
	}
	info, ok := transport.FromServerContext(ctx)
	if !ok || info.Operation() != operation {
		s.t.Errorf("传输操作名丢失: %v", info)
	}
	md, _ := metadata.FromIncomingContext(ctx)
	if values := md.Get("x-policy-test"); len(values) != 1 || values[0] != "preserved" {
		s.t.Errorf("请求元数据丢失: %v", md)
	}
}

// TestHTTPPolicyContext 验证 HTTP 中间件、原生路由和编码器共享实例策略，且不修改上游请求。
func TestHTTPPolicyContext(t *testing.T) {
	for _, resolver := range []redact.PolicyResolver{testPolicyResolver("instance-a"), testPolicyResolver("instance-b"), nil} {
		name := "static"
		want := redact.SanitizeText(policyTestText)
		if resolver != nil {
			name = string(resolver.(testPolicyResolver))
			want = name
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			check := func(ctx context.Context) {
				if redact.PolicyResolverFromContext(ctx) != resolver {
					t.Errorf("HTTP 请求策略错误: got=%v want=%v", redact.PolicyResolverFromContext(ctx), resolver)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Error("HTTP 请求 deadline 丢失")
				}
			}
			middlewareCalls := 0
			encoderCalls := 0
			middlewares := HTTPMiddlewares{func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, request any) (any, error) {
					check(ctx)
					middlewareCalls++
					return next(ctx, request)
				}
			}}
			host := policyTestModule{http: func(server *kratosHTTP.Server) {
				kratosHTTP.ResponseEncoder(func(writer http.ResponseWriter, request *http.Request, value any) error {
					check(request.Context())
					encoderCalls++
					redact.ApplyWith(request.Context(), nil, value)
					return kratosHTTP.DefaultResponseEncoder(writer, request, value)
				})(server)
				server.Route("/").GET("policy", func(ctx kratosHTTP.Context) error {
					kratosHTTP.SetOperation(ctx, "/test.Policy/Unary")
					handler := ctx.Middleware(func(ctx context.Context, _ any) (any, error) {
						check(ctx)
						return wrapperspb.String(policyTestText), nil
					})
					result, err := handler(ctx, nil)
					if err != nil {
						return err
					}
					return ctx.Result(http.StatusOK, result)
				})
				server.HandleFunc("/raw", func(writer http.ResponseWriter, request *http.Request) {
					check(request.Context())
					message := wrapperspb.String(policyTestText)
					redact.ApplyWith(request.Context(), nil, message)
					err := kratosHTTP.DefaultResponseEncoder(writer, request, message)
					if err != nil {
						t.Errorf("编码原生响应失败: %v", err)
					}
				})
			}}
			cfg := &configv1.Bootstrap{Server: &configv1.Server{Http: &configv1.Server_Http{}}}
			ctx := bootstrap.NewContextWithParam(context.Background(), nil, cfg, nil)
			server, err := NewHTTPServer(ctx, nil, middlewares, module.Modules{host}, nil, nil, nil, nil, nil, resolver)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				err := server.Stop(context.Background())
				if err != nil {
					t.Errorf("停止 HTTP 服务失败: %v", err)
				}
			})
			requestContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			inherited := testPolicyResolver("upstream")
			requestContext = redact.WithPolicyResolver(requestContext, inherited)
			for _, path := range []string{"/policy", "/raw"} {
				request := httptest.NewRequestWithContext(requestContext, http.MethodGet, path, nil)
				recorder := httptest.NewRecorder()
				server.(http.Handler).ServeHTTP(recorder, request)
				if recorder.Code != http.StatusOK {
					t.Fatalf("HTTP 状态码错误: %d %s", recorder.Code, recorder.Body.String())
				}
				var got wrapperspb.StringValue
				err = json.Unmarshal(recorder.Body.Bytes(), &got)
				if err != nil || got.Value != want {
					t.Fatalf("HTTP 响应错误: got=%q want=%q err=%v", got.Value, want, err)
				}
				if redact.PolicyResolverFromContext(request.Context()) != inherited {
					t.Fatal("上游请求上下文被修改")
				}
			}
			if middlewareCalls != 1 || encoderCalls != 1 {
				t.Fatalf("HTTP 链路未完整执行: middleware=%d encoder=%d", middlewareCalls, encoderCalls)
			}
		})
	}
}

// TestGRPCPolicyContext 验证真实 unary 和三类流调用可使用请求策略，且共用描述符不会串用实例。
func TestGRPCPolicyContext(t *testing.T) {
	description := newPolicyTestDescription()
	originalUnary := reflect.ValueOf(description.Methods[0].Handler).Pointer()
	originalStream := reflect.ValueOf(description.Streams[0].Handler).Pointer()
	for _, resolver := range []redact.PolicyResolver{testPolicyResolver("instance-a"), testPolicyResolver("instance-b"), nil} {
		name := "static"
		want := redact.SanitizeText(policyTestText)
		if resolver != nil {
			name = string(resolver.(testPolicyResolver))
			want = name
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			service := &policyTestService{t: t, resolver: resolver}
			var registeredServer *kratosGRPC.Server
			var moduleMiddlewareCalls atomic.Int32
			host := policyTestModule{grpc: func(registrar grpc.ServiceRegistrar) {
				var ok bool
				registeredServer, ok = registrar.(*kratosGRPC.Server)
				if !ok {
					t.Fatalf("模块注册器类型被改变: %T", registrar)
				}
				registeredServer.Use("/test.Policy/*", func(next middleware.Handler) middleware.Handler {
					return func(ctx context.Context, request any) (any, error) {
						service.checkContext(ctx, "/test.Policy/Unary")
						moduleMiddlewareCalls.Add(1)
						return next(ctx, request)
					}
				})
				registrar.RegisterService(description, service)
			}}
			middlewares := GRPCMiddlewares{func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, request any) (any, error) {
					service.checkContext(ctx, "/test.Policy/Unary")
					return next(ctx, request)
				}
			}}
			cfg := &configv1.Bootstrap{Server: &configv1.Server{Grpc: &configv1.Server_Grpc{Addr: "127.0.0.1:0"}}}
			ctx := bootstrap.NewContextWithParam(context.Background(), nil, cfg, nil)
			server, err := NewGRPCServer(ctx, middlewares, module.Modules{host}, resolver)
			if err != nil {
				t.Fatal(err)
			}
			if registeredServer != server {
				t.Fatal("模块未收到原始 Kratos Server")
			}
			var endpoint *url.URL
			endpoint, err = server.Endpoint()
			if err != nil {
				t.Fatal(err)
			}
			stopped := make(chan error, 1)
			go func() { stopped <- server.Start(context.Background()) }()
			t.Cleanup(func() {
				err := server.Stop(context.Background())
				if err != nil {
					t.Errorf("停止 gRPC 服务失败: %v", err)
				}
				err = <-stopped
				if err != nil {
					t.Errorf("gRPC 服务退出失败: %v", err)
				}
			})
			var conn *grpc.ClientConn
			conn, err = grpc.NewClient(endpoint.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				err := conn.Close()
				if err != nil {
					t.Errorf("关闭 gRPC 连接失败: %v", err)
				}
			})
			requestContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			requestContext = metadata.AppendToOutgoingContext(requestContext, "x-policy-test", "preserved")
			response := new(wrapperspb.StringValue)
			err = conn.Invoke(requestContext, "/test.Policy/Unary", wrapperspb.String(policyTestText), response)
			if err != nil || response.Value != want {
				t.Fatalf("Unary 响应错误: got=%q want=%q err=%v", response.Value, want, err)
			}
			if moduleMiddlewareCalls.Load() != 1 {
				t.Fatal("模块通过 Use 挂载的中间件未执行")
			}
			for _, streamDescription := range description.Streams {
				var stream grpc.ClientStream
				stream, err = conn.NewStream(requestContext, &streamDescription, "/test.Policy/"+streamDescription.StreamName)
				if err != nil {
					t.Fatal(err)
				}
				err = stream.SendMsg(wrapperspb.String(policyTestText))
				if err != nil {
					t.Fatal(err)
				}
				err = stream.CloseSend()
				if err != nil {
					t.Fatal(err)
				}
				response = new(wrapperspb.StringValue)
				err = stream.RecvMsg(response)
				if err != nil || response.Value != want {
					t.Fatalf("%s 响应错误: got=%q want=%q err=%v", streamDescription.StreamName, response.Value, want, err)
				}
				err = stream.RecvMsg(new(wrapperspb.StringValue))
				if err != io.EOF {
					t.Fatalf("%s 未正常关闭: %v", streamDescription.StreamName, err)
				}
			}
			if reflect.ValueOf(description.Methods[0].Handler).Pointer() != originalUnary || reflect.ValueOf(description.Streams[0].Handler).Pointer() != originalStream {
				t.Error("共用服务描述符被修改")
			}
		})
	}
}

// TestPolicyServerStreamContext 验证流包装保留取消信号和 deadline，显式 nil 可屏蔽上游策略。
func TestPolicyServerStreamContext(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	parent = redact.WithPolicyResolver(parent, testPolicyResolver("upstream"))
	stream := &policyServerStream{ctx: redact.WithPolicyResolver(parent, nil)}
	if redact.PolicyResolverFromContext(stream.Context()) != nil {
		t.Fatal("nil 解析器未屏蔽上游策略")
	}
	deadline, ok := parent.Deadline()
	got, gotOK := stream.Context().Deadline()
	if !ok || !gotOK || !deadline.Equal(got) {
		t.Fatal("流 deadline 丢失")
	}
	cancel()
	if stream.Context().Err() != context.Canceled {
		t.Fatal("流取消信号丢失")
	}
}

// TestPolicyServersDisabled 验证 HTTP 与 gRPC 未启用时仍可传入 nil 解析器。
func TestPolicyServersDisabled(t *testing.T) {
	ctx := bootstrap.NewContextWithParam(context.Background(), nil, nil, nil)
	grpcServer, err := NewGRPCServer(ctx, nil, nil, nil)
	if err != nil || grpcServer != nil {
		t.Fatalf("未启用 gRPC 时应跳过创建: server=%v err=%v", grpcServer, err)
	}
	var httpServer transport.Server
	httpServer, err = NewHTTPServer(ctx, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil || httpServer != nil {
		t.Fatalf("未启用 HTTP 时应跳过创建: server=%v err=%v", httpServer, err)
	}
}

// newPolicyTestDescription 使用现有 Proto 类型构造 RPC 测试描述符，覆盖生成包装器使用的调用方式。
func newPolicyTestDescription() *grpc.ServiceDesc {
	description := &grpc.ServiceDesc{
		ServiceName: "test.Policy",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{{MethodName: "Unary", Handler: func(service any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			request := new(wrapperspb.StringValue)
			err := decode(request)
			if err != nil {
				return nil, err
			}
			handler := func(ctx context.Context, request any) (any, error) {
				service.(*policyTestService).checkContext(ctx, "/test.Policy/Unary")
				redact.ApplyWith(redact.WithOperation(ctx, "/test.Policy/Unary"), nil, request)
				return request, nil
			}
			if interceptor == nil {
				return handler(ctx, request)
			}
			return interceptor(ctx, request, &grpc.UnaryServerInfo{Server: service, FullMethod: "/test.Policy/Unary"}, handler)
		}}},
		Streams: []grpc.StreamDesc{
			{StreamName: "ServerStream", ServerStreams: true},
			{StreamName: "ClientStream", ClientStreams: true},
			{StreamName: "BidiStream", ClientStreams: true, ServerStreams: true},
		},
	}
	for index := range description.Streams {
		name := description.Streams[index].StreamName
		description.Streams[index].Handler = func(service any, stream grpc.ServerStream) error {
			operation := "/test.Policy/" + name
			service.(*policyTestService).checkContext(stream.Context(), operation)
			request := new(wrapperspb.StringValue)
			err := stream.RecvMsg(request)
			if err != nil {
				return err
			}
			typed := &grpc.GenericServerStream[wrapperspb.StringValue, wrapperspb.StringValue]{ServerStream: stream}
			switch name {
			case "ServerStream":
				wrapper := &redact.ServerStreamRedactor[wrapperspb.StringValue]{ServerStreamingServer: typed, Operation: operation}
				return wrapper.Send(request)
			case "ClientStream":
				wrapper := &redact.ClientStreamRedactor[wrapperspb.StringValue, wrapperspb.StringValue]{ClientStreamingServer: typed, Operation: operation}
				return wrapper.SendAndClose(request)
			default:
				wrapper := &redact.BidiStreamRedactor[wrapperspb.StringValue, wrapperspb.StringValue]{BidiStreamingServer: typed, Operation: operation}
				return wrapper.Send(request)
			}
		}
	}
	return description
}
