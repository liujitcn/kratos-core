package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	_string "github.com/liujitcn/go-utils/string"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/internal/data/models"
	"github.com/liujitcn/kratos-core/queue"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
)

// Execution 表示一次定时任务执行上下文。
type Execution struct {
	// JobID 是任务 ID。
	JobID int64
	// Args 是任务参数。
	Args map[string]string
	// InvokeTarget 是任务执行器。
	InvokeTarget cronTransport.TaskExec
	// Context 是任务执行上下文。
	Context context.Context
	// Status 是任务执行状态。
	Status commonv1.BaseJobLogStatus
	// ErrMsg 是任务执行错误信息。
	ErrMsg string
}

// LogFailureWithInput 使用原始入参记录任务失败日志。
func LogFailureWithInput(jobID int64, input string, err error) {
	if err == nil {
		return
	}

	baseJobLog := models.BaseJobLog{
		JobID:       jobID,
		Input:       input,
		ExecuteTime: time.Now(),
	}
	baseJobLog.Status = _const.BASE_JOB_LOG_STATUS_FAIL
	baseJobLog.Error = err.Error()
	baseJobLog.ProcessTime = int32(time.Since(baseJobLog.ExecuteTime).Milliseconds())
	queue.AddQueue(_const.JOB_LOG, baseJobLog)
}

// Execute 执行任务并写入任务日志。
func (e *Execution) Execute() (err error) {
	baseJobLog := models.BaseJobLog{
		JobID:       e.JobID,
		Input:       _string.ConvertAnyToJsonString(e.Args),
		ExecuteTime: time.Now(),
	}
	ret := make([]string, 0)

	defer func() {
		if panicValue := recover(); panicValue != nil {
			err = fmt.Errorf("任务执行异常: %v", panicValue)
		}

		if err != nil {
			e.Status = commonv1.BaseJobLogStatus(_const.BASE_JOB_LOG_STATUS_FAIL)
			e.ErrMsg = err.Error()
		} else {
			e.Status = commonv1.BaseJobLogStatus(_const.BASE_JOB_LOG_STATUS_SUCCESS)
			e.ErrMsg = ""
		}
		baseJobLog.Output = strings.Join(ret, "<br/>")
		baseJobLog.Status = int32(e.Status)
		baseJobLog.Error = e.ErrMsg
		baseJobLog.ProcessTime = int32(time.Since(baseJobLog.ExecuteTime).Milliseconds())
		queue.AddQueue(_const.JOB_LOG, baseJobLog)
	}()

	runContext := e.Context
	if runContext == nil {
		runContext = context.Background()
	}
	ret, err = e.InvokeTarget.Exec(runContext, e.Args)
	return err
}
