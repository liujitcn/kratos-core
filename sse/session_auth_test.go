package sse

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// sessionTestAuthenticator 保存签发声明，以复现真实会话缓存与 SSE 认证入口。
type sessionTestAuthenticator struct {
	engine.Authenticator
	claims map[string]engine.AuthClaims
}

// CreateIdentity 创建包含唯一标识的测试令牌。
func (a *sessionTestAuthenticator) CreateIdentity(claims engine.AuthClaims) (string, error) {
	raw, err := json.Marshal(claims)
	token := string(raw)
	a.claims[token] = claims
	return token, err
}

// AuthenticateToken 返回令牌对应的认证声明。
func (a *sessionTestAuthenticator) AuthenticateToken(token string) (*engine.AuthClaims, error) {
	claims := a.claims[token]
	return &claims, nil
}

// TestRevokedSessionCannotSubscribe 验证另一设备在线时，被撤销的令牌不能通过两个 SSE 入口。
func TestRevokedSessionCannotSubscribe(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	authenticator := &sessionTestAuthenticator{claims: make(map[string]engine.AuthClaims)}
	tokens := data.NewUserToken(store, authenticator, "access:", "refresh:", time.Hour, 24*time.Hour)
	user := &data.UserTokenPayload{UserId: 1, TenantId: 1, RoleCode: "super", UserName: "alice"}
	var firstAccess, firstRefresh string
	firstAccess, firstRefresh, err = tokens.GenerateTokenForSession(user, "first")
	if err != nil {
		t.Fatal(err)
	}
	var secondAccess string
	secondAccess, _, err = tokens.GenerateTokenForSession(user, "second")
	if err != nil {
		t.Fatal(err)
	}
	if err = tokens.RemoveTokenForSession(1, "first", firstAccess, firstRefresh); err != nil {
		t.Fatal(err)
	}
	resolver := NewStreamResolver(nil, authenticator, tokens)
	runtime := &SSE{authenticator: authenticator, userToken: tokens}
	for _, test := range []struct {
		name  string
		token string
		valid bool
	}{{"已撤销设备", firstAccess, false}, {"仍在线设备", secondAccess, true}} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/events?stream=base.notification", nil)
			request.Header.Set("Authorization", "Bearer "+test.token)
			_, err := resolver(request)
			if (err == nil) != test.valid {
				t.Errorf("直接订阅入口校验结果不符: %v", err)
			}
			_, err = runtime.authenticate(request)
			if (err == nil) != test.valid {
				t.Errorf("Base SSE 入口校验结果不符: %v", err)
			}
		})
	}
}
