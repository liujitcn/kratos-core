package sse

import (
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"

	_const "github.com/liujitcn/kratos-core/pkg/const"
	"github.com/liujitcn/kratos-core/pkg/module"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
	"google.golang.org/protobuf/proto"
)

var errStreamNotRegistered = errors.New("SSE流未注册")

// Transport 描述 Core SSE transport 服务及其传输模式。
type Transport struct {
	// Server 是 Core 创建的底层 SSE 服务。
	Server *sseServer.Server
	// InProcess 表示 SSE 是否挂载到 Core HTTP 服务。
	InProcess bool
	resolver  sseServer.StreamIDResolver
}

// NewStreamResolver 创建根据模块流声明解析实际传输流的请求解析器。
func NewStreamResolver(registry *Registry, authenticator authnEngine.Authenticator, userToken *authData.UserToken) sseServer.StreamIDResolver {
	return func(request *stdhttp.Request) (string, error) {
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

// NewTransport 按 Server_Sse 配置创建 SSE transport，并调用模块注册业务流。
func NewTransport(ctx *bootstrap.Context, resolver sseServer.StreamIDResolver, modules module.Modules, registry *Registry) (*Transport, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Sse == nil {
		return nil, nil
	}
	options := make([]sseServer.ServerOption, 0, 2)
	var server *sseServer.Server
	transportResolver := resolver
	if resolver != nil {
		transportResolver = func(request *stdhttp.Request) (string, error) {
			streamID, err := resolver(request)
			if err == nil || !errors.Is(err, errStreamNotRegistered) || server == nil || request == nil || request.URL == nil {
				return streamID, err
			}
			directStreamID := request.URL.Query().Get("stream")
			if directStreamID != "" && server.GetStream(sseServer.StreamID(directStreamID)) != nil {
				return directStreamID, nil
			}
			return streamID, err
		}
		options = append(options, sseServer.WithStreamIDResolver(transportResolver), sseServer.WithAutoStream(true))
	}
	inProcess := cfg.Server.Sse.GetTransport() == bootstrapConfigv1.Server_Sse_IN_PROCESS
	var err error
	if inProcess {
		server, err = rpc.CreateSseHandler(cfg, options...)
	} else {
		// 独立模式也延迟到 Kratos Start 阶段监听，避免后续 Provider 失败时泄漏监听器。
		handlerConfig := proto.Clone(cfg).(*bootstrapConfigv1.Bootstrap)
		handlerConfig.Server.Sse.Transport = bootstrapConfigv1.Server_Sse_IN_PROCESS
		server, err = rpc.CreateSseHandler(handlerConfig, options...)
	}
	if err != nil {
		return nil, err
	}
	if err = modules.RegisterSSE(server); err != nil {
		return nil, err
	}
	if registry != nil {
		if err = registry.Register(server.StreamDefinitions()...); err != nil {
			return nil, err
		}
	}
	return &Transport{Server: server, InProcess: inProcess, resolver: transportResolver}, nil
}

// Resolver 返回当前 SSE transport 实际使用的流解析器。
func (t *Transport) Resolver() sseServer.StreamIDResolver {
	if t == nil {
		return nil
	}
	return t.resolver
}

func requestUserID(request *stdhttp.Request, authenticator authnEngine.Authenticator, userToken *authData.UserToken) (int64, error) {
	if authenticator == nil {
		return 0, nil
	}
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) == 0 {
		return 0, fmt.Errorf("SSE 请求需要认证")
	}
	if len(parts) != 2 || !strings.EqualFold(parts[0], authnEngine.BearerWord) {
		return 0, fmt.Errorf("SSE Authorization 请求头格式错误")
	}
	claims, err := authenticator.AuthenticateToken(parts[1])
	if err != nil {
		return 0, err
	}
	var userID int64
	userID, err = claims.GetInt64(authData.ClaimFieldUserID)
	if err != nil {
		return 0, err
	}
	var roleCode string
	roleCode, err = claims.GetString(authData.ClaimFieldRoleCode)
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
