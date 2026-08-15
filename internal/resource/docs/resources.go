package docs

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/liujitcn/kratos-core/pkg/module"
)

// NewRegistry 根据模块项目文档资源创建注册表。
func NewRegistry(resources module.Docs) (*Registry, error) {
	registry := &Registry{}
	for _, resource := range resources {
		projectKey, projectName := resourceProject(resource)
		data, err := readResourceFile(resource.FS, "docs.json")
		if err != nil {
			return nil, fmt.Errorf("读取项目文档资源 %q: %w", projectKey, err)
		}
		var projectRegistry *Registry
		projectRegistry, err = NewProjectRegistry(data, projectKey, projectName)
		if err != nil {
			return nil, err
		}
		if err = registry.Register(projectRegistry.Documents()...); err != nil {
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
