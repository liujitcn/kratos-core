package job

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/data/models"
	_const "github.com/liujitcn/kratos-core/pkg/const"
	"github.com/liujitcn/kratos-core/pkg/errorsx"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
	"github.com/robfig/cron/v3"
)

type baseJobArgs struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Scheduler 是 Core 唯一的定时任务调度器，负责调度数据库中的 BaseJob。
type Scheduler struct {
	server      *cronTransport.Server
	baseJobRepo *data.BaseJobRepository
	registry    *Registry
	scheduleMu  sync.Mutex
	runContext  context.Context
}

// NewScheduler 创建定时任务调度器。
func NewScheduler(baseJobRepo *data.BaseJobRepository, registry *Registry) *Scheduler {
	var server *cronTransport.Server
	if registry != nil {
		server = registry.Server()
	}
	if server == nil {
		server = cronTransport.NewServer()
	}
	return &Scheduler{
		server:      server,
		baseJobRepo: baseJobRepo,
		registry:    registry,
	}
}

// Server 返回定时任务使用的 Cron 服务。
func (c *Scheduler) Server() *cronTransport.Server {
	if c == nil {
		return nil
	}
	return c.server
}

// Reload 重载数据库中的全部定时任务状态。
func (c *Scheduler) Reload(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	c.scheduleMu.Lock()
	defer c.scheduleMu.Unlock()
	c.runContext = ctx
	return c.reloadJobs(ctx)
}

// ClearExecutionContext 清理定时任务执行上下文。
func (c *Scheduler) ClearExecutionContext() {
	if c == nil {
		return
	}
	c.scheduleMu.Lock()
	c.runContext = nil
	c.scheduleMu.Unlock()
}

// StartJob 启动单个定时任务。
func (c *Scheduler) StartJob(ctx context.Context, baseJob *models.BaseJob) error {
	// 任务实体为空时，无法继续启动调度。
	if baseJob == nil {
		return errorsx.ResourceNotFound("定时任务不存在")
	}
	c.scheduleMu.Lock()
	defer c.scheduleMu.Unlock()
	return c.startJob(ctx, baseJob)
}

// StopJob 停止单个定时任务。
func (c *Scheduler) StopJob(ctx context.Context, baseJob *models.BaseJob) error {
	// 任务实体为空时，无法继续停止调度。
	if baseJob == nil {
		return errorsx.ResourceNotFound("定时任务不存在")
	}
	c.scheduleMu.Lock()
	defer c.scheduleMu.Unlock()
	return c.stopJob(ctx, baseJob)
}

// stopJob 停止一项定时任务并同步持久化状态。
func (c *Scheduler) stopJob(ctx context.Context, baseJob *models.BaseJob) error {
	entryID := baseJob.EntryID
	var err error
	var currentJob *models.BaseJob
	currentJob, err = c.baseJobRepo.FindByID(ctx, baseJob.ID)
	if err != nil {
		return err
	}
	if currentJob.EntryID > 0 {
		entryID = currentJob.EntryID
	}
	err = c.updateBaseJobEntryID(ctx, baseJob.ID, 0)
	if err != nil {
		return err
	}

	// 任务存在调度记录时，先从运行中的 cron 服务里移除。
	if entryID > 0 {
		c.server.StopTimerJob(cron.EntryID(entryID))
	}

	baseJob.EntryID = 0
	return nil
}

// RunJob 立即执行单个定时任务。
func (c *Scheduler) RunJob(ctx context.Context, baseJob *models.BaseJob) error {
	// 任务实体为空时，无法继续执行。
	if baseJob == nil {
		return errorsx.ResourceNotFound("定时任务不存在")
	}

	invokeTarget, err := c.lookupTaskExec(baseJob.InvokeTarget)
	if err != nil {
		// 立即执行在进入任务体前失败时，也要补充失败日志，方便排查配置问题。
		LogFailureWithInput(baseJob.ID, baseJob.Args, err)
		return err
	}

	var argsMap map[string]string
	argsMap, err = parseJobArgs(baseJob.Args)
	if err != nil {
		// 参数解析失败时，保留原始入参到任务日志，便于定位非法配置。
		LogFailureWithInput(baseJob.ID, baseJob.Args, err)
		return err
	}

	execution := &Execution{
		JobID:        baseJob.ID,
		Args:         argsMap,
		InvokeTarget: invokeTarget,
		Context:      ctx,
	}
	return execution.Execute()
}

