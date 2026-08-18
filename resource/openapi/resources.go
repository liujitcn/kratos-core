package openapi

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"

	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/resource/locale"
)

var resourceFilePattern = regexp.MustCompile(`^openapi(?:\.([A-Za-z]{2,8}(?:[-_][A-Za-z0-9]{1,8})*))?\.(yaml|yml|json)$`)

type resourceDocument struct {
	locale string
	data   []byte
}

// NewRegistry 根据模块 OpenAPI 资源创建注册表。
func NewRegistry(resources module.OpenAPI) (*Registry, error) {
	registry := &Registry{}
	for _, resource := range resources {
		projectKey, projectName := resourceProject(resource)
		documents, err := readResourceFiles(resource.FS)
		if err != nil {
			return nil, fmt.Errorf("读取 OpenAPI 资源 %q: %w", projectKey, err)
		}
		for _, document := range documents {
			if err = registry.Register(Document{Key: projectKey, Name: projectName, Locale: document.locale, Data: document.data}); err != nil {
				return nil, err
			}
		}
	}
	return registry, nil
}

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

// readResourceFiles 读取默认和带语言后缀的 OpenAPI 文件。
func readResourceFiles(files fs.FS) ([]resourceDocument, error) {
	if files == nil {
		return nil, nil
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return readDefaultResourceFiles(files)
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].Name() < entries[right].Name() })
	result := make([]resourceDocument, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := resourceFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) == 0 {
			continue
		}
		var data []byte
		data, err = fs.ReadFile(files, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("读取文件 %q: %w", entry.Name(), err)
		}
		result = append(result, resourceDocument{locale: locale.Normalize(matches[1]), data: data})
	}
	return result, nil
}

// readDefaultResourceFiles 在文件系统不支持目录枚举时兼容读取默认文档。
func readDefaultResourceFiles(files fs.FS) ([]resourceDocument, error) {
	for _, name := range []string{"openapi.yaml", "openapi.yml", "openapi.json"} {
		data, err := fs.ReadFile(files, name)
		if err == nil {
			return []resourceDocument{{data: data}}, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, nil
}
