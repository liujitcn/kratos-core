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

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/server/requestmeta"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authnMiddleware "github.com/liujitcn/kratos-kit/auth/authn/middleware"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authzMiddleware "github.com/liujitcn/kratos-kit/auth/authz/middleware"
	"github.com/liujitcn/kratos-kit/auth/data"
)

const (
	fallbackAuthAction  = "ANY"
	policyDecisionAllow = int32(1)
	policyDecisionDeny  = int32(2)
	policyDecisionError = int32(3)
)

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

// auditedAuthzMiddleware 使用授权引擎装饰器记录每次真实策略评估。
func auditedAuthzMiddleware(authorizer authzEngine.Engine) middleware.Middleware {
	return authzMiddleware.Server(&auditedAuthorizer{authorizer: authorizer})
}

// auditedAuthorizer 只装饰授权引擎调用，不把后续业务处理计入策略耗时。
type auditedAuthorizer struct {
	authorizer authzEngine.Engine
}

// Name 返回实际授权引擎名称。
func (a *auditedAuthorizer) Name() string {
	return a.authorizer.Name()
}

// ProjectsAuthorized 透传项目批量授权查询。
func (a *auditedAuthorizer) ProjectsAuthorized(ctx context.Context, subjects authzEngine.Subjects, action authzEngine.Action, resource authzEngine.Resource, projects authzEngine.Projects) (authzEngine.Projects, error) {
	return a.authorizer.ProjectsAuthorized(ctx, subjects, action, resource, projects)
}

// FilterAuthorizedPairs 透传资源动作对批量过滤。
func (a *auditedAuthorizer) FilterAuthorizedPairs(ctx context.Context, subjects authzEngine.Subjects, pairs authzEngine.Pairs) (authzEngine.Pairs, error) {
	return a.authorizer.FilterAuthorizedPairs(ctx, subjects, pairs)
}

// FilterAuthorizedProjects 透传项目授权过滤。
func (a *auditedAuthorizer) FilterAuthorizedProjects(ctx context.Context, subjects authzEngine.Subjects) (authzEngine.Projects, error) {
	return a.authorizer.FilterAuthorizedProjects(ctx, subjects)
}

// IsAuthorized 记录单次真实授权引擎调用及其耗时。
func (a *auditedAuthorizer) IsAuthorized(ctx context.Context, subject authzEngine.Subject, action authzEngine.Action, resource authzEngine.Resource, project authzEngine.Project) (bool, error) {
	startedAt := time.Now()
	allowed, err := a.authorizer.IsAuthorized(ctx, subject, action, resource, project)
	a.emitPolicyEvaluation(ctx, startedAt, subject, action, resource, project, allowed, err)
	return allowed, err
}

// emitPolicyEvaluation 非阻塞投递单次策略评估事实，投递失败不改变授权结果。
func (a *auditedAuthorizer) emitPolicyEvaluation(ctx context.Context, startedAt time.Time, subject authzEngine.Subject, action authzEngine.Action, resource authzEngine.Resource, project authzEngine.Project, allowed bool, evaluationErr error) {
	decision := policyDecisionAllow
	statusCode := int32(http.StatusOK)
	reasonCode := ""
	reason := ""
	isSuccess := true
	if evaluationErr != nil {
		decision = policyDecisionError
		statusCode = int32(http.StatusInternalServerError)
		reasonCode = "INTERNAL_ERROR"
		reason = evaluationErr.Error()
		isSuccess = false
	} else if !allowed {
		decision = policyDecisionDeny
		statusCode = int32(http.StatusForbidden)
		reasonCode = "PERMISSION_DENIED"
		reason = "权限策略拒绝访问"
	}
	event := coreBiz.LogEvent{
		Kind: "policy_evaluation", Engine: a.authorizer.Name(), EvaluationType: 1,
		RequestID: requestmeta.RequestID(ctx), TraceID: requestmeta.TraceID(ctx),
		Resource: string(resource), Action: string(action), Project: string(project), Decision: decision,
		StatusCode: statusCode, IsSuccess: isSuccess, DurationMs: int32(time.Since(startedAt).Milliseconds()),
		RequestTime: startedAt, ReasonCode: reasonCode, Reason: reason, RoleCode: string(subject),
	}
	if serverTransport, ok := transport.FromServerContext(ctx); ok {
		event.Operation = serverTransport.Operation()
		if requestTransport, ok := serverTransport.(httpRequestTransport); ok && requestTransport.Request() != nil {
			request := requestTransport.Request()
			event.Method = request.Method
			event.Path = request.URL.Path
			event.ClientIP = request.RemoteAddr
			if host, _, splitErr := net.SplitHostPort(event.ClientIP); splitErr == nil {
				event.ClientIP = host
			}
			event.UserAgent = request.UserAgent()
		}
	}
	if claims, ok := authnMiddleware.FromContext(ctx); ok {
		event.UserID, _ = claims.GetInt64(data.ClaimFieldUserID)
		event.UserName, _ = claims.GetSubject()
		event.TenantID, _ = claims.GetInt64(data.ClaimFieldTenantID)
		event.TenantCode, _ = claims.GetString(data.ClaimFieldTenantCode)
		event.RoleID, _ = claims.GetInt64(data.ClaimFieldRoleID)
	}
	if claims, ok := authzEngine.AuthClaimsFromContext(ctx); ok && claims.Tenant != nil {
		event.TenantCode = string(*claims.Tenant)
	}
	if emitErr := coreBiz.EmitLog(ctx, event); emitErr != nil {
		log.Error("发送授权审计事件失败", "error", emitErr, "operation", event.Operation)
	}
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
