package migration

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// TestMigrationTranslationUsesTargetKey 验证迁移说明翻译按 target_key 区分。
func TestMigrationTranslationUsesTargetKey(t *testing.T) {
	modelSchema, err := schema.Parse(&migrationTranslation{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	targetKey := modelSchema.LookUpField("TargetKey")
	if targetKey == nil || targetKey.DBName != "target_key" {
		t.Fatalf("TargetKey 字段映射错误: %+v", targetKey)
	}
	if modelSchema.LookUpField("TargetType") != nil {
		t.Fatal("迁移说明翻译不应依赖 target_type 字段")
	}
	if DescriptionTargetKey != "base_migration.description" {
		t.Fatalf("迁移说明目标键错误: %q", DescriptionTargetKey)
	}
}
