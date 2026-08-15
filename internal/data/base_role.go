package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseRoleRepository 定义 角色信息 的基础仓储能力。
type BaseRoleRepository struct {
	data *Data
}

// NewBaseRoleRepository 创建 BaseRole 基础仓储实例。
func NewBaseRoleRepository(data *Data) *BaseRoleRepository {
	return &BaseRoleRepository{
		data: data,
	}
}

// List 查询全部角色。
func (r *BaseRoleRepository) List(ctx context.Context) ([]*models.BaseRole, error) {
	result := make([]*models.BaseRole, 0)
	err := r.data.DB(ctx).Find(&result).Error
	return result, err
}

// FindByTenantIDAndCode 按租户和角色编码查询角色。
func (r *BaseRoleRepository) FindByTenantIDAndCode(ctx context.Context, tenantID int64, code string) (*models.BaseRole, error) {
	result := new(models.BaseRole)
	err := r.data.DB(ctx).
		Where("tenant_id = ?", tenantID).
		Where("code = ?", code).
		First(result).Error
	return result, err
}

// ListByCode 查询指定角色编码的全部角色。
func (r *BaseRoleRepository) ListByCode(ctx context.Context, code string) ([]*models.BaseRole, error) {
	result := make([]*models.BaseRole, 0)
	err := r.data.DB(ctx).Where("code = ?", code).Find(&result).Error
	return result, err
}

// UpdateMenus 更新角色关联的菜单 JSON。
func (r *BaseRoleRepository) UpdateMenus(ctx context.Context, role *models.BaseRole) error {
	return r.data.DB(ctx).
		Model(&models.BaseRole{}).
		Where("id = ?", role.ID).
		Updates(map[string]any{"tenant_id": role.TenantID, "menus": role.Menus}).Error
}
