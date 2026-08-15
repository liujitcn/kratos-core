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

// ReplaceAll 使用当前数据源重建全部 Casbin 规则，并在事务内删除旧快照。
func (r *CasbinRuleRepository) ReplaceAll(ctx context.Context, items []*models.CasbinRule) error {
	return r.data.Transaction(ctx, func(ctx context.Context) error {
		db := r.data.DB(ctx)
		var err error
		err = db.Exec("DELETE FROM `casbin_rule`").Error //nolint:forbidigo // 事务内重建权限快照，避免 TRUNCATE 的隐式提交
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return db.Create(&items).Error
	})
}
