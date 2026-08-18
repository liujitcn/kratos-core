package config

import "github.com/google/wire"

// ProviderSet 解析 Bootstrap 总配置，并拆分应用、数据库及各基础设施的分项配置。
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
