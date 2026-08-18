package queue

import (
	"github.com/google/wire"
)

// ProviderSet 创建队列消费服务，并注册 Core 与宿主提供的消费者。
var ProviderSet = wire.NewSet(NewServer)
