package models

import "gorm.io/plugin/soft_delete"

// TableNameBaseUser 是用户表名。
const TableNameBaseUser = "base_user"

// BaseUser 保存登录日志回查所需的用户字段。
type BaseUser struct {
	ID        int64                 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:用户ID" json:"id"`
	TenantID  int64                 `gorm:"column:tenant_id;type:bigint;not null;uniqueIndex:unique_base_user_ user_name,priority:1;index:idx_base_user_tenant_id,priority:1;comment:租户ID" json:"tenant_id"`
	UserName  string                `gorm:"column:user_name;type:varchar(50);not null;uniqueIndex:unique_base_user_ user_name,priority:2;comment:用户账号" json:"user_name"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint unsigned;not null;uniqueIndex:unique_base_user_ user_name,priority:3;comment:删除时间;softDelete:milli" json:"deleted_at"`
}

// TableName 返回用户表名。
func (*BaseUser) TableName() string { return TableNameBaseUser }

// TableComment 返回用户表注释。
func (*BaseUser) TableComment() string { return "用户信息" }
