package biz

import "context"

// Job 定义 Core 提供给宿主业务模块的持久化任务能力。
type Job interface {
	// Start 注册持久化任务并返回新的调度入口编号。
	Start(context.Context, int64, string, string, string, int32) (int32, error)
	// Stop 移除持久化任务的调度并清空持久化入口编号。
	Stop(context.Context, int64, int32) error
	// Run 立即执行一次持久化任务。
	Run(context.Context, int64, string, string) error
}
