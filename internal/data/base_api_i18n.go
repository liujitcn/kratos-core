package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// BaseAPII18nRepository 定义 API 国际化信息的基础仓储能力。
type BaseAPII18nRepository struct {
	data *Data
}

// NewBaseAPII18nRepository 创建 API 国际化基础仓储实例。
func NewBaseAPII18nRepository(data *Data) *BaseAPII18nRepository {
	return &BaseAPII18nRepository{data: data}
}

// ReplaceAll 使用当前 OpenAPI 语言快照重建 API 国际化记录。
func (r *BaseAPII18nRepository) ReplaceAll(ctx context.Context, items []*models.BaseAPII18n) error {
	return r.data.Transaction(ctx, func(ctx context.Context) error {
		db := r.data.DB(ctx)
		var err error
		err = db.Exec("DELETE FROM `base_api_i18n`").Error //nolint:forbidigo // 事务内重建接口翻译快照
		if err != nil {
			return err
		}
		translatedItems := make([]*models.BaseAPII18n, 0, len(items))
		for _, item := range items {
			if item == nil || item.Locale == "" {
				continue
			}
			translatedItems = append(translatedItems, item)
		}
		if len(translatedItems) == 0 {
			return nil
		}
		return db.Create(&translatedItems).Error
	})
}
