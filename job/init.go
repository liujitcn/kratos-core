package job

import (
	"github.com/google/wire"
)

// ProviderSet 创建模块任务注册表、持久化调度器、生命周期服务及任务业务入口。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewScheduler,
	NewServer,
	NewJob,
)
