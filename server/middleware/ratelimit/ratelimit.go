package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net"
	"strconv"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/cache"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Dimension 标识接口限流使用的身份维度。
type Dimension string

const (
	// DimensionGlobal 将同一接口的所有请求共享一个令牌桶。
	DimensionGlobal Dimension = "GLOBAL"
	// DimensionIP 按网络对端 IP 地址分别限流。
	DimensionIP Dimension = "IP"
	// DimensionTenant 按认证租户分别限流。
	DimensionTenant Dimension = "TENANT"
	// DimensionUser 按认证用户分别限流。
	DimensionUser Dimension = "USER"
	// DimensionOauthClient 按 OAuth 客户端分别限流。
	DimensionOauthClient Dimension = "OAUTH_CLIENT"
)

// Policy 描述一个接口限流维度及其令牌桶参数。
type Policy struct {
	// Dimension 是限流身份维度。
	Dimension Dimension
	// TokensPerSecond 是令牌生成速率。
	TokensPerSecond float64
	// Burst 是允许的突发请求数。
	Burst int
}

// PolicyResolver 按完整 RPC 操作名读取当前启用的限流策略。
type PolicyResolver interface {
	Resolve(context.Context, string) ([]Policy, error)
}

// NewMiddleware 创建按接口策略执行原子令牌桶限流的服务端中间件。
func NewMiddleware(store cache.Cache, resolver PolicyResolver) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			if store == nil || resolver == nil {
				return next(ctx, request)
			}
			serverTransport, ok := transport.FromServerContext(ctx)
			if !ok || serverTransport.Operation() == "" {
				return next(ctx, request)
			}
			policies, err := resolver.Resolve(ctx, serverTransport.Operation())
			if err != nil {
				return nil, status.Error(codes.Unavailable, "rate limit policy is unavailable")
			}
			if len(policies) == 0 {
				return next(ctx, request)
			}
			buckets := make([]cache.TokenBucketRequest, 0, len(policies))
			for _, policy := range policies {
				identity, identityErr := policyIdentity(ctx, request, serverTransport, policy.Dimension)
				if identityErr != nil || policy.TokensPerSecond <= 0 || math.IsNaN(policy.TokensPerSecond) || math.IsInf(policy.TokensPerSecond, 0) || policy.Burst <= 0 {
					return nil, status.Error(codes.Unavailable, "rate limit policy cannot be evaluated")
				}
				buckets = append(buckets, cache.TokenBucketRequest{
					Key:             bucketKey(serverTransport.Operation(), policy.Dimension, identity),
					TokensPerSecond: policy.TokensPerSecond,
					Burst:           policy.Burst,
				})
			}
			allowed, retryAfter, err := store.TakeTokenBuckets(buckets)
			if err != nil {
				return nil, status.Error(codes.Unavailable, "rate limit storage is unavailable")
			}
			if !allowed {
				seconds := int64(math.Ceil(retryAfter.Seconds()))
				if seconds < 1 {
					seconds = 1
				}
				serverTransport.ReplyHeader().Set("Retry-After", strconv.FormatInt(seconds, 10))
				return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
			}
			return next(ctx, request)
		}
	}
}

// policyIdentity 从当前请求读取指定限流维度对应的稳定身份。
func policyIdentity(ctx context.Context, request any, serverTransport transport.Transporter, dimension Dimension) (string, error) {
	switch dimension {
	case DimensionGlobal:
		return "global", nil
	case DimensionIP:
		identity := remoteIP(ctx)
		if identity != "" {
			return identity, nil
		}
	case DimensionTenant, DimensionUser, DimensionOauthClient:
		if dimension == DimensionOauthClient {
			if clientRequest, ok := request.(interface{ GetClientId() string }); ok && clientRequest.GetClientId() != "" {
				return clientRequest.GetClientId(), nil
			}
		}
		identity, err := auth.FromContext(ctx)
		if err != nil {
			return "", err
		}
		if identity == nil {
			return "", status.Error(codes.Unavailable, "rate limit identity is unavailable")
		}
		switch dimension {
		case DimensionTenant:
			if identity.TenantId > 0 {
				return strconv.FormatInt(identity.TenantId, 10), nil
			}
		case DimensionUser:
			if identity.UserId != 0 {
				return strconv.FormatInt(identity.UserId, 10), nil
			}
		case DimensionOauthClient:
			if identity.UserId < 0 && identity.UserCode != "" {
				return identity.UserCode, nil
			}
		}
	}
	return "", status.Error(codes.Unavailable, "rate limit identity is unavailable")
}

// remoteIP 读取 HTTP 或 gRPC 网络对端地址，不信任转发请求头。
func remoteIP(ctx context.Context) string {
	if request, ok := kratosHTTP.RequestFromServerContext(ctx); ok && request != nil {
		return hostIP(request.RemoteAddr)
	}
	if remote, ok := peer.FromContext(ctx); ok && remote.Addr != nil {
		return hostIP(remote.Addr.String())
	}
	return ""
}

// bucketKey 为 Redis Cluster 生成同槽位且不暴露请求身份的限流键。
func bucketKey(operation string, dimension Dimension, identity string) string {
	operationHash := sha256.Sum256([]byte(operation))
	identityHash := sha256.Sum256([]byte(identity))
	return "api-rate-limit:{" + hex.EncodeToString(operationHash[:12]) + "}:" + string(dimension) + ":" + hex.EncodeToString(identityHash[:])
}

// hostIP 提取 IP 地址，避免把端口并入限流身份。
func hostIP(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		if net.ParseIP(host) != nil {
			return host
		}
		return ""
	}
	if net.ParseIP(address) != nil {
		return address
	}
	return ""
}
