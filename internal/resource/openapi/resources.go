package openapi

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/liujitcn/kratos-core/pkg/module"
)

// NewRegistry 根据模块 OpenAPI 资源创建注册表。
func NewRegistry(resources module.OpenAPI) (*Registry, error) {
	registry := &Registry{}
	for _, resource := range resources {
		projectKey, projectName := resourceProject(resource)
		data, err := readResourceFile(resource.FS, "openapi.yaml", "openapi.yml", "openapi.json")
		if err != nil {
			return nil, fmt.Errorf("读取 OpenAPI 资源 %q: %w", projectKey, err)
		}
		if len(data) == 0 {
			continue
		}
		if err = registry.Register(Document{Key: projectKey, Name: projectName, Data: data}); err != nil {
			return nil, err
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

func readResourceFile(files fs.FS, names ...string) ([]byte, error) {
	if files == nil {
		return nil, nil
	}
	for _, name := range names {
		data, err := fs.ReadFile(files, name)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, nil
}
