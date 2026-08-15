package _const

import "github.com/liujitcn/kratos-kit/transport/queue"

const (
	// LOG 表示通用日志异步写入队列。
	LOG queue.Stream = "log_queue"
	// JOB_LOG 表示定时任务执行日志队列。
	JOB_LOG queue.Stream = "job_log_queue"
)
