package models

// TableNameBaseAPII18n 是 API 国际化表名。
const TableNameBaseAPII18n = "base_api_i18n"

// BaseAPII18n 保存 OpenAPI 接口展示文本的语言版本。
type BaseAPII18n struct {
	ID          int64  `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:API国际化ID" json:"id"`
	Operation   string `gorm:"column:operation;type:varchar(255);not null;uniqueIndex:unique_base_api_i18n,priority:1;comment:接口操作" json:"operation"`
	Locale      string `gorm:"column:locale;type:varchar(32);not null;uniqueIndex:unique_base_api_i18n,priority:2;comment:语言标识" json:"locale"`
	ToolPrompts string `gorm:"column:tool_prompts;type:json;not null;comment:工具提示词" json:"tool_prompts"`
	ServiceDesc string `gorm:"column:service_desc;type:varchar(255);not null;comment:服务描述" json:"service_desc"`
	Desc        string `gorm:"column:desc;type:varchar(500);not null;comment:接口描述" json:"desc"`
}

// TableName 返回 API 国际化表名。
func (*BaseAPII18n) TableName() string { return TableNameBaseAPII18n }

// TableComment 返回 API 国际化表注释。
func (*BaseAPII18n) TableComment() string { return "API国际化信息" }
