package sse

import (
	"context"
	"net/http"
	"strings"
	"sync"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-core/pkg/biz"
	_const "github.com/liujitcn/kratos-core/pkg/const"
	"github.com/liujitcn/kratos-core/pkg/errorsx"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

// SSE 实现 Core 的 SSE 订阅与发布能力。
type SSE struct {
	mu            sync.RWMutex
	authenticator authnEngine.Authenticator
	userToken     *authData.UserToken
	server        *sseServer.Server
	registry      *Registry
	publisher     *Publisher
}

var _ biz.SSE = (*SSE)(nil)

// NewSSE 创建 SSE 服务。
func NewSSE(authenticator authnEngine.Authenticator, userToken *authData.UserToken, registry *Registry) *SSE {
	return &SSE{authenticator: authenticator, userToken: userToken, registry: registry}
}

// Serve 校验当前 HTTP 请求并建立指定业务流的 SSE 订阅。
func (r *SSE) Serve(ctx context.Context, streamID, channelID string) error {
	server, registry, _ := r.snapshot()
	if server == nil || registry == nil {
		return errorsx.Internal("SSE服务未初始化")
	}
	w, ok := kratosHTTP.ResponseWriterFromServerContext(ctx)
	if !ok || w == nil {
		return errorsx.InvalidArgument("SSE订阅仅支持HTTP访问")
	}
	var request *http.Request
	request, ok = kratosHTTP.RequestFromServerContext(ctx)
	if !ok || request == nil {
		return errorsx.InvalidArgument("SSE订阅仅支持HTTP访问")
	}
	userToken, err := r.authenticate(request)
	if err != nil || userToken.RoleCode == _const.BASE_ROLE_CODE_USER || userToken.RoleCode == _const.BASE_ROLE_CODE_AUTHUSER {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	var transportID string
	var found bool
	transportID, found, err = registry.Resolve(streamID, channelID, userToken.UserId)
	if !found {
		return errorsx.InvalidArgument("SSE流不支持")
	}
	if err != nil {
		return err
	}
	server.ServeStreamHTTP(w, request, sseServer.StreamID(transportID))
	return nil
}

// PublishEnabled 判断当前运行时是否已启用 SSE 消息发布。
func (r *SSE) PublishEnabled() bool {
	_, _, publisher := r.snapshot()
	return publisher != nil
}

// PublishJSON 尽力向指定 SSE 流发布结构化消息。
func (r *SSE) PublishJSON(ctx context.Context, streamID, eventID string, payload any) {
	_, _, publisher := r.snapshot()
	if publisher != nil {
		publisher.TryPublishJSON(ctx, streamID, eventID, payload)
	}
}

// BindTransport 绑定应用启动后创建的 SSE 传输实现。
func (r *SSE) BindTransport(server *sseServer.Server) {
	r.mu.Lock()
	r.server = server
	r.publisher = NewPublisher(server)
	r.mu.Unlock()
}

// UnbindTransport 清除已经停止的 SSE 传输实现。
func (r *SSE) UnbindTransport() {
	r.mu.Lock()
	r.server = nil
	r.publisher = nil
	r.mu.Unlock()
}

// authenticate 从 SSE HTTP 请求中解析并校验后台用户令牌。
func (r *SSE) authenticate(request *http.Request) (*authData.UserTokenPayload, error) {
	if r.authenticator == nil || r.userToken == nil {
		return nil, errorsx.Unauthenticated("SSE认证未配置")
	}
	token := request.Header.Get(authnEngine.HeaderAuthorize)
	if token == "" {
		return nil, errorsx.Unauthenticated("SSE访问令牌为空")
	}
	if len(token) >= len(authnEngine.BearerWord)+1 && strings.EqualFold(token[:len(authnEngine.BearerWord)+1], authnEngine.BearerWord+" ") {
		token = token[len(authnEngine.BearerWord)+1:]
	}
	authClaims, err := r.authenticator.AuthenticateToken(token)
	if err != nil || authClaims == nil {
		return nil, errorsx.Unauthenticated("SSE访问令牌无效").WithCause(err)
	}
	userToken := &authData.UserTokenPayload{}
	if err = userToken.ExtractAuthClaims(authClaims); err != nil || userToken.UserId == 0 {
		return nil, errorsx.Unauthenticated("SSE访问令牌无效").WithCause(err)
	}
	if !r.userToken.IsExistAccessToken(userToken.UserId) {
		return nil, errorsx.Unauthenticated("SSE访问令牌已失效")
	}
	return userToken, nil
}

// snapshot 获取当前 SSE 传输实现的并发安全快照。
func (r *SSE) snapshot() (*sseServer.Server, *Registry, *Publisher) {
	r.mu.RLock()
	server := r.server
	registry := r.registry
	publisher := r.publisher
	r.mu.RUnlock()
	return server, registry, publisher
}
