package data

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseJobLogRepository 定义定时任务日志信息的基础仓储能力。
type BaseJobLogRepository struct {
	data *Data
}

// NewBaseJobLogRepository 创建 BaseJobLog 基础仓储实例。
func NewBaseJobLogRepository(data *Data) *BaseJobLogRepository {
	return &BaseJobLogRepository{data: data}
}

// Create 创建一条定时任务日志记录。
func (r *BaseJobLogRepository) Create(ctx context.Context, entity *models.BaseJobLog) error {
	if entity == nil {
		return errors.New("entity is nil")
	}
	return r.data.DB(ctx).Create(entity).Error
}
