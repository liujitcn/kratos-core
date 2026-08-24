package migration

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	kitMigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

var localizedReadmePattern = regexp.MustCompile(`^README\.([A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*)\.md$`)

// DescriptionTranslation 表示一个模块迁移版本说明的单语言译文。
type DescriptionTranslation struct {
	Module      string
	Version     string
	DataSource  string
	Locale      string
	Description string
}

// loadDescriptionTranslations 读取迁移资源路径中的 README 语言说明。
func loadDescriptionTranslations(moduleName string, files fs.FS, root string) ([]DescriptionTranslation, error) {
	translations := make([]DescriptionTranslation, 0)
	cleanRoot := path.Clean(root)
	err := fs.WalkDir(files, cleanRoot, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		matches := localizedReadmePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil
		}
		relativeName := name
		if cleanRoot != "." {
			relativeName = strings.TrimPrefix(name, cleanRoot+"/")
		}
		parts := strings.Split(relativeName, "/")
		if len(parts) != 3 && len(parts) != 4 {
			return fmt.Errorf("迁移说明路径无效: %s", name)
		}
		dataSource := kitMigration.DefaultTarget
		if len(parts) == 4 {
			dataSource = parts[2]
		}
		var content []byte
		content, walkErr = fs.ReadFile(files, name)
		if walkErr != nil {
			return walkErr
		}
		translations = append(translations, DescriptionTranslation{
			Module:      moduleName,
			Version:     parts[0],
			DataSource:  dataSource,
			Locale:      matches[1],
			Description: string(content),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortDescriptionTranslations(translations)
	return translations, nil
}

// sortDescriptionTranslations 按资源定位字段稳定排序迁移说明译文。
func sortDescriptionTranslations(translations []DescriptionTranslation) {
	sort.Slice(translations, func(index, other int) bool {
		left := translations[index]
		right := translations[other]
		if left.Module != right.Module {
			return left.Module < right.Module
		}
		if left.Version != right.Version {
			return left.Version < right.Version
		}
		if left.DataSource != right.DataSource {
			return left.DataSource < right.DataSource
		}
		return left.Locale < right.Locale
	})
}

type migrationSourceFS struct {
	fs.FS
}

// Open 打开迁移执行器可见的脚本或主语言 README。
func (f migrationSourceFS) Open(name string) (fs.File, error) {
	if isHiddenDescription(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return f.FS.Open(name)
}

// ReadDir 读取目录并隐藏由宿主同步的本地化 README 和其他 Markdown。
func (f migrationSourceFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.FS, name)
	if err != nil {
		return nil, err
	}
	visible := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !isHiddenDescription(entry.Name()) {
			visible = append(visible, entry)
		}
	}
	return visible, nil
}

// isHiddenDescription 判断说明文件是否不应交给迁移执行器读取。
func isHiddenDescription(name string) bool {
	baseName := path.Base(name)
	return path.Ext(baseName) == ".md" && baseName != "README.md"
}
