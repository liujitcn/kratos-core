package config

import "github.com/google/wire"

// ProviderSet 提供数据库之外的启动配置解析能力。
var ProviderSet = wire.NewSet(
	ParseBootstrap,
	ParseAppInfo,
	ParseDatabase,
	ParseOSS,
	ParseRedis,
	ParseQueue,
	ParsePprof,
	ParseAuthnJWT,
	ParseTranslator,
)
