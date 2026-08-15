package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseJobRepository 定义定时任务信息的基础仓储能力。
type BaseJobRepository struct {
	data *Data
}

// NewBaseJobRepository 创建 BaseJob 基础仓储实例。
func NewBaseJobRepository(data *Data) *BaseJobRepository {
	return &BaseJobRepository{data: data}
}

// List 查询全部定时任务记录。
func (r *BaseJobRepository) List(ctx context.Context) ([]*models.BaseJob, error) {
	result := make([]*models.BaseJob, 0)
	err := r.data.DB(ctx).Find(&result).Error
	return result, err
}

// FindByID 按任务 ID 查询定时任务记录。
func (r *BaseJobRepository) FindByID(ctx context.Context, jobID int64) (*models.BaseJob, error) {
	result := new(models.BaseJob)
	err := r.data.DB(ctx).Where("id = ?", jobID).First(result).Error
	return result, err
}

// UpdateEntryID 更新定时任务的调度入口编号。
func (r *BaseJobRepository) UpdateEntryID(ctx context.Context, jobID int64, entryID int32) error {
	return r.data.DB(ctx).Model(&models.BaseJob{}).
		Where("id = ?", jobID).
		Update("entry_id", entryID).Error
}
