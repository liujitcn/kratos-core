package logging

const (
	// HEADER_KEY_USER_AGENT 表示 User-Agent 请求头。
	HEADER_KEY_USER_AGENT = "User-Agent"
	// HEADER_KEY_REFERER 表示 Referer 请求头。
	HEADER_KEY_REFERER = "Referer"
	// HEADER_KEY_AUTHORIZATION 表示 Authorization 请求头。
	HEADER_KEY_AUTHORIZATION = "Authorization"

	// HEADER_KEY_X_REQUEST_ID 表示请求标识头。
	HEADER_KEY_X_REQUEST_ID = "X-Request-id"
	// HEADER_KEY_X_FC_REQUEST_ID 表示函数计算请求标识头。
	HEADER_KEY_X_FC_REQUEST_ID = "x-fc-request-id"
	// HEADER_KEY_X_CORRELATION_ID 表示链路关联标识头。
	HEADER_KEY_X_CORRELATION_ID = "X-Correlation-ID"
	// HEADER_KEY_X_FORWARDED_FOR 表示代理转发客户端地址头。
	HEADER_KEY_X_FORWARDED_FOR = "X-Forwarded-For"
	// HEADER_KEY_X_REAL_IP 表示真实客户端地址头。
	HEADER_KEY_X_REAL_IP = "X-Real-IP"
	// HEADER_KEY_X_CLIENT_IP 表示客户端地址头。
	HEADER_KEY_X_CLIENT_IP = "X-Client-IP"
)
