package data

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseLogRepository 定义系统日志信息的基础仓储能力。
type BaseLogRepository struct {
	data *Data
}

// NewBaseLogRepository 创建 BaseLog 基础仓储实例。
func NewBaseLogRepository(data *Data) *BaseLogRepository {
	return &BaseLogRepository{data: data}
}

// Create 创建一条系统日志记录。
func (r *BaseLogRepository) Create(ctx context.Context, entity *models.BaseLog) error {
	if entity == nil {
		return errors.New("entity is nil")
	}
	return r.data.DB(ctx).Create(entity).Error
}
