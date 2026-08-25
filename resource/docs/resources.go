package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"

	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/resource/locale"
)

var docsResourceFilePattern = regexp.MustCompile(`^docs(?:\.([A-Za-z]{2,8}(?:[-_][A-Za-z0-9]{1,8})*))?\.json$`)

// resourceCatalog 保存资源文件名中解析出的语言和 JSON 内容。
type resourceCatalog struct {
	locale string
	data   []byte
}

// parsedResourceCatalog 保存资源文件及其已校验目录快照。
type parsedResourceCatalog struct {
	locale   string
	snapshot catalogSnapshot
}

// NewRegistry 根据模块默认语言和语言后缀项目文档资源创建注册表。
func NewRegistry(resources module.Docs) (*Registry, error) {
	registry := &Registry{}
	for _, resource := range resources {
		projectKey, projectName := resourceProject(resource)
		catalogs, err := readResourceCatalogs(resource.FS)
		if err != nil {
			return nil, fmt.Errorf("读取项目文档资源 %q: %w", projectKey, err)
		}
		parsed := make([]parsedResourceCatalog, 0, len(catalogs))
		var defaults *catalogSnapshot
		locales := make(map[string]struct{})
		for _, current := range catalogs {
			if _, exists := locales[current.locale]; exists {
				return nil, fmt.Errorf("项目文档资源 %q 的语言 %q 重复", projectKey, current.locale)
			}
			locales[current.locale] = struct{}{}
			var snapshot catalogSnapshot
			snapshot, err = parseCatalog(current.data, projectKey, projectName)
			if err != nil {
				return nil, fmt.Errorf("解析项目文档资源 %q 的语言 %q: %w", projectKey, current.locale, err)
			}
			parsed = append(parsed, parsedResourceCatalog{locale: current.locale, snapshot: snapshot})
			if current.locale == "" {
				defaults = &parsed[len(parsed)-1].snapshot
			}
		}
		if len(parsed) > 0 && defaults == nil {
			return nil, fmt.Errorf("项目文档资源 %q 缺少 docs.json", projectKey)
		}
		for _, current := range parsed {
			if current.locale != "" {
				err = validateLocalizedCatalog(*defaults, current.snapshot, current.locale)
				if err != nil {
					return nil, fmt.Errorf("校验项目文档资源 %q: %w", projectKey, err)
				}
			}
			err = registry.registerCatalog(current.locale, current.snapshot)
			if err != nil {
				return nil, err
			}
		}
	}
	return registry, nil
}

// resourceProject 返回项目文档资源的稳定项目标识和展示名称。
func resourceProject(resources module.ResourcesItem) (string, string) {
	projectKey := resources.ProjectKey
	if projectKey == "" {
		projectKey = "kratos-core"
	}
	projectName := resources.ProjectName
	if projectName == "" {
		projectName = projectKey
	}
	return projectKey, projectName
}

// readResourceCatalogs 读取默认和带语言后缀的项目文档 JSON 文件。
func readResourceCatalogs(files fs.FS) ([]resourceCatalog, error) {
	if files == nil {
		return nil, nil
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return readDefaultResourceCatalog(files)
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].Name() < entries[right].Name() })
	result := make([]resourceCatalog, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := docsResourceFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) == 0 {
			continue
		}
		var data []byte
		data, err = fs.ReadFile(files, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("读取文件 %q: %w", entry.Name(), err)
		}
		result = append(result, resourceCatalog{locale: locale.Normalize(matches[1]), data: data})
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].locale == "" {
			return true
		}
		if result[right].locale == "" {
			return false
		}
		return result[left].locale < result[right].locale
	})
	return result, nil
}

// readDefaultResourceCatalog 在文件系统不支持目录枚举时兼容读取默认项目文档。
func readDefaultResourceCatalog(files fs.FS) ([]resourceCatalog, error) {
	data, err := fs.ReadFile(files, "docs.json")
	if err == nil {
		return []resourceCatalog{{data: data}}, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return nil, err
}
