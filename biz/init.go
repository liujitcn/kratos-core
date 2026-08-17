package biz

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/pprof"
	"github.com/liujitcn/kratos-kit/queue"
	"github.com/liujitcn/kratos-kit/translator"
)

// ProviderSet 提供 Core 业务能力共用的基础对象和业务实例。
var ProviderSet = wire.NewSet(
	pprof.NewPprof,
	cache.NewCache,
	queue.NewQueue,
	oss.NewOSS,
	translator.NewTranslator,
	NewClients,
	NewBaseCase,
	NewAuthenticator,
	NewAuthzEngine,
	NewUserToken,
	NewCasbin,
	NewJob,
	NewOpenAPI,
	NewDocs,
	NewSSE,
)
