package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/models"
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

// ReplaceAll 使用当前数据源重建全部接口记录，并保留已有 API 的运行时配置。
func (r *BaseAPIRepository) ReplaceAll(ctx context.Context, items []*models.BaseAPI) error {
	return r.data.Transaction(ctx, func(ctx context.Context) error {
		db := r.data.DB(ctx)
		var err error
		if len(items) > 0 {
			var existing []*models.BaseAPI
			err = db.Find(&existing).Error
			if err != nil {
				return err
			}
			preserveBaseAPISettings(items, existing)
		}
		err = db.Exec("DELETE FROM `base_api`").Error //nolint:forbidigo // 事务内重建接口快照，避免 TRUNCATE 的隐式提交
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return db.Create(&items).Error
	})
}

// preserveBaseAPISettings 将已有 API 的稳定标识和运行时配置复制到新的快照。
func preserveBaseAPISettings(items, existing []*models.BaseAPI) {
	existingByKey := make(map[string]*models.BaseAPI, len(existing))
	for _, item := range existing {
		if item == nil {
			continue
		}
		key := baseAPIIdentity(item)
		if key == "" {
			continue
		}
		if _, exists := existingByKey[key]; !exists {
			existingByKey[key] = item
		}
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		existingItem, exists := existingByKey[baseAPIIdentity(item)]
		if !exists {
			continue
		}
		item.ID = existingItem.ID
		item.McpStatus = existingItem.McpStatus
		item.AgentStatus = existingItem.AgentStatus
		item.ToolPrompts = existingItem.ToolPrompts
	}
}

// baseAPIIdentity 使用 OpenAPI operation 作为 API 快照的稳定关联键。
func baseAPIIdentity(item *models.BaseAPI) string {
	if item == nil {
		return ""
	}
	return item.Operation
}
