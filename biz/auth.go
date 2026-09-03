package biz

import (
	"context"
	"fmt"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/authn/engine/jwt"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine/casbin"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/sdk"
)

const jwtKeyName = "kratos-kit:authn/jwt"

// NewAuthenticator 创建认证器。
func NewAuthenticator(cfg *configv1.Authentication_Jwt) (engine.Authenticator, error) {
	if cfg == nil {
		return nil, nil
	}
	secret := cfg.GetSecret()
	if secret == "" {
		keyValue := sdk.Runtime.GetKey()
		if keyValue == nil {
			return nil, fmt.Errorf("JWT密钥为空且运行时密钥未初始化")
		}
		var err error
		var derived []byte
		derived, err = keyValue.Derive(context.Background(), jwtKeyName)
		if err != nil {
			return nil, fmt.Errorf("派生 JWT 密钥失败: %w", err)
		}
		secret = string(derived)
	}
	return jwt.NewAuthenticator(
		jwt.WithKey([]byte(secret)),
		jwt.WithSigningMethod(cfg.GetMethod()),
	)
}

// NewAuthzEngine 创建鉴权引擎
func NewAuthzEngine() (authzEngine.Engine, error) {
	return casbin.NewEngine(context.Background())
}

// NewUserToken 创建用户令牌管理器。
func NewUserToken(cfg *configv1.Authentication_Jwt, cache cache.Cache, authenticator engine.Authenticator) *data.UserToken {
	if cfg == nil || cache == nil || authenticator == nil {
		return nil
	}
	const (
		// USER_ACCESS_TOKEN_KEY_PREFIX 表示用户访问令牌缓存前缀。
		USER_ACCESS_TOKEN_KEY_PREFIX = "uat_"
		// USER_REFRESH_TOKEN_KEY_PREFIX 表示用户刷新令牌缓存前缀。
		USER_REFRESH_TOKEN_KEY_PREFIX = "urt_"
	)
	return data.NewUserToken(
		cache,
		authenticator,
		USER_ACCESS_TOKEN_KEY_PREFIX,
		USER_REFRESH_TOKEN_KEY_PREFIX,
		cfg.GetAccessTokenExpires().AsDuration(),
		cfg.GetRefreshTokenExpires().AsDuration(),
	)
}
