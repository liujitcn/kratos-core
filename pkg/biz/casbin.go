package biz

import "context"

// Casbin 定义 Core 提供给宿主业务模块的 Casbin 策略能力。
type Casbin interface {
	// RebuildPolicy 根据当前业务数据重建 Casbin 策略。
	RebuildPolicy(context.Context) error
}
