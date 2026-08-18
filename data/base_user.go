package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/models"
)

// BaseUserRepository 定义用户信息的基础仓储能力。
type BaseUserRepository struct {
	data *Data
}

// NewBaseUserRepository 创建 BaseUser 基础仓储实例。
func NewBaseUserRepository(data *Data) *BaseUserRepository {
	return &BaseUserRepository{data: data}
}

// FindByTenantCodeAndUserName 按租户编码和用户账号查询用户。
func (r *BaseUserRepository) FindByTenantCodeAndUserName(ctx context.Context, tenantCode, userName string) (*models.BaseUser, error) {
	result := new(models.BaseUser)
	err := r.data.DB(ctx).
		Model(result).
		Joins("JOIN base_tenant ON base_tenant.id = base_user.tenant_id").
		Where("base_tenant.code = ?", tenantCode).
		Where("base_user.user_name = ?", userName).
		First(result).Error
	return result, err
}
