package biz

import (
	"context"
	"errors"

	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/errorsx"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

// BaseTenantCase 提供启动期租户基础数据同步能力。
type BaseTenantCase struct {
	tx              data.Transaction
	permissionStore data.PermissionStore
}

// NewBaseTenantCase 创建启动期租户基础数据同步实例。
func NewBaseTenantCase(
	tx data.Transaction,
	permissionStore data.PermissionStore,
) *BaseTenantCase {
	return &BaseTenantCase{
		tx:              tx,
		permissionStore: permissionStore,
	}
}

// SyncTenantRoleMenus 将默认租户管理员角色菜单同步到所有普通租户的角色副本。
//
// 该方法仅在服务启动时调用，必须位于 OpenAPI 接口同步之后、全量 Casbin 规则重建之前。
// 默认租户或角色模板尚未初始化时返回 nil，使首次导入初始化数据前的启动流程保持幂等。
func (c *BaseTenantCase) SyncTenantRoleMenus(ctx context.Context) error {
	defaultTenant, err := c.permissionStore.FindTenantByCode(ctx, databaseGorm.DefaultTenantCode)
	// 首次启动尚未导入初始化数据时没有默认租户，等待后续启动再同步。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return errorsx.Internal("查询默认租户失败").WithCause(err)
	}

	var templateRole data.RoleRecord
	templateRole, err = c.permissionStore.FindRoleByTenantIDAndCode(ctx, defaultTenant.ID, _const.BASE_ROLE_CODE_TENANT)
	// 初始化数据尚未写入租户角色模板时无需执行同步。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		var baseRoleList []data.RoleRecord
		baseRoleList, err = c.permissionStore.ListRolesByCode(ctx, _const.BASE_ROLE_CODE_TENANT)
		if err != nil {
			return err
		}
		for _, item := range baseRoleList {
			// 默认租户模板和已同步副本无需重复写入。
			if item.ID == templateRole.ID || item.Menus == templateRole.Menus {
				continue
			}
			err = c.permissionStore.UpdateRoleMenus(ctx, data.RoleRecord{
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
