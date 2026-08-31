package data

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/models"
)

// BasePolicyEvaluationLogRepository 定义策略评估日志的 Core 写入能力。
type BasePolicyEvaluationLogRepository struct {
	data *Data
}

// NewBasePolicyEvaluationLogRepository 创建策略评估日志仓储实例。
func NewBasePolicyEvaluationLogRepository(data *Data) *BasePolicyEvaluationLogRepository {
	return &BasePolicyEvaluationLogRepository{data: data}
}

// Create 写入一条策略评估日志。
func (r *BasePolicyEvaluationLogRepository) Create(ctx context.Context, item *models.BasePolicyEvaluationLog) error {
	return r.data.DB(ctx).Create(item).Error
}
