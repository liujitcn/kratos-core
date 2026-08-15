package queue

import (
	"github.com/google/wire"
)

// ProviderSet 提供 Core 队列服务。
var ProviderSet = wire.NewSet(NewServer)
