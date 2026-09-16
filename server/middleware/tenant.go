package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	tenantFieldName protoreflect.Name = "tenant_id"
)

// NewTenantScopeMiddleware 创建请求租户补全与跨租户输入拦截中间件。
//
// 请求消息中的 tenant_id 统一按当前认证身份处理：普通租户缺省时自动填充，
// 已传入其他租户时拒绝；默认租户保留未指定状态，交由 Proto 和业务规则决定。
func NewTenantScopeMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			message, ok := req.(proto.Message)
			if !ok || !messageHasTenantField(message.ProtoReflect(), make(map[protoreflect.FullName]struct{})) {
				return handler(ctx, req)
			}

			authInfo, err := auth.FromContext(ctx)
			if err != nil || authInfo == nil {
				return nil, errorsx.Unauthenticated("缺少有效的登录身份").WithCause(err)
			}
			err = applyTenantScope(message.ProtoReflect(), authInfo, make(map[protoreflect.FullName]struct{}))
			if err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}
	}
}

// messageHasTenantField 判断当前消息实例中是否包含需要处理的租户字段。
func messageHasTenantField(message protoreflect.Message, visited map[protoreflect.FullName]struct{}) bool {
	if !message.IsValid() {
		return false
	}
	name := message.Descriptor().FullName()
	if _, ok := visited[name]; ok {
		return false
	}
	visited[name] = struct{}{}
	defer delete(visited, name)
	fields := message.Descriptor().Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		if field.Name() == tenantFieldName {
			return true
		}
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			continue
		}
		if field.IsList() {
			items := message.Get(field).List()
			for itemIndex := 0; itemIndex < items.Len(); itemIndex++ {
				if messageHasTenantField(items.Get(itemIndex).Message(), visited) {
					return true
				}
			}
			continue
		}
		if message.Has(field) && messageHasTenantField(message.Get(field).Message(), visited) {
			return true
		}
	}
	return false
}

// applyTenantScope 递归补全请求中的租户字段并校验输入租户范围。
func applyTenantScope(message protoreflect.Message, authInfo *data.UserTokenPayload, visited map[protoreflect.FullName]struct{}) error {
	if !message.IsValid() {
		return nil
	}
	name := message.Descriptor().FullName()
	if _, ok := visited[name]; ok {
		return nil
	}
	visited[name] = struct{}{}
	defer delete(visited, name)
	fields := message.Descriptor().Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		if field.Name() == tenantFieldName {
			if field.Kind() != protoreflect.Int64Kind {
				continue
			}
			if err := applyTenantID(message, field, authInfo); err != nil {
				return err
			}
			continue
		}
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			continue
		}
		if field.IsList() {
			items := message.Get(field).List()
			for itemIndex := 0; itemIndex < items.Len(); itemIndex++ {
				if err := applyTenantScope(items.Get(itemIndex).Message(), authInfo, visited); err != nil {
					return err
				}
			}
			continue
		}
		if message.Has(field) {
			if err := applyTenantScope(message.Get(field).Message(), authInfo, visited); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyTenantID 处理单个租户字段的默认填充和越权输入。
func applyTenantID(message protoreflect.Message, field protoreflect.FieldDescriptor, authInfo *data.UserTokenPayload) error {
	requestedTenantID := message.Get(field).Int()
	if authInfo.TenantCode == databaseGorm.DefaultTenantCode {
		return nil
	}
	if authInfo.TenantId <= 0 {
		return errorsx.InvalidArgument("当前用户所属租户无效")
	}
	if field.HasPresence() && message.Has(field) && requestedTenantID == 0 {
		return errorsx.InvalidArgument("租户ID必须为正数")
	}
	if requestedTenantID < 0 {
		return nil
	}
	if requestedTenantID == 0 {
		message.Set(field, protoreflect.ValueOfInt64(authInfo.TenantId))
		return nil
	}
	if requestedTenantID != authInfo.TenantId {
		return errorsx.PermissionDenied("不能操作其他租户的数据")
	}
	return nil
}
