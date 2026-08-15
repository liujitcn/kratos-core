package models

import "gorm.io/plugin/soft_delete"

// TableNameBaseMenu 是菜单表名。
const TableNameBaseMenu = "base_menu"

// BaseMenu 保存权限重建所需的菜单关联字段。
type BaseMenu struct {
	ID        int64                 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:菜单ID" json:"id"`
	API       string                `gorm:"column:api;type:json;not null;comment:分配的API列表" json:"api"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint unsigned;not null;comment:删除时间;softDelete:milli" json:"deleted_at"`
}

// TableName 返回菜单表名。
func (*BaseMenu) TableName() string { return TableNameBaseMenu }

// TableComment 返回菜单表注释。
func (*BaseMenu) TableComment() string { return "菜单信息" }
