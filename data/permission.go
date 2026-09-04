package data

import "context"

// MenuRecord 表示权限重建所需的菜单字段。
type MenuRecord struct {
	// ID 是菜单编号。
	ID int64
	// API 是菜单关联的接口列表 JSON。
	API string
}

// RoleRecord 表示权限重建和租户菜单同步所需的角色字段。
type RoleRecord struct {
	// ID 是角色编号。
	ID int64
	// TenantID 是租户编号。
	TenantID int64
	// Code 是角色编码。
	Code string
	// Menus 是角色关联的菜单列表 JSON。
	Menus string
}

// TenantRecord 表示权限重建所需的租户字段。
type TenantRecord struct {
	// ID 是租户编号。
	ID int64
	// Code 是租户编码。
	Code string
}

// PolicyRecord 表示 Casbin 权限规则所需的字段。
type PolicyRecord struct {
	// Ptype 是策略类型。
	Ptype string
	// V0 是租户编码。
	V0 string
	// V1 是角色编码。
	V1 string
	// V2 是资源路径。
	V2 string
	// V3 是请求方法。
	V3 string
	// V4 是项目范围。
	V4 string
}

// PermissionStore 提供启动期权限资源同步所需的宿主持久化能力。
type PermissionStore interface {
	// FindTenantByCode 按租户编码查询租户。
	FindTenantByCode(context.Context, string) (TenantRecord, error)
	// ListTenants 查询全部租户的权限字段。
	ListTenants(context.Context) ([]TenantRecord, error)
	// FindRoleByTenantIDAndCode 按租户和角色编码查询角色。
	FindRoleByTenantIDAndCode(context.Context, int64, string) (RoleRecord, error)
	// ListRoles 查询全部角色的权限字段。
	ListRoles(context.Context) ([]RoleRecord, error)
	// ListRolesByCode 查询指定编码的全部角色。
	ListRolesByCode(context.Context, string) ([]RoleRecord, error)
	// UpdateRoleMenus 更新角色关联的菜单列表。
	UpdateRoleMenus(context.Context, RoleRecord) error
	// ListMenusByIDs 按编号批量查询菜单关联接口。
	ListMenusByIDs(context.Context, []int64) ([]MenuRecord, error)
	// ListPolicies 查询全部 Casbin 规则。
	ListPolicies(context.Context) ([]PolicyRecord, error)
	// ReplacePolicies 替换全部 Casbin 规则。
	ReplacePolicies(context.Context, []PolicyRecord) error
}
