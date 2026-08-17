package middleware

import (
	"context"

	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	i18n2 "github.com/liujitcn/kratos-core/internal/resource/i18n"
	"github.com/liujitcn/kratos-kit/cache"
)

// NewI18nCatalogMiddleware 使用 Core 已装载的国际化目录创建本地化拦截器。
func NewI18nCatalogMiddleware(catalog *i18n2.I18n, cache cache.Cache) kratosMiddleware.Middleware {
	if catalog.Empty() {
		return nil
	}
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			localeValue := ""
			if transporter, ok := transport.FromServerContext(ctx); ok {
				localeValue = transporter.RequestHeader().Get("Accept-Language")
			}
			var fallbackLocale string
			var err error
			fallbackLocale, err = cache.Get(_const.I18nDefaultLanguage)
			if err != nil {
				fallbackLocale = "zh-CN"
			}
			ctx = biz.WithLocale(ctx, localeValue)
			var reply any
			reply, err = handler(ctx, req)
			if err != nil {
				return nil, i18n2.LocalizeError(catalog, localeValue, fallbackLocale, err)
			}
			return reply, nil
		}
	}
}
