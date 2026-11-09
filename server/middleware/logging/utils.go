package logging

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/liujitcn/go-utils/geoip/qqwry"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/transport/clientip"

	"github.com/liujitcn/kratos-core/server/requestmeta"
)

var ipClient = qqwry.NewClient()

// extractAuthToken 从JWT Token中提取用户信息
func extractAuthToken(authToken string, authenticator engine.Authenticator) *data.UserTokenPayload {
	// 认证头为空时，无法继续提取用户信息。
	if len(authToken) == 0 {
		return nil
	}

	jwtToken := strings.TrimPrefix(authToken, "Bearer ")
	authnClaims, _ := authenticator.AuthenticateToken(jwtToken)
	// 令牌校验失败时，直接返回空用户信息。
	if authnClaims == nil {
		return nil
	}

	ut, _ := data.NewUserTokenPayloadWithClaims(authnClaims)
	// Claims 无法转换成用户载荷时，直接返回空结果。
	if ut == nil {
		return nil
	}

	return ut
}

// getClientRealIP 与登录及业务审计共用受信代理解析规则，未配置时只使用 TCP 对端。
func getClientRealIP(request *http.Request) string {
	if request == nil {
		return ""
	}
	return clientip.FromRequest(request)
}

// getRequestID 获取请求 ID
func getRequestID(ctx context.Context, request *http.Request) string {
	// 先读取统一请求上下文，保证 API、鉴权和业务审计复用同一请求编号。
	if requestID := requestmeta.RequestID(ctx); requestID != "" {
		return requestID
	}

	// 请求对象为空时，无法继续解析请求标识。
	if request == nil {
		return ""
	}

	// 先检查 X-Request-ID 头。这是比较常见的用于标识请求的自定义头部字段。例如，在一个微服务架构的系统中，当一个请求从前端应用发送到后端的多个微服务时，每个微服务都可以在 X-Request-ID 字段中获取到相同的请求标识，从而方便追踪请求在各个服务节点中的处理情况。
	xri := request.Header.Get(HEADER_KEY_X_REQUEST_ID)
	// 显式传入 X-Request-ID 时，优先使用该请求标识。
	if xri != "" {
		return xri
	}

	// 接着检查 X-Correlation-ID 头。它和 X-Request-ID 类似，用于关联一系列相关的请求或者事务。比如，在一个包含多个子请求的复杂业务流程中，X-Correlation-ID 可以用于跟踪整个业务流程中各个子请求之间的关系。
	xci := request.Header.Get(HEADER_KEY_X_CORRELATION_ID)
	// 业务链路存在关联标识时，回退使用该请求标识。
	if xci != "" {
		return xci
	}

	// 函数计算的请求 ID
	xfcri := request.Header.Get(HEADER_KEY_X_FC_REQUEST_ID)
	// 云函数透传请求标识存在时，回退使用该请求标识。
	if xfcri != "" {
		return xfcri
	}

	return ""
}

// clientIPToLocation 获取客户端 IP的地理位置
func clientIPToLocation(ip string) string {
	// IP 为空时，无法继续做地理位置解析。
	if ip == "" {
		return ""
	}
	// 内网地址统一标记为内网 IP，不再访问地理库。
	if qqwry.IsPrivateIP(ip) {
		return "内网IP"
	}
	res, err := ipClient.Query(ip)
	if err != nil {
		return ""
	}
	return res.City
}

// getIPFromRemoteAddr 从 RemoteAddr 中提取客户端 IP。
func getIPFromRemoteAddr(hostAddress string) string {
	// RemoteAddr 可能携带端口，需要先拆分 host 部分。
	if strings.Contains(hostAddress, ":") {
		// 能正常拆分 host:port 时，优先校验拆分后的 host。
		host, _, err := net.SplitHostPort(hostAddress)
		// host:port 拆分成功时，继续校验 host 是否为合法 IP。
		if err == nil {
			// 只有合法 IP 才允许作为客户端地址返回。
			if net.ParseIP(host) != nil {
				return host
			}
		}
	}
	// 未携带端口时，直接校验原始地址是否为合法 IP，原始地址本身就是合法 IP 时直接返回。
	if net.ParseIP(hostAddress) != nil {
		return hostAddress
	}
	return ""
}
