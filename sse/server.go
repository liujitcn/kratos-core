package sse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport"
	_const "github.com/liujitcn/kratos-core/const"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	serversse "github.com/liujitcn/kratos-kit/server/sse"
	"github.com/liujitcn/kratos-kit/transport/sse"
	"google.golang.org/protobuf/proto"
)

var errStreamNotRegistered = errors.New("SSE流未注册")

// Server 描述 Core SSE 服务及其传输模式。
type Server struct {
	// Server 是 Core 创建的底层 SSE 服务。
	Server *sse.Server
	// InProcess 表示 SSE 是否挂载到 Core HTTP 服务。
	InProcess bool
	resolver  sse.StreamIDResolver
}

// Streams 聚合宿主提供的 SSE 业务流定义。
type Streams []sse.SSEStream

var _ transport.Server = (*Server)(nil)
var _ transport.Endpointer = (*Server)(nil)

// NewStreamResolver 创建根据宿主流声明解析实际传输流的请求解析器。
func NewStreamResolver(registry *Registry, authenticator engine.Authenticator, userToken *data.UserToken) sse.StreamIDResolver {
	return func(request *http.Request) (string, error) {
		if request == nil || request.URL == nil {
			return "", fmt.Errorf("SSE 请求地址不能为空")
		}
		streamID := request.URL.Query().Get("stream")
		channelID := request.URL.Query().Get("channel")
		userID, err := requestUserID(request, authenticator, userToken)
		if err != nil {
			return "", err
		}
		if registry == nil {
			return streamID, nil
		}
		var transportID string
		var found bool
		transportID, found, err = registry.Resolve(streamID, channelID, userID)
		if err != nil {
			return "", err
		}
		if found {
			return transportID, nil
		}
		return "", fmt.Errorf("%w: %s", errStreamNotRegistered, streamID)
	}
}

// NewServer 按 Server_Sse 配置创建 SSE 服务、注册宿主业务流，并返回进程内服务清理函数。
func NewServer(ctx *bootstrap.Context, resolver sse.StreamIDResolver, streams Streams, registry *Registry) (*Server, func(), error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Sse == nil {
		return nil, func() {}, nil
	}
	options := make([]sse.ServerOption, 0, 2)
	var server *sse.Server
	transportResolver := resolver
	if resolver != nil {
		options = append(options, sse.WithStreamIDResolver(transportResolver), sse.WithAutoStream(true))
	}
	inProcess := cfg.Server.Sse.GetTransport() == configv1.Server_Sse_IN_PROCESS
	var err error
	if inProcess {
		server, err = serversse.CreateSseHandler(cfg, options...)
	} else {
		// 独立模式也延迟到 Kratos Start 阶段监听，避免后续 Provider 失败时泄漏监听器。
		handlerConfig := proto.Clone(cfg).(*configv1.Bootstrap)
		handlerConfig.Server.Sse.Transport = configv1.Server_Sse_IN_PROCESS
		server, err = serversse.CreateSseHandler(handlerConfig, options...)
	}
	if err != nil {
		return nil, func() {}, err
	}
	runtime := &Server{Server: server, InProcess: inProcess, resolver: transportResolver}
	cleanup := func() {
		if runtime.Server == nil {
			return
		}
		if stopErr := runtime.Stop(context.Background()); stopErr != nil {
			log.Error("停止 SSE 服务失败", "error", stopErr)
		}
	}
	if err = server.RegisterStream(streams...); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if registry != nil {
		if err = registry.Register(server.StreamDefinitions()...); err != nil {
			cleanup()
			return nil, func() {}, err
		}
	}
	return runtime, cleanup, nil
}

// Start 启动独立 SSE 服务，进程内模式由 HTTP 宿主负责承载。
func (s *Server) Start(ctx context.Context) error {
	if s == nil || s.Server == nil || s.InProcess {
		return nil
	}
	return s.Server.Start(ctx)
}

// Stop 停止 SSE 服务并释放连接资源。
func (s *Server) Stop(ctx context.Context) error {
	if s == nil || s.Server == nil {
		return nil
	}
	return s.Server.Stop(ctx)
}

// Endpoint 返回独立 SSE 服务的注册端点。
func (s *Server) Endpoint() (*url.URL, error) {
	if s == nil || s.Server == nil {
		return nil, errors.New("SSE服务未初始化")
	}
	return s.Server.Endpoint()
}

// Resolver 返回当前 SSE 服务实际使用的流解析器。
func (t *Server) Resolver() sse.StreamIDResolver {
	if t == nil {
		return nil
	}
	return t.resolver
}

func requestUserID(request *http.Request, authenticator engine.Authenticator, userToken *data.UserToken) (int64, error) {
	if authenticator == nil {
		return 0, nil
	}
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) == 0 {
		return 0, fmt.Errorf("SSE 请求需要认证")
	}
	if len(parts) != 2 || !strings.EqualFold(parts[0], engine.BearerWord) {
		return 0, fmt.Errorf("SSE Authorization 请求头格式错误")
	}
	claims, err := authenticator.AuthenticateToken(parts[1])
	if err != nil {
		return 0, err
	}
	var userID int64
	userID, err = claims.GetInt64(data.ClaimFieldUserID)
	if err != nil {
		return 0, err
	}
	var roleCode string
	roleCode, err = claims.GetString(data.ClaimFieldRoleCode)
	if err != nil {
		return 0, err
	}
	if roleCode == _const.BASE_ROLE_CODE_USER || roleCode == _const.BASE_ROLE_CODE_AUTHUSER {
		return 0, fmt.Errorf("SSE 仅允许后台用户访问")
	}
	if userID != 0 && userToken != nil && !userToken.IsExistAccessToken(userID) {
		return 0, fmt.Errorf("SSE 访问令牌已失效")
	}
	return userID, nil
}
