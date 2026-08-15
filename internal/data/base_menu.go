package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseMenuRepository 定义 菜单信息 的基础仓储能力。
type BaseMenuRepository struct {
	data *Data
}

// NewBaseMenuRepository 创建 BaseMenu 基础仓储实例。
func NewBaseMenuRepository(data *Data) *BaseMenuRepository {
	return &BaseMenuRepository{
		data: data,
	}
}

// ListByIDs 按菜单 ID 批量查询菜单。
func (r *BaseMenuRepository) ListByIDs(ctx context.Context, ids []int64) ([]*models.BaseMenu, error) {
	if len(ids) == 0 {
		return []*models.BaseMenu{}, nil
	}
	result := make([]*models.BaseMenu, 0, len(ids))
	err := r.data.DB(ctx).Where("id IN ?", ids).Find(&result).Error
	return result, err
}
