package biz

import (
	"context"

	utilsTranslator "github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/kratos-core/pkg/errorsx"
	"github.com/liujitcn/kratos-kit/auth"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/cache"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/pprof"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	"github.com/liujitcn/kratos-kit/sdk"
)

// BaseCase 承载所有业务模块共用的启动上下文、基础对象和请求认证信息读取能力。
type BaseCase struct {
	*bootstrap.Context
	// pprof Go 官方提供的性能分析组件
	pprof pprof.Pprof
	// Cache 是 Core 创建的缓存实例。
	Cache cache.Cache
	// Queue 是 Core 创建的队列实例。
	Queue kitQueue.Queue
	// OSS 是 Core 创建的对象存储实例。
	OSS oss.OSS
	// Translator 是 Core 创建的翻译器实例。
	Translator utilsTranslator.Translator
	// GormClients Gorm多数据源客户端
	GormClients map[string]*databaseGorm.Client
}

// NewBaseCase 创建供业务模块复用的基础业务实例，并同步基础对象到 SDK Runtime。
func NewBaseCase(
	ctx *bootstrap.Context,
	pprof pprof.Pprof,
	cacheValue cache.Cache,
	queueValue kitQueue.Queue,
	ossValue oss.OSS,
	translatorValue utilsTranslator.Translator,
	gormClients map[string]*databaseGorm.Client,
) (*BaseCase, func()) {
	sdk.Runtime.SetCache(cacheValue)
	sdk.Runtime.SetQueue(queueValue)
	sdk.Runtime.SetOSS(ossValue)
	sdk.Runtime.SetTranslator(translatorValue)
	sdk.Runtime.SetGormClients(gormClients)

	if pprof != nil {
		pprof.Start()
	}
	return &BaseCase{
			Context:     ctx,
			Cache:       cacheValue,
			Queue:       queueValue,
			OSS:         ossValue,
			Translator:  translatorValue,
			GormClients: gormClients,
		}, func() {
			if pprof != nil {
				pprof.Stop()
			}
		}
}

// GetAuthInfo 获取当前登录用户认证信息。
func (c *BaseCase) GetAuthInfo(ctx context.Context) (*authData.UserTokenPayload, error) {
	authInfo, err := auth.FromContext(ctx)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户认证失败").WithCause(err)
	}
	return authInfo, nil
}
