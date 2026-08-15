package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseTenantRepository 定义 租户信息表 的基础仓储能力。
type BaseTenantRepository struct {
	data *Data
}

// NewBaseTenantRepository 创建 BaseTenant 基础仓储实例。
func NewBaseTenantRepository(data *Data) *BaseTenantRepository {
	return &BaseTenantRepository{
		data: data,
	}
}

// FindByCode 按租户编码查询租户。
func (r *BaseTenantRepository) FindByCode(ctx context.Context, code string) (*models.BaseTenant, error) {
	result := new(models.BaseTenant)
	err := r.data.DB(ctx).Where("code = ?", code).First(result).Error
	return result, err
}

// List 查询全部租户。
func (r *BaseTenantRepository) List(ctx context.Context) ([]*models.BaseTenant, error) {
	result := make([]*models.BaseTenant, 0)
	err := r.data.DB(ctx).Find(&result).Error
	return result, err
}
