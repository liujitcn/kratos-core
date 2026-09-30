package data

import (
	"fmt"
	"slices"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/set"
	"github.com/liujitcn/kratos-core/module"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// NewClients 创建 Core 使用的多数据源 GORM 客户端集合。
//
// 模块声明但未配置的数据源回退到默认数据源建连并记录告警，保证不配置独立数据源也能启动。
func NewClients(configs map[string]*configv1.Data_Database, moduleModels module.Models) (map[string]*gorm.Client, func(), error) {
	if len(configs) == 0 {
		return map[string]*gorm.Client{}, func() {}, nil
	}
	if configs[gorm.DefaultClientName] == nil {
		return nil, func() {}, fmt.Errorf("默认数据源未配置")
	}
	nameSet := set.NewWithSize[string](len(configs) + len(moduleModels))
	for name, config := range configs {
		if name == "" {
			return nil, func() {}, fmt.Errorf("数据源名称不能为空")
		}
		if config != nil {
			nameSet.Add(name)
		}
	}
	for name := range moduleModels {
		if name == "" {
			return nil, func() {}, fmt.Errorf("数据源名称不能为空")
		}
		if name != gorm.DefaultClientName {
			if config, exists := configs[name]; !exists || config == nil {
				log.Warn(fmt.Sprintf("数据源 %q 未配置，回退默认数据源", name))
			}
		}
		nameSet.Add(name)
	}
	names := set.Sorted(nameSet)
	clients := make(map[string]*gorm.Client, len(names))
	cleanups := make(map[string]func(), len(names))
	cleanup := func() {
		for _, name := range slices.Backward(names) {
			if cleanupClient := cleanups[name]; cleanupClient != nil {
				cleanupClient()
			}
		}
	}
	var err error
	for _, name := range names {
		config := configs[name]
		// 未配置的数据源复用默认数据源建连，模型迁移与查询跟随回落到主库。
		if config == nil {
			config = configs[gorm.DefaultClientName]
		}
		var client *gorm.Client
		var cleanupClient func()
		options := []gorm.ClientOption{gorm.WithName(name)}
		options = append(options, gorm.WithMigrateModels(moduleModels[name]...))
		client, cleanupClient, err = gorm.NewGormClient(config, options...)
		if err != nil {
			if cleanupClient != nil {
				cleanupClient()
			}
			cleanup()
			return nil, func() {}, fmt.Errorf("创建数据源 %q: %w", name, err)
		}
		clients[name] = client
		cleanups[name] = cleanupClient
	}
	return clients, cleanup, nil
}
