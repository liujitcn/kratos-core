package biz

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/pprof"
	"github.com/liujitcn/kratos-kit/queue"
	"github.com/liujitcn/kratos-kit/translator"
)

// ProviderSet 提供认证授权、基础设施客户端及 Core 业务共用的 BaseCase。
var ProviderSet = wire.NewSet(
	NewAuthenticator,
	NewAuthzEngine,
	NewUserToken,
	pprof.NewPprof,
	cache.NewCache,
	queue.NewQueue,
	oss.NewOSS,
	translator.NewTranslator,
	NewBaseCase,
)
