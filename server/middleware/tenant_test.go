package middleware

import (
	"testing"

	"github.com/liujitcn/kratos-kit/auth/data"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TestTenantScopeMiddlewareFillsNestedTenantIDs 验证普通租户可递归补全嵌套表单中的租户字段。
func TestTenantScopeMiddlewareFillsNestedTenantIDs(t *testing.T) {
	request := newTenantScopeTestRequest(t)
	authInfo := &data.UserTokenPayload{TenantId: 7, TenantCode: "tenant-7"}

	if err := applyTenantScope(request.ProtoReflect(), authInfo, make(map[protoreflect.FullName]struct{})); err != nil {
		t.Fatalf("applyTenantScope() error = %v", err)
	}
	if got := request.Get(request.Descriptor().Fields().ByName("tenant_id")).Int(); got != 7 {
		t.Fatalf("顶层租户ID = %d, want 7", got)
	}
	nested := request.Get(request.Descriptor().Fields().ByName("nested")).Message()
	if got := nested.Get(nested.Descriptor().Fields().ByName("tenant_id")).Int(); got != 7 {
		t.Fatalf("嵌套租户ID = %d, want 7", got)
	}
}

// TestTenantScopeMiddlewareRejectsCrossTenantInput 验证普通租户不能伪造其他租户。
func TestTenantScopeMiddlewareRejectsCrossTenantInput(t *testing.T) {
	request := newTenantScopeTestRequest(t)
	field := request.Descriptor().Fields().ByName("tenant_id")
	request.Set(field, protoreflect.ValueOfInt64(8))

	err := applyTenantScope(request.ProtoReflect(), &data.UserTokenPayload{TenantId: 7, TenantCode: "tenant-7"}, make(map[protoreflect.FullName]struct{}))
	if err == nil {
		t.Fatal("applyTenantScope() error = nil, want cross-tenant rejection")
	}
}

// TestTenantScopeMiddlewareKeepsDefaultTenantZero 验证默认租户不会自动改写零值。
func TestTenantScopeMiddlewareKeepsDefaultTenantZero(t *testing.T) {
	request := newTenantScopeTestRequest(t)
	if err := applyTenantScope(request.ProtoReflect(), &data.UserTokenPayload{TenantCode: defaultTenantCode}, make(map[protoreflect.FullName]struct{})); err != nil {
		t.Fatalf("applyTenantScope() error = %v", err)
	}
	field := request.Descriptor().Fields().ByName("tenant_id")
	if got := request.Get(field).Int(); got != 0 {
		t.Fatalf("默认租户租户ID = %d, want 0", got)
	}
}

// newTenantScopeTestRequest 创建带有顶层和嵌套 tenant_id 的动态 Proto 请求。
func newTenantScopeTestRequest(t *testing.T) *dynamicpb.Message {
	t.Helper()
	optional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	int64Type := descriptorpb.FieldDescriptorProto_TYPE_INT64
	nestedType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	descriptor, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String("tenant_scope_test.proto"),
		Package: proto.String("tenant.scope.test"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Request"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("tenant_id"), Number: proto.Int32(1), Label: &optional, Type: &int64Type},
					{Name: proto.String("nested"), Number: proto.Int32(2), Label: &optional, Type: &nestedType, TypeName: proto.String(".tenant.scope.test.Nested")},
				},
			},
			{
				Name: proto.String("Nested"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("tenant_id"), Number: proto.Int32(1), Label: &optional, Type: &int64Type},
				},
			},
		},
	}, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatalf("创建租户测试描述符失败: %v", err)
	}
	requestDescriptor := descriptor.Messages().ByName("Request")
	nestedDescriptor := descriptor.Messages().ByName("Nested")
	request := dynamicpb.NewMessage(requestDescriptor)
	nested := dynamicpb.NewMessage(nestedDescriptor)
	request.Set(requestDescriptor.Fields().ByName("nested"), protoreflect.ValueOfMessage(nested))
	return request
}