// startJob 注册一项已完成参数校验的定时任务。
func (c *Scheduler) startJob(ctx context.Context, baseJob *models.BaseJob) error {
	invokeTarget, err := c.lookupTaskExec(baseJob.InvokeTarget)
	if err != nil {
		return err
	}

	var argsMap map[string]string
	argsMap, err = parseJobArgs(baseJob.Args)
	if err != nil {
		return err
	}

	oldEntryID := baseJob.EntryID
	var currentJob *models.BaseJob
	currentJob, err = c.baseJobRepo.FindByID(ctx, baseJob.ID)
	if err != nil {
		return err
	}
	if currentJob.EntryID > 0 {
		oldEntryID = currentJob.EntryID
	}

	jobID := baseJob.ID
	executionContext := c.runContext
	if executionContext == nil {
		executionContext = ctx
	}
	var entryID cron.EntryID
	entryID, err = c.server.StartTimerJob(cronTransport.Spec(baseJob.CronExpression), func() {
		clonedArgs := make(map[string]string, len(argsMap))
		for key, value := range argsMap {
			clonedArgs[key] = value
		}
		execution := &Execution{
			JobID:        jobID,
			Args:         clonedArgs,
			InvokeTarget: invokeTarget,
			Context:      executionContext,
		}
		// 单次调度执行失败时，仅记录错误日志，不影响后续调度。
		execErr := execution.Execute()
		if execErr != nil {
			log.Error(fmt.Sprintf("cron job execute failed, jobID=%d err=%v", jobID, execErr))
		}
	})
	if err != nil {
		return err
	}

	err = c.updateBaseJobEntryID(ctx, baseJob.ID, int32(entryID))
	// 调度记录落库失败时，立即回滚刚注册的内存任务，避免运行态与数据库状态分叉。
	if err != nil {
		c.server.StopTimerJob(entryID)
		return err
	}

	// 新任务成功落库后再移除旧任务，避免注册或落库失败时丢失原有调度。
	if oldEntryID > 0 && oldEntryID != int32(entryID) {
		c.server.StopTimerJob(cron.EntryID(oldEntryID))
	}
	baseJob.EntryID = int32(entryID)
	return nil
}

// reloadJobs 重载数据库中的全部定时任务状态。
func (c *Scheduler) reloadJobs(ctx context.Context) error {
	list, err := c.baseJobRepo.List(ctx)
	if err != nil {
		return err
	}

	startedJobs := make([]*models.BaseJob, 0)
	for _, item := range list {
		// 空任务记录不参与后续重载。
		if item == nil {
			continue
		}

		err = c.stopJob(ctx, item)
		if err != nil {
			return err
		}

		// 未启用任务只重置状态，不重新注册。
		if item.Status != _const.STATUS_STATUS_ENABLE {
			continue
		}

		err = c.startJob(ctx, item)
		// 重载启用任务失败时，直接中断启动流程。
		if err != nil {
			// 回滚失败时仅记录日志，优先保留原始启动错误。
			for i := len(startedJobs) - 1; i >= 0; i-- {
				startedJob := startedJobs[i]
				// 空任务记录不参与回滚。
				if startedJob == nil {
					continue
				}
				rollbackErr := c.stopJob(ctx, startedJob)
				if rollbackErr != nil {
					log.Error(fmt.Sprintf("cron rollback started jobs failed, err=%v", rollbackErr))
					break
				}
			}
			return err
		}
		startedJobs = append(startedJobs, item)
	}
	return nil
}

// lookupTaskExec 按调用目标名称查找任务执行器。
func (c *Scheduler) lookupTaskExec(invokeTarget string) (cronTransport.TaskExec, error) {
	// 调用目标为空时，无法定位实际任务实现。
	if invokeTarget == "" {
		return nil, errorsx.ResourceNotFound("调用目标不存在")
	}

	invokeTargetExec, ok := c.registry.Lookup(invokeTarget)
	// 调用目标未注册时，直接返回明确错误。
	if !ok || invokeTargetExec == nil {
		return nil, errorsx.ResourceNotFound("调用目标不存在")
	}
	return invokeTargetExec, nil
}

// updateBaseJobEntryID 更新任务的调度 entryID。
func (c *Scheduler) updateBaseJobEntryID(ctx context.Context, jobID int64, entryID int32) error {
	return c.baseJobRepo.UpdateEntryID(ctx, jobID, entryID)
}

// parseJobArgs 解析任务参数 JSON 为执行参数 map。
func parseJobArgs(rawArgs string) (map[string]string, error) {
	// 空参数直接返回空 map，避免上层判空分支过多。
	if rawArgs == "" {
		return map[string]string{}, nil
	}

	args := make([]*baseJobArgs, 0)
	err := json.Unmarshal([]byte(rawArgs), &args)
	if err != nil {
		return nil, err
	}

	argsMap := make(map[string]string, len(args))
	for _, item := range args {
		// 空参数项或空 key 不参与最终执行参数组装。
		if item == nil || item.Key == "" {
			continue
		}
		argsMap[item.Key] = item.Value
	}
	return argsMap, nil
}
