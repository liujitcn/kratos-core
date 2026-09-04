package data

import (
	"context"
	"time"
)

// JobRecord 表示 Core 调度任务所需的字段。
type JobRecord struct {
	// ID 是任务编号。
	ID int64
	// InvokeTarget 是任务执行器名称。
	InvokeTarget string
	// Args 是任务参数 JSON。
	Args string
	// CronExpression 是 Cron 表达式。
	CronExpression string
	// Status 是任务状态。
	Status int32
	// EntryID 是运行中的 Cron 入口编号。
	EntryID int32
}

// JobLogRecord 表示 Core 任务执行日志所需的字段。
type JobLogRecord struct {
	// ID 是日志编号。
	ID int64 `json:"id"`
	// JobID 是任务编号。
	JobID int64 `json:"job_id"`
	// Input 是执行参数。
	Input string `json:"input"`
	// Output 是执行输出。
	Output string `json:"output"`
	// Error 是执行错误。
	Error string `json:"error"`
	// Status 是执行状态。
	Status int32 `json:"status"`
	// ProcessTime 是执行耗时，单位为毫秒。
	ProcessTime int32 `json:"process_time"`
	// ExecuteTime 是执行时间。
	ExecuteTime time.Time `json:"execute_time"`
}

// JobStore 提供任务调度和任务日志所需的宿主持久化能力。
type JobStore interface {
	// List 查询全部任务。
	List(context.Context) ([]JobRecord, error)
	// FindByID 按任务编号查询任务。
	FindByID(context.Context, int64) (JobRecord, error)
	// UpdateEntryID 更新任务的 Cron 入口编号。
	UpdateEntryID(context.Context, int64, int32) error
	// CreateLog 创建任务执行日志。
	CreateLog(context.Context, JobLogRecord) error
}
