package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
)

// CasbinRuleRepository 定义 Casbin 权限规则的基础仓储能力。
type CasbinRuleRepository struct {
	data *Data
}

// NewCasbinRuleRepository 创建 Casbin 权限规则仓储实例。
func NewCasbinRuleRepository(data *Data) *CasbinRuleRepository {
	return &CasbinRuleRepository{data: data}
}

// FindAll 查询全部 Casbin 权限规则。
func (r *CasbinRuleRepository) FindAll(ctx context.Context) ([]*models.CasbinRule, error) {
	result := make([]*models.CasbinRule, 0)
	err := r.data.DB(ctx).Find(&result).Error
	return result, err
}

// ReplaceAll 使用当前数据源重建全部 Casbin 规则，并通过 TRUNCATE 重置自增 ID。
func (r *CasbinRuleRepository) ReplaceAll(ctx context.Context, items []*models.CasbinRule) error {
	return r.data.Transaction(ctx, func(ctx context.Context) error {
		db := r.data.DB(ctx)
		var err error
		err = db.Exec("TRUNCATE TABLE `casbin_rule`").Error //nolint:forbidigo // 重建规则并重置自增 ID，GORM 无法表达 TRUNCATE
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return db.Create(&items).Error
	})
}
