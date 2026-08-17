package _const

const (
	// BASE_ROLE_CODE_SUPER 表示超级管理员角色编码。
	BASE_ROLE_CODE_SUPER = "super"
	// BASE_ROLE_CODE_TENANT 表示租户管理员角色编码。
	BASE_ROLE_CODE_TENANT = "tenant"
	// BASE_ROLE_CODE_ADMIN 表示平台管理员角色编码。
	BASE_ROLE_CODE_ADMIN = "admin"
	// BASE_ROLE_CODE_AUTHUSER 表示认证用户角色编码。
	BASE_ROLE_CODE_AUTHUSER = "authuser"
	// BASE_ROLE_CODE_USER 表示普通用户角色编码。
	BASE_ROLE_CODE_USER = "user"
)

// IsDefaultBaseRole 判断角色是否为系统内置角色。
func IsDefaultBaseRole(roleCode string) bool {
	return roleCode == BASE_ROLE_CODE_SUPER || roleCode == BASE_ROLE_CODE_TENANT
}

// IsBaseRoleStatusProtected 判断角色是否禁止通过角色管理启用或禁用。
func IsBaseRoleStatusProtected(roleCode string) bool {
	return roleCode == BASE_ROLE_CODE_ADMIN ||
		roleCode == BASE_ROLE_CODE_AUTHUSER ||
		roleCode == BASE_ROLE_CODE_USER
}
