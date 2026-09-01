package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/models"
)

// BaseAPILogRepository 定义 API 访问日志的 Core 写入能力。
type BaseAPILogRepository struct {
	data *Data
}

// NewBaseAPILogRepository 创建 API 访问日志仓储实例。
func NewBaseAPILogRepository(data *Data) *BaseAPILogRepository {
	return &BaseAPILogRepository{data: data}
}

// Create 写入一条 API 访问日志。
func (r *BaseAPILogRepository) Create(ctx context.Context, item *models.BaseAPILog) error {
	return r.data.DB(ctx).Create(item).Error
}

// FindByID 按主键查询 API 访问日志，用于确认重复队列投递。
func (r *BaseAPILogRepository) FindByID(ctx context.Context, id int64) (*models.BaseAPILog, error) {
	item := &models.BaseAPILog{}
	err := r.data.DB(ctx).First(item, id).Error
	return item, err
}
