package biz

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/data/models"
	_const "github.com/liujitcn/kratos-core/pkg/const"
	"github.com/liujitcn/kratos-core/pkg/errorsx"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

// BaseTenantCase 提供启动期租户基础数据同步能力。
type BaseTenantCase struct {
	tx             data.Transaction
	baseRoleRepo   *data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
}

// NewBaseTenantCase 创建启动期租户基础数据同步实例。
func NewBaseTenantCase(
	tx data.Transaction,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
) *BaseTenantCase {
	return &BaseTenantCase{
		tx:             tx,
		baseRoleRepo:   baseRoleRepo,
		baseTenantRepo: baseTenantRepo,
	}
}

// SyncTenantRoleMenus 将默认租户管理员角色菜单同步到所有普通租户的角色副本。
//
// 该方法仅在服务启动时调用，必须位于 OpenAPI 接口同步之后、全量 Casbin 规则重建之前。
// 默认租户或角色模板尚未初始化时返回 nil，使首次导入初始化数据前的启动流程保持幂等。
func (c *BaseTenantCase) SyncTenantRoleMenus(ctx context.Context) error {
	defaultTenant, err := c.baseTenantRepo.FindByCode(ctx, databaseGorm.DefaultTenantCode)
	// 首次启动尚未导入初始化数据时没有默认租户，等待后续启动再同步。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return errorsx.Internal("查询默认租户失败").WithCause(err)
	}

	var templateRole *models.BaseRole
	templateRole, err = c.baseRoleRepo.FindByTenantIDAndCode(ctx, defaultTenant.ID, _const.BASE_ROLE_CODE_TENANT)
	// 初始化数据尚未写入租户角色模板时无需执行同步。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		var baseRoleList []*models.BaseRole
		baseRoleList, err = c.baseRoleRepo.ListByCode(ctx, _const.BASE_ROLE_CODE_TENANT)
		if err != nil {
			return err
		}
		for _, item := range baseRoleList {
			// 默认租户模板和已同步副本无需重复写入。
			if item.ID == templateRole.ID || item.Menus == templateRole.Menus {
				continue
			}
			err = c.baseRoleRepo.UpdateMenus(ctx, &models.BaseRole{
				ID:       item.ID,
				TenantID: item.TenantID,
				Menus:    templateRole.Menus,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
