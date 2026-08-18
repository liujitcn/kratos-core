package models

// TableNameCasbinRule 是 Casbin 规则表名。
const TableNameCasbinRule = "casbin_rule"

// CasbinRule 保存 Casbin 策略规则。
type CasbinRule struct {
	ID    int64  `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement:true;comment:策略规则ID" json:"id"`
	Ptype string `gorm:"column:ptype;type:varchar(100);not null;uniqueIndex:idx_casbin_rule,priority:1;comment:策略类型" json:"ptype"`
	V0    string `gorm:"column:v0;type:varchar(100);not null;uniqueIndex:idx_casbin_rule,priority:2;comment:租户编码" json:"v0"`
	V1    string `gorm:"column:v1;type:varchar(100);not null;uniqueIndex:idx_casbin_rule,priority:3;comment:角色编码" json:"v1"`
	V2    string `gorm:"column:v2;type:varchar(100);not null;uniqueIndex:idx_casbin_rule,priority:4;comment:资源路径" json:"v2"`
	V3    string `gorm:"column:v3;type:varchar(100);not null;uniqueIndex:idx_casbin_rule,priority:5;comment:请求方法" json:"v3"`
	V4    string `gorm:"column:v4;type:varchar(100);not null;uniqueIndex:idx_casbin_rule,priority:6;comment:项目范围" json:"v4"`
}

// TableName 返回 Casbin 规则表名。
func (*CasbinRule) TableName() string { return TableNameCasbinRule }

// TableComment 返回 Casbin 规则表注释。
func (*CasbinRule) TableComment() string { return "Casbin权限信息" }
