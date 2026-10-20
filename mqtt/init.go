package mqtt

import "github.com/google/wire"

// ProviderSet 创建 Core MQTT 运行服务。
var ProviderSet = wire.NewSet(NewServer)
