package migration

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	kitMigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

var localizedReadmePattern = regexp.MustCompile(`^README\.([A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*)\.md$`)

// DescriptionTargetType 是迁移说明在 base_i18n 中使用的目标类型。
const DescriptionTargetType int32 = 7

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
	file, err := f.FS.Open(name)
	if err != nil || path.Ext(name) != ".sql" {
		return file, err
	}
	var info fs.FileInfo
	info, err = file.Stat()
	if err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("读取迁移脚本信息失败: %w; 关闭迁移脚本失败: %v", err, closeErr)
		}
		return nil, err
	}
	var content []byte
	content, err = io.ReadAll(file)
	closeErr := file.Close()
	if err != nil {
		return nil, fmt.Errorf("读取迁移脚本失败: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("关闭迁移脚本失败: %w", closeErr)
	}
	return &migrationSQLFile{Reader: bytes.NewReader(stripSQLComments(content)), info: info}, nil
}

type migrationSQLFile struct {
	*bytes.Reader
	info fs.FileInfo
}

// Stat 返回清理注释后的迁移脚本文件信息。
func (f *migrationSQLFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

// Close 关闭清理注释后的迁移脚本文件。
func (f *migrationSQLFile) Close() error {
	return nil
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

type sqlCommentState uint8

const (
	sqlNormal sqlCommentState = iota
	sqlSingleQuote
	sqlDoubleQuote
	sqlBacktick
	sqlLineComment
	sqlBlockComment
)

// stripSQLComments 删除 SQL 行注释和块注释，同时保留字符串及标识符中的注释样式。
func stripSQLComments(content []byte) []byte {
	result := make([]byte, 0, len(content))
	state := sqlNormal
	for index := 0; index < len(content); {
		character := content[index]
		switch state {
		case sqlNormal:
			switch character {
			case '\'':
				result = append(result, character)
				state = sqlSingleQuote
				index++
			case '"':
				result = append(result, character)
				state = sqlDoubleQuote
				index++
			case '`':
				result = append(result, character)
				state = sqlBacktick
				index++
			case '#':
				state = sqlLineComment
				index++
			case '-':
				if index+1 < len(content) && content[index+1] == '-' && (index+2 == len(content) || isSQLWhitespace(content[index+2])) {
					state = sqlLineComment
					index += 2
					continue
				}
				result = append(result, character)
				index++
			case '/':
				if index+1 < len(content) && content[index+1] == '*' {
					state = sqlBlockComment
					index += 2
					continue
				}
				result = append(result, character)
				index++
			default:
				result = append(result, character)
				index++
			}
		case sqlSingleQuote, sqlDoubleQuote, sqlBacktick:
			result = append(result, character)
			index++
			if character == '\\' && index < len(content) {
				result = append(result, content[index])
				index++
				continue
			}
			if (state == sqlSingleQuote && character == '\'') ||
				(state == sqlDoubleQuote && character == '"') ||
				(state == sqlBacktick && character == '`') {
				if index < len(content) && content[index] == character {
					result = append(result, content[index])
					index++
					continue
				}
				state = sqlNormal
			}
		case sqlLineComment:
			if character == '\n' || character == '\r' {
				result = append(result, character)
				state = sqlNormal
			}
			index++
		case sqlBlockComment:
			if character == '*' && index+1 < len(content) && content[index+1] == '/' {
				state = sqlNormal
				index += 2
				continue
			}
			index++
		}
	}
	return result
}

// isSQLWhitespace 判断字符是否为 SQL 行注释要求的空白字符。
func isSQLWhitespace(character byte) bool {
	return character == ' ' || character == '\t' || character == '\n' || character == '\r' || character == '\f'
}
