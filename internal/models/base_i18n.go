package models

const TableNameBaseI18N = "base_i18n"

// BaseI18N 国际化翻译信息
type BaseI18N struct {
	ID         int64  `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:主键ID" json:"id"`                                                          // 主键ID
	TargetType int32  `gorm:"column:target_type;type:tinyint;not null;uniqueIndex:unique_base_i18n,priority:1;comment:目标类型：枚举【I18nTargetType】" json:"target_type"` // 目标类型：枚举【I18nTargetType】
	TargetID   int64  `gorm:"column:target_id;type:bigint;not null;uniqueIndex:unique_base_i18n,priority:2;comment:目标ID" json:"target_id"`                         // 目标ID
	Locale     string `gorm:"column:locale;type:varchar(16);not null;uniqueIndex:unique_base_i18n,priority:3;comment:语言区域" json:"locale"`                          // 语言区域
	Name       string `gorm:"column:name;type:text;comment:翻译文本" json:"name"`                                                                                      // 翻译文本
}

// TableName BaseI18N's table name
func (*BaseI18N) TableName() string {
	return TableNameBaseI18N
}
