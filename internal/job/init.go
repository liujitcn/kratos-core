package job

import "github.com/google/wire"

// ProviderSet 提供 Core 任务注册、持久化调度和生命周期运行时。
var ProviderSet = wire.NewSet(NewRegistry, NewScheduler, NewRuntime)
