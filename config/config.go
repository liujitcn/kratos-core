package config

import (
	"errors"
	"fmt"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// ParseBootstrap 解析启动上下文中的完整配置。
func ParseBootstrap(ctx *bootstrap.Context) (*configv1.Bootstrap, error) {
	if ctx == nil || ctx.GetConfig() == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	return ctx.GetConfig(), nil
}

// ParsePprof 解析性能分析配置。
func ParsePprof(cfg *configv1.Bootstrap) (*configv1.Pprof, error) {
	if cfg == nil || cfg.GetPprof() == nil {
		return nil, errors.New("性能分析配置不能为空")
	}
	return cfg.GetPprof(), nil
}

// ParseAppInfo 解析启动上下文中的应用信息。
func ParseAppInfo(ctx *bootstrap.Context) (*configv1.AppInfo, error) {
	if ctx == nil || ctx.GetAppInfo() == nil {
		return nil, errors.New("应用信息不能为空")
	}
	return ctx.GetAppInfo(), nil
}

// ParseOSS 解析对象存储配置。
func ParseOSS(cfg *configv1.Bootstrap) (*configv1.Oss, error) {
	if cfg == nil || cfg.GetOss() == nil {
		return nil, errors.New("对象存储配置不能为空")
	}
	return cfg.GetOss(), nil
}

// ParseQueue 解析队列配置。
func ParseQueue(cfg *configv1.Bootstrap) (*configv1.Data_Queue, error) {
	if cfg == nil || cfg.GetData() == nil || cfg.GetData().GetQueue() == nil {
		return nil, errors.New("队列配置不能为空")
	}
	return cfg.GetData().GetQueue(), nil
}

// ParseRedis 解析 Redis 配置。
func ParseRedis(cfg *configv1.Bootstrap) (*configv1.Data_Redis, error) {
	if cfg == nil || cfg.GetData() == nil || cfg.GetData().GetRedis() == nil {
		return nil, errors.New("Redis配置不能为空")
	}
	return cfg.GetData().GetRedis(), nil
}

// ParseDatabase 解析启动配置中的全部数据库连接。
func ParseDatabase(config *configv1.Bootstrap) (map[string]*configv1.Data_Database, error) {
	if config == nil {
		return nil, errors.New("配置不能为空")
	}
	data := config.GetData()
	if data == nil {
		return nil, errors.New("config[data] is nil")
	}

	databases := make(map[string]*configv1.Data_Database, len(data.GetDatabases())+1)
	for name, database := range data.GetDatabases() {
		if database != nil {
			databases[name] = database
		}
	}
	if database := data.GetDatabase(); database != nil {
		databases[gorm.DefaultClientName] = database
	}
	if len(databases) == 0 {
		return nil, errors.New("config[databases] is empty")
	}
	return databases, nil
}

// ParseAuthnJWT 解析 JWT 认证配置。
func ParseAuthnJWT(cfg *configv1.Bootstrap) (*configv1.Authentication_Jwt, error) {
	if cfg == nil || cfg.GetAuthn() == nil || cfg.GetAuthn().GetJwt() == nil {
		return nil, errors.New("JWT认证配置不能为空")
	}
	jwtConfig := cfg.GetAuthn().GetJwt()
	if jwtConfig.GetSecret() == "" {
		return nil, errors.New("JWT密钥不能为空")
	}
	return jwtConfig, nil
}

// ParseTranslator 解析机器翻译配置。
func ParseTranslator(cfg *configv1.Bootstrap) (*configv1.Translator, error) {
	if cfg == nil || cfg.GetTranslator() == nil {
		return nil, errors.New("机器翻译配置不能为空")
	}
	return cfg.GetTranslator(), nil
}
