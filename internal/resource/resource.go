package resource

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-core/internal/biz"
	"github.com/liujitcn/kratos-core/internal/data/models"
	"github.com/liujitcn/kratos-core/internal/resource/migration"
	"github.com/liujitcn/kratos-core/internal/resource/openapi"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// PermissionSynchronizer 负责按依赖顺序同步 API、租户角色菜单和 Casbin 规则。
type PermissionSynchronizer struct {
	baseAPICase    *biz.BaseAPICase
	baseTenantCase *biz.BaseTenantCase
	casbinRuleCase *biz.CasbinRuleCase
}

// NewPermissionSynchronizer 创建启动期权限资源同步器，并完成首次同步。
func NewPermissionSynchronizer(
	ctx *bootstrap.Context,
	migrations *migration.Migration,
	registry *openapi.Registry,
	baseAPICase *biz.BaseAPICase,
	baseTenantCase *biz.BaseTenantCase,
	casbinRuleCase *biz.CasbinRuleCase,
) (*PermissionSynchronizer, error) {
	if migrations == nil {
		return nil, fmt.Errorf("数据库迁移资源未初始化")
	}
	if baseAPICase == nil || baseTenantCase == nil || casbinRuleCase == nil {
		return nil, fmt.Errorf("权限同步业务资源未初始化")
	}
	s := &PermissionSynchronizer{
		baseAPICase:    baseAPICase,
		baseTenantCase: baseTenantCase,
		casbinRuleCase: casbinRuleCase,
	}
	var documents []openapi.Document
	if registry != nil {
		documents = registry.Documents()
	}
	err := s.Sync(ctx.Context(), documents)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Sync 按 API、租户角色菜单和 Casbin 规则的依赖顺序同步权限资源。
//
// API 快照先清空并重建，再同步租户角色菜单，最后重建数据库规则和内存策略。
func (s *PermissionSynchronizer) Sync(ctx context.Context, documents []openapi.Document) error {
	baseAPIList := make([]*models.BaseAPI, 0)
	var err error
	for _, document := range documents {
		var items []*models.BaseAPI
		items, err = s.baseAPICase.OpenAPIDataToBaseAPI(document.Data)
		if err != nil {
			return fmt.Errorf("解析 OpenAPI 文档 %q: %w", document.Key, err)
		}
		baseAPIList = append(baseAPIList, items...)
	}
	err = s.baseAPICase.ReplaceAll(ctx, baseAPIList)
	if err != nil {
		return fmt.Errorf("同步 OpenAPI 接口: %w", err)
	}
	err = s.baseTenantCase.SyncTenantRoleMenus(ctx)
	if err != nil {
		return fmt.Errorf("同步租户角色菜单: %w", err)
	}
	err = s.casbinRuleCase.RebuildPolicy(ctx)
	if err != nil {
		return fmt.Errorf("重建 Casbin 规则: %w", err)
	}
	return nil
}
