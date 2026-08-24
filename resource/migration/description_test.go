package migration

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

// TestLoadDescriptionTranslations 验证默认和命名数据源的 README 译文解析。
func TestLoadDescriptionTranslations(t *testing.T) {
	files := fstest.MapFS{
		"assets/v0.0.1/mysql/README.md":                 {Data: []byte("主说明")},
		"assets/v0.0.1/mysql/README.en-US.md":           {Data: []byte("English")},
		"assets/v0.0.1/mysql/README.zh-TW.md":           {Data: []byte("繁體中文")},
		"assets/v0.0.1/mysql/analytics/README.ja-JP.md": {Data: []byte("日本語")},
		"assets/v0.0.1/mysql/default_data.up.sql":       {Data: []byte("SELECT 1;")},
	}
	want := []DescriptionTranslation{
		{Module: "admin", Version: "v0.0.1", DataSource: "analytics", Locale: "ja-JP", Description: "日本語"},
		{Module: "admin", Version: "v0.0.1", DataSource: "default", Locale: "en-US", Description: "English"},
		{Module: "admin", Version: "v0.0.1", DataSource: "default", Locale: "zh-TW", Description: "繁體中文"},
	}
	got, err := loadDescriptionTranslations("admin", files, "assets")
	if err != nil {
		t.Fatalf("读取迁移说明译文失败: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("迁移说明译文不一致\nwant: %#v\n got: %#v", want, got)
	}
}

// TestMigrationSourceFS 验证迁移执行器只读取统一主说明 README.md。
func TestMigrationSourceFS(t *testing.T) {
	files := fstest.MapFS{
		"v0.0.1/mysql/README.md":                   {Data: []byte("主说明")},
		"v0.0.1/mysql/README.en-US.md":             {Data: []byte("English")},
		"v0.0.1/mysql/default_data.description.md": {Data: []byte("旧说明")},
		"v0.0.1/mysql/default_data.up.sql":         {Data: []byte("SELECT 1;")},
	}
	visibleFS := migrationSourceFS{FS: files}
	entries, err := fs.ReadDir(visibleFS, "v0.0.1/mysql")
	if err != nil {
		t.Fatalf("读取迁移目录失败: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	wantNames := []string{"README.md", "default_data.up.sql"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("迁移执行器可见文件不一致\nwant: %#v\n got: %#v", wantNames, names)
	}
	var content []byte
	content, err = fs.ReadFile(visibleFS, "v0.0.1/mysql/README.md")
	if err != nil {
		t.Fatalf("读取主说明失败: %v", err)
	}
	if string(content) != "主说明" {
		t.Fatalf("主说明内容不一致: %q", content)
	}
	_, err = fs.ReadFile(visibleFS, "v0.0.1/mysql/README.en-US.md")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("本地化 README 应对迁移执行器隐藏: %v", err)
	}
}

// TestMigrationSourceFSCleansSQLComments 验证迁移执行器读取 SQL 时移除注释、保留原有空行并保留字符串内容。
func TestMigrationSourceFSCleansSQLComments(t *testing.T) {
	files := fstest.MapFS{
		"v0.0.1/mysql/default_data.up.sql": {Data: []byte("-- 表结构\n\nCREATE TABLE `demo` (\n  `value` varchar(32) DEFAULT '/* 保留 */',/* 字段说明 */\n  `expr` int DEFAULT (a--b),# 行尾说明\n\n  `id` bigint NOT NULL\n);\n")},
	}
	visibleFS := migrationSourceFS{FS: files}
	content, err := fs.ReadFile(visibleFS, "v0.0.1/mysql/default_data.up.sql")
	if err != nil {
		t.Fatalf("读取迁移 SQL 失败: %v", err)
	}
	want := "\nCREATE TABLE `demo` (\n  `value` varchar(32) DEFAULT '/* 保留 */',\n  `expr` int DEFAULT (a--b),\n\n  `id` bigint NOT NULL\n);\n"
	if string(content) != want {
		t.Fatalf("迁移 SQL 注释清理结果不一致\nwant: %q\n got: %q", want, content)
	}
}
