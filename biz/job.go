package biz

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/data/models"
	"github.com/liujitcn/kratos-core/internal/job"
)

// Job 实现 Core 的持久化任务能力。
type Job struct {
	server *job.Scheduler
}

// NewJob 创建持久化任务服务。
func NewJob(server *job.Scheduler) *Job {
	return &Job{
		server: server,
	}
}

// Start 注册持久化任务并返回新的调度入口编号。
func (s *Job) Start(ctx context.Context, jobID int64, cronExpression string, invokeTarget string, args string, entryID int32) (int32, error) {
	baseJob := &models.BaseJob{
		ID:             jobID,
		CronExpression: cronExpression,
		InvokeTarget:   invokeTarget,
		Args:           args,
		EntryID:        entryID,
	}
	err := s.server.StartJob(ctx, baseJob)
	if err != nil {
		return 0, err
	}
	return baseJob.EntryID, nil
}

// Stop 移除持久化任务的调度并清空持久化入口编号。
func (s *Job) Stop(ctx context.Context, jobID int64, entryID int32) error {
	baseJob := &models.BaseJob{
		ID:      jobID,
		EntryID: entryID,
	}
	return s.server.StopJob(ctx, baseJob)
}

// Run 立即执行一次持久化任务。
func (s *Job) Run(ctx context.Context, jobID int64, invokeTarget string, args string) error {
	baseJob := &models.BaseJob{
		ID:           jobID,
		InvokeTarget: invokeTarget,
		Args:         args,
	}
	return s.server.RunJob(ctx, baseJob)
}
