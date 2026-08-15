package job

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/pkg/biz"
)

// ProviderSet 提供 Core 任务注册、持久化调度和任务服务。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewScheduler,
	NewServer,
	NewJob,
	wire.Bind(new(biz.Job), new(*Job)),
)
