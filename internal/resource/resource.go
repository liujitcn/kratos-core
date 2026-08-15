package resource

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-core/internal/biz"
	"github.com/liujitcn/kratos-core/internal/data/models"
	"github.com/liujitcn/kratos-core/internal/resource/openapi"
)

// SyncAccessControl 按 API、租户角色菜单和 Casbin 规则的依赖顺序同步启动资源。
//
// API 快照先清空并重建，再同步租户角色菜单，最后重建数据库规则和内存策略。
func SyncAccessControl(
	ctx context.Context,
	documents []openapi.Document,
	baseAPICase *biz.BaseAPICase,
	baseTenantCase *biz.BaseTenantCase,
	casbinRuleCase *biz.CasbinRuleCase,
) error {
	baseAPIList := make([]*models.BaseAPI, 0)
	var err error
	for _, document := range documents {
		var items []*models.BaseAPI
		items, err = baseAPICase.OpenAPIDataToBaseAPI(document.Data)
		if err != nil {
			return fmt.Errorf("解析 OpenAPI 文档 %q: %w", document.Key, err)
		}
		baseAPIList = append(baseAPIList, items...)
	}
	err = baseAPICase.ReplaceAll(ctx, baseAPIList)
	if err != nil {
		return fmt.Errorf("同步 OpenAPI 接口: %w", err)
	}
	err = baseTenantCase.SyncTenantRoleMenus(ctx)
	if err != nil {
		return fmt.Errorf("同步租户角色菜单: %w", err)
	}
	err = casbinRuleCase.RebuildPolicy(ctx)
	if err != nil {
		return fmt.Errorf("重建 Casbin 规则: %w", err)
	}
	return nil
}
