package resource

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-core/internal/biz"
	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/data/models"
	"github.com/liujitcn/kratos-core/internal/resource/migration"
	"github.com/liujitcn/kratos-core/internal/resource/openapi"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// SyncResult 记录启动期资源同步结果。
type SyncResult struct {
	// DocumentCount 是参与同步的 OpenAPI 文档数量。
	DocumentCount int
	// APICount 是同步生成的 API 数量。
	APICount int
}

type synchronizer struct {
	baseAPICase    *biz.BaseAPICase
	baseTenantCase *biz.BaseTenantCase
	casbinRuleCase *biz.CasbinRuleCase
	transaction    data.Transaction
}

// NewSyncResult 执行启动期资源同步并返回同步结果。
func NewSyncResult(
	ctx *bootstrap.Context,
	migrations *migration.Migration,
	registry *openapi.Registry,
	transaction data.Transaction,
	baseAPICase *biz.BaseAPICase,
	baseTenantCase *biz.BaseTenantCase,
	casbinRuleCase *biz.CasbinRuleCase,
) (*SyncResult, error) {
	if migrations == nil {
		return nil, fmt.Errorf("数据库迁移资源未初始化")
	}
	if transaction == nil || baseAPICase == nil || baseAPICase.BaseAPIRepository == nil || baseAPICase.BaseAPII18nRepository == nil || baseTenantCase == nil || casbinRuleCase == nil {
		return nil, fmt.Errorf("资源同步依赖未初始化")
	}
	s := &synchronizer{
		baseAPICase:    baseAPICase,
		baseTenantCase: baseTenantCase,
		casbinRuleCase: casbinRuleCase,
		transaction:    transaction,
	}
	var documents []openapi.Document
	var translations []*models.BaseAPII18n
	var err error
	if registry != nil {
		documents = registry.DocumentsByLocale("")
		locales := registry.Locales()
		for _, locale := range locales {
			for _, document := range registry.DocumentsByLocale(locale) {
				var items []*models.BaseAPII18n
				items, err = baseAPICase.OpenAPIDataToBaseAPII18n(document.Data, locale)
				if err != nil {
					return nil, fmt.Errorf("解析 OpenAPI 国际化文档 %q/%q: %w", document.Key, locale, err)
				}
				translations = append(translations, items...)
			}
		}
	}
	var apiCount int
	apiCount, err = s.sync(ctx.Context(), documents, translations)
	if err != nil {
		return nil, err
	}
	return &SyncResult{DocumentCount: len(documents), APICount: apiCount}, nil
}

// sync 按 API、租户角色菜单和 Casbin 规则的依赖顺序同步资源。
//
// API 快照先清空并重建，再同步租户角色菜单，最后重建数据库规则和内存策略。
func (s *synchronizer) sync(ctx context.Context, documents []openapi.Document, translations []*models.BaseAPII18n) (int, error) {
	baseAPIList := make([]*models.BaseAPI, 0)
	var err error
	for _, document := range documents {
		var items []*models.BaseAPI
		items, err = s.baseAPICase.OpenAPIDataToBaseAPI(document.Data)
		if err != nil {
			return 0, fmt.Errorf("解析 OpenAPI 文档 %q: %w", document.Key, err)
		}
		baseAPIList = append(baseAPIList, items...)
	}
	err = s.transaction.Transaction(ctx, func(transactionContext context.Context) error {
		err = s.baseAPICase.BaseAPIRepository.ReplaceAll(transactionContext, baseAPIList)
		if err != nil {
			return fmt.Errorf("同步 OpenAPI 接口: %w", err)
		}
		err = s.baseAPICase.BaseAPII18nRepository.ReplaceAll(transactionContext, translations)
		if err != nil {
			return fmt.Errorf("同步 OpenAPI 国际化接口: %w", err)
		}
		err = s.baseTenantCase.SyncTenantRoleMenus(transactionContext)
		if err != nil {
			return fmt.Errorf("同步租户角色菜单: %w", err)
		}
		err = s.casbinRuleCase.RebuildPolicyData(transactionContext)
		if err != nil {
			return fmt.Errorf("重建 Casbin 规则: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("同步权限资源事务: %w", err)
	}
	err = s.casbinRuleCase.RefreshPolicy(ctx)
	if err != nil {
		return 0, fmt.Errorf("刷新 Casbin 内存策略: %w", err)
	}
	return len(baseAPIList), nil
}
