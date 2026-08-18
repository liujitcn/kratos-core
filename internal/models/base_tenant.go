package models

import "gorm.io/plugin/soft_delete"

// TableNameBaseTenant 是租户表名。
const TableNameBaseTenant = "base_tenant"

// BaseTenant 保存租户权限重建所需的字段。
type BaseTenant struct {
	ID        int64                 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:租户ID" json:"id"`
	Code      string                `gorm:"column:code;type:varchar(50);not null;uniqueIndex:unique_base_tenant,priority:1;comment:租户编码" json:"code"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint unsigned;not null;uniqueIndex:unique_base_tenant,priority:2;comment:删除时间;softDelete:milli" json:"deleted_at"`
}

// TableName 返回租户表名。
func (*BaseTenant) TableName() string { return TableNameBaseTenant }

// TableComment 返回租户表注释。
func (*BaseTenant) TableComment() string { return "租户信息表" }
