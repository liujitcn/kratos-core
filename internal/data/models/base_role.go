package models

import "gorm.io/plugin/soft_delete"

// TableNameBaseRole 是角色表名。
const TableNameBaseRole = "base_role"

// BaseRole 保存权限重建所需的角色字段。
type BaseRole struct {
	ID        int64                 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:角色ID" json:"id"`
	TenantID  int64                 `gorm:"column:tenant_id;type:bigint;not null;uniqueIndex:unique_base_role,priority:1;index:idx_base_role_tenant_id,priority:1;comment:租户ID" json:"tenant_id"`
	Code      string                `gorm:"column:code;type:varchar(100);not null;uniqueIndex:unique_base_role,priority:2;comment:角色代码" json:"code"`
	Menus     string                `gorm:"column:menus;type:json;not null;comment:分配的菜单列表" json:"menus"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint unsigned;not null;uniqueIndex:unique_base_role,priority:3;comment:删除时间;softDelete:milli" json:"deleted_at"`
}

// TableName 返回角色表名。
func (*BaseRole) TableName() string { return TableNameBaseRole }

// TableComment 返回角色表注释。
func (*BaseRole) TableComment() string { return "角色信息" }
