package models

import "time"

// TableNameBaseMigration 是迁移记录表名。
const TableNameBaseMigration = "base_migration"

// BaseMigration 保存版本化迁移的执行记录。
type BaseMigration struct {
	// ID 是记录主键。
	ID int64 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:主键ID" json:"id"`
	// Module 是迁移模块名称。
	Module string `gorm:"column:module;type:varchar(64);not null;uniqueIndex:unique_base_migration,priority:1;comment:迁移模块" json:"module"`
	// DataSource 是迁移目标数据源。
	DataSource string `gorm:"column:data_source;type:varchar(20);not null;uniqueIndex:unique_base_migration,priority:2;comment:数据源" json:"data_source"`
	// Version 是迁移版本目录名称。
	Version string `gorm:"column:version;type:varchar(50);not null;uniqueIndex:unique_base_migration,priority:3;comment:迁移版本" json:"version"`
	// UpSql 是升级脚本内容。
	UpSql string `gorm:"column:up_sql;type:longtext;comment:升级脚本" json:"up_sql"`
	// DownSql 是回退脚本内容。
	DownSql string `gorm:"column:down_sql;type:longtext;comment:回退脚本" json:"down_sql"`
	// Description 是迁移说明。
	Description string `gorm:"column:description;type:longtext;comment:升级描述" json:"description"`
	// CreatedAt 是记录创建时间。
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;comment:创建时间" json:"created_at"`
}

// TableName 返回迁移记录表名。
func (*BaseMigration) TableName() string { return TableNameBaseMigration }
