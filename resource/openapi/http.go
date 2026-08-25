package openapi

import (
	"fmt"
	"net/http"
	"strings"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	swaggerUI "github.com/liujitcn/kratos-kit/swagger-ui"
)

const (
	defaultDocumentPath = "/api/docs/openapi"
	defaultSwaggerPath  = "/api/docs/swagger"
)

// HTTPOptions 配置 OpenAPI 原文和 Swagger UI 路由。
type HTTPOptions struct {
	// DocumentPath 是原始 OpenAPI 文档的路由前缀。
	DocumentPath string
	// SwaggerPath 是 Swagger UI 的路由前缀。
	SwaggerPath string
	// Authorizer 校验原始文档请求，nil 表示允许访问。
	Authorizer func(*http.Request) bool
}

// RegisterHTTP 为注册表中的每份文档挂载原文接口和独立 Swagger UI。
func RegisterHTTP(server *kratosHTTP.Server, registry *Registry, opts HTTPOptions) error {
	documentPath := normalizePath(opts.DocumentPath, defaultDocumentPath)
	swaggerPath := normalizePath(opts.SwaggerPath, defaultSwaggerPath)
	authorizer := opts.Authorizer
	if authorizer == nil {
		authorizer = func(*http.Request) bool { return true }
	}

	var err error
	locales := append([]string{""}, registry.Locales()...)
	for _, locale := range locales {
		for _, document := range registry.DocumentsByLocale(locale) {
			rawPath := documentPath + "/" + document.Key + documentLocalePath(locale)
			err = swaggerUI.RegisterOpenAPIServerWithOption(
				server,
				swaggerUI.WithOpenAPIPath(rawPath),
				swaggerUI.WithMemoryData(document.Data, "yaml"),
				swaggerUI.WithOpenAPIAuthorizer(authorizer),
			)
			if err != nil {
				return fmt.Errorf("注册 OpenAPI 文档: %w", err)
			}
			err = swaggerUI.RegisterSwaggerUIServerWithOption(
				server,
				swaggerUI.WithTitle(document.Name),
				swaggerUI.WithBasePath(swaggerPath+"/"+document.Key+documentLocalePath(locale)+"/"),
				swaggerUI.WithRemoteFileURL(rawPath),
			)
			if err != nil {
				return fmt.Errorf("注册 Swagger UI: %w", err)
			}
		}
	}
	return nil
}

// documentLocalePath 返回语言文档在 HTTP 路由中的可选路径段。
func documentLocalePath(locale string) string {
	if locale == "" {
		return ""
	}
	return "/" + locale
}

// normalizePath 将自定义 HTTP 路径规范化为带前导斜杠的路由前缀。
func normalizePath(value, fallback string) string {
	value = "/" + strings.Trim(value, "/")
	if value == "/" {
		return fallback
	}
	return value
}
