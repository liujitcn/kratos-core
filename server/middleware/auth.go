package middleware

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/audit"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authnMiddleware "github.com/liujitcn/kratos-kit/auth/authn/middleware"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authzMiddleware "github.com/liujitcn/kratos-kit/auth/authz/middleware"
	"github.com/liujitcn/kratos-kit/auth/data"
)

const fallbackAuthAction = "ANY"

type httpRequestTransport interface {
	Request() *http.Request
}

type methodTransport interface {
	Method() string
}

// NewAuthMiddleware 创建使用真实请求方式的统一鉴权中间件。
func NewAuthMiddleware(
	authenticator engine.Authenticator,
	authorizer authzEngine.Engine,
	userToken *data.UserToken,
	cfg *configv1.Authentication_Jwt,
) middleware.Middleware {
	fullAuth := middleware.Chain(
		authnMiddleware.Server(authenticator, authnMiddleware.WithAuthErrorMapper(mapAuthnError)),
		authClaimsMiddleware(userToken),
		auditedAuthzMiddleware(authorizer),
	)

	optionalAuth := auth.OptionalServer(authenticator, userToken)
	return func(handler middleware.Handler) middleware.Handler {
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

// auditedAuthzMiddleware 记录授权引擎允许、拒绝和异常决策。
func auditedAuthzMiddleware(authorizer authzEngine.Engine) middleware.Middleware {
	delegated := authzMiddleware.Server(authorizer)
	return func(handler middleware.Handler) middleware.Handler {
		wrapped := delegated(handler)
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			startedAt := time.Now()
			reply, err := wrapped(ctx, req)
			decision := int32(1)
			if err != nil {
				decision = 2
				if structured := kratosErrors.FromError(err); structured != nil && structured.Code >= http.StatusInternalServerError {
					decision = 3
				}
			}
			operation := ""
			requestID := ""
			method := requestActionValue(ctx)
			path := ""
			clientIP := ""
			userAgent := ""
			if serverTransport, ok := transport.FromServerContext(ctx); ok {
				operation = serverTransport.Operation()
				if requestTransport, ok := serverTransport.(httpRequestTransport); ok && requestTransport.Request() != nil {
					request := requestTransport.Request()
					requestID = request.Header.Get("X-Request-ID")
					path = request.URL.Path
					clientIP = request.RemoteAddr
					if host, _, splitErr := net.SplitHostPort(clientIP); splitErr == nil {
						clientIP = host
					}
					userAgent = request.UserAgent()
				}
			}
			event := audit.Event{
				Kind: "policy_evaluation", Engine: "casbin", EvaluationType: 1,
				RequestID: requestID, Method: method, Operation: operation, Path: path, ClientIP: clientIP, UserAgent: userAgent,
				Resource: operation, Action: method, Decision: decision, StatusCode: int32(http.StatusOK), IsSuccess: err == nil,
				DurationMs:  int32(time.Since(startedAt).Milliseconds()),
				RequestTime: startedAt,
				Reason:      errorText(err), ReasonCode: errorReason(err),
			}
			if claims, ok := authnMiddleware.FromContext(ctx); ok {
				event.UserID, _ = claims.GetInt64(data.ClaimFieldUserID)
				event.UserName, _ = claims.GetSubject()
				event.TenantID, _ = claims.GetInt64(data.ClaimFieldTenantID)
				event.TenantCode, _ = claims.GetString(data.ClaimFieldTenantCode)
				event.RoleID, _ = claims.GetInt64(data.ClaimFieldRoleID)
				event.RoleCode, _ = claims.GetString(data.ClaimFieldRoleCode)
			}
			if emitErr := audit.Emit(ctx, event); emitErr != nil {
				log.Error("发送授权审计事件失败", "error", emitErr, "operation", operation)
			}
			return reply, err
		}
	}
}

func requestActionValue(ctx context.Context) string {
	if serverTransport, ok := transport.FromServerContext(ctx); ok {
		return string(requestAction(serverTransport))
	}
	return fallbackAuthAction
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func errorReason(err error) string {
	if err == nil {
		return ""
	}
	if structured := kratosErrors.FromError(err); structured != nil {
		return structured.Reason
	}
	return "INTERNAL_ERROR"
}

// authClaimsMiddleware 将认证声明转换为 Casbin 鉴权声明。
func authClaimsMiddleware(userToken *data.UserToken) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			serverTransport, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, auth.ErrWrongContext
			}

			authnClaims, ok := authnMiddleware.FromContext(ctx)
			if !ok {
				return nil, auth.ErrWrongContext
			}

			var err error
			err = verifyAccessToken(userToken, authnClaims)
			if err != nil {
				return nil, err
			}

			var tenantCode string
			tenantCode, err = authnClaims.GetString(data.ClaimFieldTenantCode)
			if err != nil || tenantCode == "" {
				return nil, auth.ErrExtractTenantFailed
			}
			var roleCode string
			roleCode, err = authnClaims.GetString(data.ClaimFieldRoleCode)
			if err != nil || roleCode == "" {
				return nil, auth.ErrExtractSubjectFailed
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
func verifyAccessToken(userToken *data.UserToken, authnClaims *engine.AuthClaims) error {
	userID, err := authnClaims.GetInt64(data.ClaimFieldUserID)
	if err != nil {
		return auth.ErrExtractUserInfoFailed
	}
	// 用户 id 为 0 表示内部调用，跳过用户令牌缓存校验。
	if userID == 0 {
		return nil
	}
	if !userToken.IsExistAccessToken(userID) {
		return auth.ErrAccessTokenExpired
	}
	return nil
}

// mapAuthnError 将底层认证错误转换为对外稳定的访问令牌错误。
func mapAuthnError(err error) error {
	if errors.Is(err, engine.ErrMissingBearerToken) {
		return auth.ErrAccessTokenNotExist
	}
	if errors.Is(err, engine.ErrTokenExpired) {
		return auth.ErrAccessTokenExpired
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
	if slices.Contains(whiteList.Path, operation) {
		return true
	}
	return slices.Contains(whiteList.Match, operation)
}
