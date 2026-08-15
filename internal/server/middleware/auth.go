package middleware

import (
	"context"
	"errors"
	stdhttp "net/http"
	"regexp"
	"strings"

	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authkit "github.com/liujitcn/kratos-kit/auth"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authnMiddleware "github.com/liujitcn/kratos-kit/auth/authn/middleware"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authzMiddleware "github.com/liujitcn/kratos-kit/auth/authz/middleware"
	authData "github.com/liujitcn/kratos-kit/auth/data"
)

const fallbackAuthAction = "ANY"

type httpRequestTransport interface {
	Request() *stdhttp.Request
}

type methodTransport interface {
	Method() string
}

// NewAuthMiddleware 创建使用真实请求方式的统一鉴权中间件。
func NewAuthMiddleware(
	authenticator authnEngine.Authenticator,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	cfg *configv1.Authentication_Jwt,
) kratosMiddleware.Middleware {
	fullAuth := kratosMiddleware.Chain(
		authnMiddleware.Server(authenticator, authnMiddleware.WithAuthErrorMapper(mapAuthnError)),
		authClaimsMiddleware(userToken),
		authzMiddleware.Server(authorizer),
	)

	optionalAuth := authkit.OptionalServer(authenticator, userToken)
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		fullAuthHandler := fullAuth(handler)
		optionalAuthHandler := optionalAuth(handler)
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			serverTransport, ok := transport.FromServerContext(ctx)
			if !ok {
				// 无法识别请求元信息时，回退到完整鉴权链路。
				return fullAuthHandler(ctx, req)
			}

			operation := serverTransport.Operation()
			if matchWhiteList(cfg.GetOptionalAuth(), operation) {
				// 可选鉴权接口只解析身份，不强制拦截未登录请求。
				return optionalAuthHandler(ctx, req)
			}
			if matchWhiteList(cfg.GetWhiteList(), operation) {
				// 白名单接口直接透传给业务处理器。
				return handler(ctx, req)
			}
			return fullAuthHandler(ctx, req)
		}
	}
}

// authClaimsMiddleware 将认证声明转换为 Casbin 鉴权声明。
func authClaimsMiddleware(userToken *authData.UserToken) kratosMiddleware.Middleware {
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			serverTransport, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, authkit.ErrWrongContext
			}

			authnClaims, ok := authnMiddleware.FromContext(ctx)
			if !ok {
				return nil, authkit.ErrWrongContext
			}

			var err error
			err = verifyAccessToken(userToken, authnClaims)
			if err != nil {
				return nil, err
			}

			var tenantCode string
			tenantCode, err = authnClaims.GetString(authData.ClaimFieldTenantCode)
			if err != nil || tenantCode == "" {
				return nil, authkit.ErrExtractTenantFailed
			}
			var roleCode string
			roleCode, err = authnClaims.GetString(authData.ClaimFieldRoleCode)
			if err != nil || roleCode == "" {
				return nil, authkit.ErrExtractSubjectFailed
			}

			action := requestAction(serverTransport)
			authzClaims := authzEngine.AuthClaims{
				Tenant:   new(authzEngine.Tenant(tenantCode)),
				Subject:  new(authzEngine.Subject(roleCode)),
				Action:   &action,
				Resource: new(authzEngine.Resource(serverTransport.Operation())),
			}

			ctx = authzMiddleware.NewContext(ctx, &authzClaims)
			return handler(ctx, req)
		}
	}
}

// requestAction 读取真实 HTTP 请求方式，非 HTTP 场景回退到 ANY。
func requestAction(serverTransport transport.Transporter) authzEngine.Action {
	method := ""
	if htr, ok := serverTransport.(httpRequestTransport); ok && htr.Request() != nil {
		method = htr.Request().Method
	}
	if method == "" {
		if mtr, ok := serverTransport.(methodTransport); ok {
			method = mtr.Method()
		}
	}
	if method == "" {
		method = fallbackAuthAction
	}
	return authzEngine.Action(strings.ToUpper(method))
}

// verifyAccessToken 校验访问令牌仍在缓存有效期内。
func verifyAccessToken(userToken *authData.UserToken, authnClaims *authnEngine.AuthClaims) error {
	userID, err := authnClaims.GetInt64(authData.ClaimFieldUserID)
	if err != nil {
		return authkit.ErrExtractUserInfoFailed
	}
	// 用户 id 为 0 表示内部调用，跳过用户令牌缓存校验。
	if userID == 0 {
		return nil
	}
	if !userToken.IsExistAccessToken(userID) {
		return authkit.ErrAccessTokenExpired
	}
	return nil
}

// mapAuthnError 将底层认证错误转换为对外稳定的访问令牌错误。
func mapAuthnError(err error) error {
	if errors.Is(err, authnEngine.ErrMissingBearerToken) {
		return authkit.ErrAccessTokenNotExist
	}
	if errors.Is(err, authnEngine.ErrTokenExpired) {
		return authkit.ErrAccessTokenExpired
	}
	return authnMiddleware.ErrUnauthorized
}

// matchWhiteList 判断指定 operation 是否命中鉴权白名单。
func matchWhiteList(whiteList *configv1.Authentication_Jwt_WhiteList, operation string) bool {
	if whiteList == nil {
		return false
	}
	for _, prefix := range whiteList.Prefix {
		if strings.HasPrefix(operation, prefix) {
			return true
		}
	}
	for _, regexValue := range whiteList.Regex {
		regex, err := regexp.Compile(regexValue)
		if err != nil {
			continue
		}
		if regex.FindString(operation) == operation {
			return true
		}
	}
	for _, path := range whiteList.Path {
		if path == operation {
			return true
		}
	}
	for _, item := range whiteList.Match {
		if item == operation {
			return true
		}
	}
	return false
}
