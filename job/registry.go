package job

import (
	"github.com/liujitcn/kratos-kit/transport/cron"
)

// Tasks 聚合宿主提供的定时任务执行器。
type Tasks []*cron.Task

// Registry 保存宿主通过 Cron Server 注册的定时任务执行器。
type Registry struct {
	server *cron.Server
}

// NewRegistry 创建 Cron Server 并注册宿主提供的任务执行器。
func NewRegistry(tasks Tasks) (*Registry, error) {
	server := cron.NewServer()
	if err := server.RegisterTask(tasks...); err != nil {
		return nil, err
	}
	return &Registry{server: server}, nil
}

// Server 返回承载任务执行器和数据库任务调度的 Cron Server。
func (r *Registry) Server() *cron.Server {
	if r == nil {
		return nil
	}
	return r.server
}

// Lookup 按名称查询已注册的任务执行器。
func (r *Registry) Lookup(name string) (cron.TaskExec, bool) {
	if r == nil || r.server == nil {
		return nil, false
	}
	task, exists := r.server.LookupTask(name)
	if !exists {
		return nil, false
	}
	return task.Exec, true
}
