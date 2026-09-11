package migration

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// TestReadFiles 验证迁移详情按文件读取、内容校验及资源隔离。
func TestReadFiles(t *testing.T) {
	files := fstest.MapFS{
		"assets/v0.0.1/mysql/a.up.sql": {Data: []byte("-- 注释\nSELECT 1;")},
		"assets/v0.0.1/mysql/b.up.sql": {Data: []byte("SELECT 2;")},
	}
	references, err := migration.EncodeFileReferences(migrationSourceFS{FS: files}, []string{
		"assets/v0.0.1/mysql/a.up.sql", "assets/v0.0.1/mysql/b.up.sql",
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &Migration{sources: module.Migrations{{Name: "admin", FS: files, Path: "assets"}}}
	var content []*FileContent
	content, err = runtime.ReadFiles("admin", "v0.0.1", "default", references)
	if err != nil || len(content) != 2 || content[0].Path != "assets/v0.0.1/mysql/a.up.sql" || content[0].Content != "SELECT 1;" || content[1].Path != "assets/v0.0.1/mysql/b.up.sql" || content[1].Content != "SELECT 2;" {
		t.Fatalf("内容=%+v, 错误=%v", content, err)
	}
	for _, test := range []struct{ name, module, version, source, refs string }{
		{"其他模块", "other", "v0.0.1", "default", references},
		{"其他版本", "admin", "v0.0.2", "default", references},
		{"其他数据源", "admin", "v0.0.1", "tenant", references},
		{"目录逃逸", "admin", "v0.0.1", "default", strings.ReplaceAll(references, "assets/v0.0.1", "assets/../v0.0.1")},
		{"旧正文", "admin", "v0.0.1", "default", "SELECT 1;"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, readErr := runtime.ReadFiles(test.module, test.version, test.source, test.refs)
			if readErr == nil {
				t.Fatal("应拒绝无效文件引用")
			}
		})
	}
	files["assets/v0.0.1/mysql/a.up.sql"].Data = []byte("SELECT 3;")
	_, err = runtime.ReadFiles("admin", "v0.0.1", "default", references)
	if err == nil || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("应拒绝被修改的文件: %v", err)
	}
	delete(files, "assets/v0.0.1/mysql/a.up.sql")
	_, err = runtime.ReadFiles("admin", "v0.0.1", "default", references)
	if err == nil {
		t.Fatal("缺少历史文件应返回错误")
	}
}

// TestDescriptionReferences 验证多语言说明只保留引用并从文件读取正文。
func TestDescriptionReferences(t *testing.T) {
	files := fstest.MapFS{"v0.0.1/mysql/README.en-US.md": {Data: []byte("English description")}}
	translations, err := loadDescriptionTranslations("admin", files, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(translations) != 1 || strings.Contains(translations[0].Description, "English description") {
		t.Fatalf("说明未转换为文件引用: %+v", translations)
	}
	runtime := &Migration{sources: module.Migrations{{Name: "admin", FS: files, Path: "."}}}
	var content []*FileContent
	content, err = runtime.ReadFiles("admin", "v0.0.1", "default", translations[0].Description)
	if err != nil || len(content) != 1 || content[0].Path != "v0.0.1/mysql/README.en-US.md" || content[0].Content != "English description" {
		t.Fatalf("内容=%+v, 错误=%v", content, err)
	}
}
