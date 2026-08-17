package biz

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/resource/biz"
)

// Casbin 提供 Casbin 权限规则重建能力。
type Casbin struct {
	*biz.CasbinRuleCase
}

// NewCasbin 创建权限规则业务实例。
func NewCasbin(
	casbinRuleCase *biz.CasbinRuleCase,
) (*Casbin, error) {
	return &Casbin{
		CasbinRuleCase: casbinRuleCase,
	}, nil
}

// RebuildPolicy 按角色、菜单和 API 全量重建 Casbin 规则与内存策略。
func (c *Casbin) RebuildPolicy(ctx context.Context) error {
	err := c.RebuildPolicyData(ctx)
	if err != nil {
		return err
	}
	return c.RefreshPolicy(ctx)
}
