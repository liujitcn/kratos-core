package migration

import (
	"context"
	"fmt"
	"sort"

	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// Migration 保存宿主数据库客户端、迁移注册表及执行顺序。
type Migration struct {
	registry  *migration.Registry
	names     []migration.ModuleName
	databases map[string]*gorm.Client
}

// NewMigration 创建迁移注册表，注册资源后立即执行数据库迁移。
func NewMigration(ctx *bootstrap.Context, databases map[string]*gorm.Client, migrations module.Migrations) (*Migration, error) {
	contributors := make(migration.AdditionalMigrations, 0, len(migrations))
	names := make([]migration.ModuleName, 0, len(migrations))
	descriptionTranslations := make([]DescriptionTranslation, 0)
	var err error
	for _, value := range migrations {
		dependencies := make([]migration.ModuleName, 0, len(value.Dependencies))
		for _, dependency := range value.Dependencies {
			dependencies = append(dependencies, migration.ModuleName(dependency))
		}
		name := migration.ModuleName(value.Name)
		var translations []DescriptionTranslation
		translations, err = loadDescriptionTranslations(value.Name, value.FS, value.Path)
		if err != nil {
			return nil, fmt.Errorf("读取数据库迁移说明 %q: %w", value.Name, err)
		}
		descriptionTranslations = append(descriptionTranslations, translations...)
		contributors = append(contributors, migrationContributor{
			name:       name,
			migrations: []migration.Migration{{FS: migrationSourceFS{FS: value.FS}, Path: value.Path, Dependencies: dependencies}},
		})
		names = append(names, name)
	}
	sortDescriptionTranslations(descriptionTranslations)
	registry := &Migration{names: names, databases: databases}
	if len(contributors) == 0 {
		return registry, nil
	}
	registry.registry, err = migration.NewRegistry(contributors)
	if err != nil {
		return nil, fmt.Errorf("注册数据库迁移: %w", err)
	}
	err = registry.Run(ctx.Context())
	if err != nil {
		return nil, err
	}
	err = syncDescriptionTranslations(ctx.Context(), databases, descriptionTranslations)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

// Run 使用当前注册表中的多数据源客户端并按依赖顺序执行宿主迁移。
func (r *Migration) Run(ctx context.Context) error {
	if r == nil || r.registry == nil {
		return nil
	}
	clients := r.databases
	var runner *migration.Runner
	var err error
	runner, err = migration.NewRunner(r.registry)
	if err != nil {
		return fmt.Errorf("创建迁移执行器: %w", err)
	}
	clientNames := make([]string, 0, len(clients))
	for name := range clients {
		clientNames = append(clientNames, name)
	}
	sort.Strings(clientNames)
	for _, name := range clientNames {
		err = runner.SetClient(clients[name])
		if err != nil {
			return fmt.Errorf("注入迁移数据源 %q: %w", name, err)
		}
	}
	for _, name := range r.names {
		err = runner.Run(ctx, name)
		if err != nil {
			return fmt.Errorf("执行模块数据库迁移 %q: %w", name, err)
		}
	}
	return nil
}

type migrationContributor struct {
	name       migration.ModuleName
	migrations []migration.Migration
}

// Name 返回迁移模块名称。
func (c migrationContributor) Name() migration.ModuleName {
	return c.name
}

// Migrations 返回迁移资源快照。
func (c migrationContributor) Migrations() []migration.Migration {
	return append([]migration.Migration(nil), c.migrations...)
}
