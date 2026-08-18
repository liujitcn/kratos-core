package errorsx

import "github.com/go-kratos/kratos/v3/errors"

const (
	// METADATA_KEY_CONFLICT_TYPE 标识冲突类型。
	METADATA_KEY_CONFLICT_TYPE = "conflict_type"
	// METADATA_KEY_RESOURCE 标识资源名称。
	METADATA_KEY_RESOURCE = "resource"
	// METADATA_KEY_FIELD 标识字段名称。
	METADATA_KEY_FIELD = "field"
	// METADATA_KEY_CONSTRAINT 标识数据库约束名称。
	METADATA_KEY_CONSTRAINT = "constraint"
	// METADATA_KEY_CHILD_RESOURCE 标识子资源名称。
	METADATA_KEY_CHILD_RESOURCE = "child_resource"
	// METADATA_KEY_CURRENT_STATE 标识当前状态。
	METADATA_KEY_CURRENT_STATE = "current_state"
	// METADATA_KEY_EXPECTED_STATE 标识期望状态。
	METADATA_KEY_EXPECTED_STATE = "expected_state"

	// CONFLICT_TYPE_UNIQUE_VIOLATION 表示唯一约束冲突。
	CONFLICT_TYPE_UNIQUE_VIOLATION = "unique_violation"
	// CONFLICT_TYPE_HAS_CHILDREN 表示仍存在子资源。
	CONFLICT_TYPE_HAS_CHILDREN = "has_children"
	// CONFLICT_TYPE_STATE_CONFLICT 表示状态冲突。
	CONFLICT_TYPE_STATE_CONFLICT = "state_conflict"
	// CONFLICT_TYPE_PROTECTED_RESOURCE 表示受保护资源。
	CONFLICT_TYPE_PROTECTED_RESOURCE = "protected_resource"
)

// UniqueConflict 构造唯一约束冲突错误。
func UniqueConflict(message, resource, field, constraint string) *errors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_UNIQUE_VIOLATION,
		METADATA_KEY_RESOURCE:      resource,
		METADATA_KEY_FIELD:         field,
	}
	// 提供了约束名时，再补充到错误元数据中。
	if constraint != "" {
		metadata[METADATA_KEY_CONSTRAINT] = constraint
	}
	return Conflict(message).WithMetadata(metadata)
}

// HasChildrenConflict 构造存在子资源的冲突错误。
func HasChildrenConflict(message, resource, childResource string) *errors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_HAS_CHILDREN,
		METADATA_KEY_RESOURCE:      resource,
	}
	// 已知子资源名称时，再补充到错误元数据中。
	if childResource != "" {
		metadata[METADATA_KEY_CHILD_RESOURCE] = childResource
	}
	return Conflict(message).WithMetadata(metadata)
}

// StateConflict 构造状态冲突错误。
func StateConflict(message, resource, currentState, expectedState string) *errors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_STATE_CONFLICT,
		METADATA_KEY_RESOURCE:      resource,
	}
	// 提供了当前状态时，再补充到错误元数据中。
	if currentState != "" {
		metadata[METADATA_KEY_CURRENT_STATE] = currentState
	}
	// 提供了期望状态时，再补充到错误元数据中。
	if expectedState != "" {
		metadata[METADATA_KEY_EXPECTED_STATE] = expectedState
	}
	return Conflict(message).WithMetadata(metadata)
}

// ProtectedResourceConflict 构造受保护资源冲突错误。
func ProtectedResourceConflict(message, resource string) *errors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_PROTECTED_RESOURCE,
	}
	// 提供了资源名称时，再补充到错误元数据中。
	if resource != "" {
		metadata[METADATA_KEY_RESOURCE] = resource
	}
	return Conflict(message).WithMetadata(metadata)
}
