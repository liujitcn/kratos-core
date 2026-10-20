package mqtt

import (
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	mqtttransport "github.com/liujitcn/kratos-kit/transport/mqtt"
)

// Server 表示 Core 统一管理生命周期的 MQTT 订阅服务。
type Server = mqtttransport.Server

// NewServer 按 server.mqtt 配置创建 MQTT 订阅服务。
func NewServer(cfg *configv1.Bootstrap) (*Server, error) {
	if cfg == nil || cfg.GetServer() == nil {
		return nil, nil
	}
	return mqtttransport.NewServerFromConfig(cfg.GetServer().GetMqtt())
}
