package migration

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// FileContent 保存一个迁移文件的路径及独立内容。
type FileContent struct {
	Path    string
	Content string
}

// ReadFiles 根据数据库引用逐个读取迁移文件，保留文件顺序和边界并校验内容。
func (r *Migration) ReadFiles(moduleName, version, dataSource, references string) ([]*FileContent, error) {
	if references == "" {
		return nil, nil
	}
	var files []migration.FileReference
	err := json.Unmarshal([]byte(references), &files)
	if err != nil {
		return nil, fmt.Errorf("迁移记录不是文件引用，请重建开发库或显式转换历史记录: %w", err)
	}
	if len(files) == 0 {
		return nil, nil
	}
	for _, source := range r.sources {
		if source.Name != moduleName {
			continue
		}
		contents := make([]*FileContent, 0, len(files))
		for _, file := range files {
			root := path.Clean(source.Path)
			relative := file.Path
			if root != "." {
				if !strings.HasPrefix(relative, root+"/") {
					return nil, fmt.Errorf("迁移文件超出资源目录: %s", file.Path)
				}
				relative = strings.TrimPrefix(relative, root+"/")
			}
			parts := strings.Split(relative, "/")
			if !fs.ValidPath(file.Path) || (len(parts) != 3 && len(parts) != 4) || parts[0] != version {
				return nil, fmt.Errorf("迁移文件路径无效: %s", file.Path)
			}
			target := migration.DefaultTarget
			if len(parts) == 4 {
				target = parts[2]
			}
			if target != dataSource {
				return nil, fmt.Errorf("迁移文件数据源不匹配: %s", file.Path)
			}
			var content []byte
			content, err = fs.ReadFile(source.FS, file.Path)
			if err != nil {
				return nil, fmt.Errorf("读取迁移文件 %s: %w", file.Path, err)
			}
			// SQL 校验值对应迁移执行器实际使用的去注释内容。
			if path.Ext(file.Path) == ".sql" {
				content = stripSQLComments(content)
			}
			if migration.NewFileReference(file.Path, content).SHA256 != file.SHA256 {
				return nil, fmt.Errorf("迁移文件内容与记录不一致: %s", file.Path)
			}
			contents = append(contents, &FileContent{Path: file.Path, Content: string(content)})
		}
		return contents, nil
	}
	return nil, fmt.Errorf("迁移模块未注册: %s", moduleName)
}
