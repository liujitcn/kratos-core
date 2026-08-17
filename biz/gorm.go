package biz

import (
	"fmt"

	"github.com/liujitcn/go-utils/set"
	"github.com/liujitcn/kratos-core/module"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// NewClients 创建 Core 使用的多数据源 GORM 客户端集合。
func NewClients(configs map[string]*configv1.Data_Database, moduleModels module.Models) (map[string]*databaseGorm.Client, func(), error) {
	if len(configs) == 0 {
		return map[string]*databaseGorm.Client{}, func() {}, nil
	}
	if configs[databaseGorm.DefaultClientName] == nil {
		return nil, func() {}, fmt.Errorf("默认数据源未配置")
	}
	nameSet := set.NewWithSize[string](len(configs))
	for name, config := range configs {
		if name == "" && config != nil {
			return nil, func() {}, fmt.Errorf("数据源名称不能为空")
		}
		if config != nil {
			nameSet.Add(name)
		}
	}
	err := validateModelDataSources(configs, moduleModels)
	if err != nil {
		return nil, func() {}, err
	}
	names := set.Sorted(nameSet)
	clients := make(map[string]*databaseGorm.Client, len(names))
	cleanups := make(map[string]func(), len(names))
	cleanup := func() {
		for index := len(names) - 1; index >= 0; index-- {
			if cleanupClient := cleanups[names[index]]; cleanupClient != nil {
				cleanupClient()
			}
		}
	}
	for _, name := range names {
		config := configs[name]
		if config == nil {
			config = configs[databaseGorm.DefaultClientName]
		}
		var client *databaseGorm.Client
		var cleanupClient func()
		options := []databaseGorm.ClientOption{databaseGorm.WithName(name)}
		options = append(options, databaseGorm.WithMigrateModels(moduleModels[name]...))
		client, cleanupClient, err = databaseGorm.NewGormClient(config, options...)
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

// validateModelDataSources 校验模块模型声明的数据源都已配置。
func validateModelDataSources(configs map[string]*configv1.Data_Database, models module.Models) error {
	for name, values := range models {
		if len(values) == 0 {
			continue
		}
		if name == "" {
			return fmt.Errorf("数据源名称不能为空")
		}
		if name != databaseGorm.DefaultClientName && configs[name] == nil {
			return fmt.Errorf("数据源 %q 未配置", name)
		}
	}
	return nil
}
