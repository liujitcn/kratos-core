package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"testing/fstest"

	"github.com/liujitcn/kratos-core/pkg/module"
)

// NewCatalog 根据 Core 和模块国际化资源创建目录。
func NewCatalog(resources module.I18n) (*I18n, error) {
	items := module.ResourcesItem{
		ProjectKey:  "core",
		ProjectName: "Kratos Core",
		FS:          Assets(),
	}
	resources = append(resources, items)
	files := fstest.MapFS{}
	var err error
	for _, resource := range resources {
		var entries []fs.DirEntry
		entries, err = fs.ReadDir(resource.FS, ".")
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			var data []byte
			data, err = fs.ReadFile(resource.FS, entry.Name())
			if err != nil {
				return nil, err
			}
			if len(bytes.TrimSpace(data)) == 0 {
				continue
			}
			var messages map[string]json.RawMessage
			if err = json.Unmarshal(data, &messages); err != nil {
				return nil, fmt.Errorf("解析国际化资源 %s: %w", entry.Name(), err)
			}
			path := entry.Name()
			if existing := files[path]; existing != nil {
				var current map[string]json.RawMessage
				if err = json.Unmarshal(existing.Data, &current); err != nil {
					return nil, err
				}
				for key, value := range messages {
					if previous, exists := current[key]; exists && string(previous) != string(value) {
						return nil, fmt.Errorf("国际化消息键重复: %s/%s", path, key)
					}
					current[key] = value
				}
				data, err = json.Marshal(current)
				if err != nil {
					return nil, err
				}
			}
			files[path] = &fstest.MapFile{Data: data}
		}
	}
	var i18n *I18n
	i18n, err = NewI18n("modules", files)
	if err != nil {
		return nil, err
	}
	return i18n, nil
}
