package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	coreBiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/server/requestmeta"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/go-kratos/kratos/v3/transport/http/status"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const operationFileService = "/base.v1.FileService/"

// Redacter 定义日志脱敏接口。
type Redacter interface {
	Redact() string
}

// Server 创建服务端访问日志中间件。
func Server(_ *slog.Logger,
	_ *data.BaseUserRepository,
	authenticator engine.Authenticator,
) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			startTime := time.Now()
			// 日志信息
			baseLog := coreBiz.LogEvent{
				RequestTime: startTime,
				// 默认返回码按成功初始化，后续再根据实际错误覆盖。
				StatusCode: int32(status.FromGRPCCode(codes.OK)),
			}
			// 当前上下文存在服务端传输信息时，补充访问日志的请求元数据。
			if info, ok := transport.FromServerContext(ctx); ok {
				baseLog.Operation = info.Operation()
				// 当前请求走 HTTP 传输层时，补充 HTTP 相关访问日志字段。
				if htr, htrOk := info.(*http.Transport); htrOk {
					baseLog.RequestID = getRequestID(ctx, htr.Request())
					baseLog.TraceID = requestmeta.TraceID(ctx)
					// 文件上传不存请求内容，文件上传和下载请求体通常较大，不记录请求体内容。
					if !strings.HasPrefix(htr.Operation(), operationFileService) {
						baseLog.RequestBody = extractArgs(req)
					}

					clientIP := getClientRealIP(htr.Request())
					baseLog.Method = htr.Request().Method
					baseLog.Path = htr.PathTemplate()
					baseLog.ClientIP = clientIP
					authToken := htr.RequestHeader().Get(HEADER_KEY_AUTHORIZATION)
					ut := extractAuthToken(authToken, authenticator)
					if ut != nil {
						baseLog.TenantID = ut.TenantId
						baseLog.TenantCode = ut.TenantCode
						baseLog.UserID = ut.UserId
						baseLog.UserName = ut.UserName
					}
					baseLog.UserAgent = htr.RequestHeader().Get(HEADER_KEY_USER_AGENT)
				}
			}
			reply, err = handler(ctx, req)
			baseLog.IsSuccess = err == nil
			// 当前错误可转换为 Kratos 标准错误时，补充业务错误码和原因。
			if se := errors.FromError(err); se != nil {
				baseLog.StatusCode = se.Code
				baseLog.ReasonCode = se.Reason
				baseLog.Reason = se.Reason
			}
			baseLog.CostTime = time.Since(startTime).Milliseconds()
			level := log.LevelInfo
			stack := ""
			// 请求处理失败时，将访问日志提升为错误级别并保留堆栈。
			if err != nil {
				level = log.LevelError
				stack = fmt.Sprintf("%+v", err)
			}
			// 存在堆栈信息时，追加到日志原因字段便于排查。
			if len(stack) > 0 {
				baseLog.Reason = fmt.Sprintf("[%s]%s", baseLog.Reason, stack)
			}
			// 写入日志
			if emitErr := coreBiz.EmitLog(ctx, baseLog); emitErr != nil {
				log.Error("发送审计事件失败", "error", emitErr, "operation", baseLog.Operation)
			}
			logLine := fmt.Sprintf(
				"operation=%s method=%s path=%s args=%s code=%d latency=%s",
				normalizeLogField(baseLog.Operation),
				normalizeLogField(baseLog.Method),
				normalizeLogField(baseLog.Path),
				normalizeLogField(baseLog.RequestBody),
				baseLog.StatusCode,
				fmt.Sprintf("%dms", baseLog.CostTime),
			)
			// 错误请求使用错误级别输出，便于在控制台快速筛选异常请求。
			if level == log.LevelError {
				log.Error(logLine)
			} else {
				// 非错误请求统一输出单行文本日志，避免结构化日志过于冗长。
				log.Info(logLine)
			}
			return
		}
	}
}

// marshalFallbackText 将兜底文本包装成合法 JSON 字符串。
func marshalFallbackText(text string) string {
	textBytes, err := json.Marshal(text)
	if err != nil {
		return text
	}
	return string(textBytes)
}

// extractArgs 提取请求体日志内容。
func extractArgs(req interface{}) string {
	requestBody, err := marshalRequestBody(req)
	// 请求对象能正常序列化时，统一按 JSON 写入日志，便于后台直接格式化展示。
	if err == nil {
		return string(redactLogJSON(requestBody))
	}

	// 请求对象实现脱敏接口但无法直接序列化时，回退记录脱敏后的 JSON 字符串。
	if redacter, ok := req.(Redacter); ok {
		return marshalFallbackText(redacter.Redact())
	}
	// 请求对象实现 Stringer 时，回退复用其字符串表示。
	if stringer, ok := req.(fmt.Stringer); ok {
		return marshalFallbackText(stringer.String())
	}
	return marshalFallbackText(fmt.Sprintf("%+v", req))
}

// marshalRequestBody 将请求对象统一序列化成 JSON。
func marshalRequestBody(req interface{}) ([]byte, error) {
	// 空请求统一写成空对象，避免日志字段出现空串。
	if req == nil {
		return []byte("{}"), nil
	}

	// Proto 请求优先使用 protojson，并与 HTTP 接口统一使用 Proto 字段名。
	if message, ok := req.(proto.Message); ok {
		return protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		}.Marshal(message)
	}

	return json.Marshal(req)
}

// normalizeLogField 将日志字段压缩成单行文本。
func normalizeLogField(value string) string {
	// 空值字段统一输出占位符，避免控制台日志字段缺失。
	if value == "" {
		return "-"
	}
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.Join(strings.Fields(value), " ")
}
