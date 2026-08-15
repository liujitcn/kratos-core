package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseAPIRepository 定义 API 信息的基础仓储能力。
type BaseAPIRepository struct {
	data *Data
}

// NewBaseAPIRepository 创建 BaseAPI 基础仓储实例。
func NewBaseAPIRepository(data *Data) *BaseAPIRepository {
	return &BaseAPIRepository{data: data}
}

// FindAll 查询全部接口记录。
func (r *BaseAPIRepository) FindAll(ctx context.Context) ([]*models.BaseAPI, error) {
	result := make([]*models.BaseAPI, 0)
	err := r.data.DB(ctx).Find(&result).Error
	return result, err
}

// FindByID 按 API ID 查询接口记录。
func (r *BaseAPIRepository) FindByID(ctx context.Context, id int64) (*models.BaseAPI, error) {
	result := new(models.BaseAPI)
	err := r.data.DB(ctx).First(result, id).Error
	return result, err
}

// FindByOperation 按 HTTP 路径和请求方法查询接口记录。
func (r *BaseAPIRepository) FindByOperation(ctx context.Context, path, method string) (*models.BaseAPI, error) {
	result := new(models.BaseAPI)
	err := r.data.DB(ctx).Where(&models.BaseAPI{Path: path, Method: method}).First(result).Error
	return result, err
}

// Save 保存接口定义，并保留调用方未修改的工具状态和提示词。
func (r *BaseAPIRepository) Save(ctx context.Context, item *models.BaseAPI) error {
	return r.data.DB(ctx).Save(item).Error
}

// Create 创建接口记录。
func (r *BaseAPIRepository) Create(ctx context.Context, item *models.BaseAPI) error {
	return r.data.DB(ctx).Create(item).Error
}

// ReplaceAll 使用当前数据源重建全部接口记录，并通过 TRUNCATE 重置自增 ID。
func (r *BaseAPIRepository) ReplaceAll(ctx context.Context, items []*models.BaseAPI) error {
	return r.data.Transaction(ctx, func(ctx context.Context) error {
		db := r.data.DB(ctx)
		var err error
		err = db.Exec("TRUNCATE TABLE `base_api`").Error //nolint:forbidigo // 重建接口快照并重置自增 ID，GORM 无法表达 TRUNCATE
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return db.Create(&items).Error
	})
}
