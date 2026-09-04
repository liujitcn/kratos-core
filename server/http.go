package server

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-core/mcp"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/resource/i18n"
	"github.com/liujitcn/kratos-core/resource/openapi"
	coreMiddleware "github.com/liujitcn/kratos-core/server/middleware"
	"github.com/liujitcn/kratos-core/server/middleware/logging"
	"github.com/liujitcn/kratos-core/sse"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/oss"
	serverhttp "github.com/liujitcn/kratos-kit/server/http"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

// HTTPMiddlewares 表示 HTTP 服务中间件链。
type HTTPMiddlewares []middleware.Middleware

const defaultStaticRootDirectory = "./data"

// NewHTTPMiddleware 创建 HTTP 服务统一中间件链。
func NewHTTPMiddleware(
	ctx *bootstrap.Context,
	authenticator engine.Authenticator,
	authorizer authzEngine.Engine,
	userToken *data.UserToken,
	jwtCfg *configv1.Authentication_Jwt,
	cache cache.Cache,
	catalog *i18n.I18n,
) HTTPMiddlewares {
	var httpMiddlewares HTTPMiddlewares
	cfg := ctx.GetConfig()
	// i18n国际化
	if i18nMiddleware := coreMiddleware.NewI18nCatalogMiddleware(catalog, cache); i18nMiddleware != nil {
		httpMiddlewares = append(httpMiddlewares, i18nMiddleware)
	}
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Http != nil && cfg.Server.Http.Middleware != nil && cfg.Server.Http.Middleware.EnableLogging && authenticator != nil {
		httpMiddlewares = append(httpMiddlewares, logging.Server(ctx.GetLogger(), authenticator))
	}
	if authenticator != nil && authorizer != nil && userToken != nil && jwtCfg != nil {
		httpMiddlewares = append(httpMiddlewares, coreMiddleware.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	}
	// 按 HTTP 服务配置挂载 Core 的校验错误转换，避免未启用时处理校验错误。
	if cfg != nil && cfg.Server != nil && cfg.Server.Http != nil && cfg.Server.Http.Middleware != nil && cfg.Server.Http.Middleware.GetEnableValidate() {
		httpMiddlewares = append(httpMiddlewares, coreMiddleware.NewValidateMiddleware())
	}
	return httpMiddlewares
}

// NewHTTPServer 创建 HTTP Server 并注册已启用业务模块与前端静态路由。
func NewHTTPServer(
	ctx *bootstrap.Context,
	appInfo *configv1.AppInfo,
	middlewares HTTPMiddlewares,
	modules module.Modules,
	authenticator engine.Authenticator,
	userToken *data.UserToken,
	openAPIRegistry *openapi.Registry,
	mcpServer *mcp.Server,
	sseServer *sse.Server,
) (transport.Server, error) {
	cfg := ctx.GetConfig()
	httpConfigured := cfg != nil && cfg.Server != nil && cfg.Server.Http != nil
	err := validateInProcessHTTPHost(httpConfigured, mcpServer, sseServer)
	if err != nil {
		return nil, err
	}
	// 未启用 HTTP 配置时，跳过 HTTP 服务创建。
	if !httpConfigured {
		return nil, nil
	}

	var srv *kratosHTTP.Server
	srv, err = serverhttp.CreateHttpServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}
	serverReturned := false
	defer func() {
		if serverReturned {
			return
		}
		if stopErr := srv.Stop(context.Background()); stopErr != nil {
			log.Error("停止 Core HTTP 服务失败", "error", stopErr)
		}
	}()

	ossRootDirectory := defaultStaticRootDirectory
	// 配置了本地 OSS 根目录时，优先使用配置值覆盖默认目录。
	if cfg.GetOss() != nil && cfg.GetOss().GetRootDirectory() != "" {
		ossRootDirectory = cfg.GetOss().GetRootDirectory()
	}
	// 只有本地 OSS 才能由当前 HTTP 服务直接暴露对象目录。
	if cfg.GetOss() == nil || cfg.GetOss().GetType() == "" || cfg.GetOss().GetType() == string(oss.Local) {
		// OSS 本地对象统一通过 /data/ 暴露，业务模块不参与静态资源路由注册。
		registerDataStaticRoute(srv, ossRootDirectory)
	}
	// 先注册可回退到 index.html 的 SPA 路由，避免通用项目静态路由提前截获前端客户端路由。
	registerLocalSPARoutes(srv, defaultStaticRootDirectory)
	modules.RegisterHTTP(srv)
	if mcpServer != nil && mcpServer.Server != nil && mcpServer.InProcess {
		var mcpHandler http.Handler
		mcpHandler, err = mcpServer.Server.HTTPHandler()
		if err != nil {
			return nil, err
		}
		mcpPath := mcpserver.DefaultMCPHandlerPath
		if cfg.GetServer().GetMcp().GetPath() != "" {
			mcpPath = cfg.GetServer().GetMcp().GetPath()
		}
		srv.Handle(mcpPath, mcpHandler)
	}
	if sseServer != nil && sseServer.Server != nil && sseServer.InProcess {
		ssePath := "/events"
		if cfg.GetServer().GetSse().GetPath() != "" {
			ssePath = cfg.GetServer().GetSse().GetPath()
		}
		srv.Handle(ssePath, NewSSEHTTPHandler(sseServer.Server, sseServer.Resolver()))
	}
	// 显式启用 Swagger 时，使用同一个注册表挂载 Core 和模块文档。
	if cfg.GetServer().GetHttp().GetEnableSwagger() {
		err = openapi.RegisterHTTP(srv, openAPIRegistry, openapi.HTTPOptions{
			DocumentPath: "/api/docs/openapi",
			SwaggerPath:  "/api/docs/swagger",
			Authorizer:   newOpenAPIAuthorizer(authenticator, userToken),
		})
		if err != nil {
			return nil, err
		}
	}

	serverReturned = true
	return srv, nil
}

// validateInProcessHTTPHost 校验进程内传输是否具备 HTTP 宿主。
func validateInProcessHTTPHost(httpConfigured bool, mcpServer *mcp.Server, sseServer *sse.Server) error {
	if httpConfigured {
		return nil
	}
	if mcpServer != nil && mcpServer.InProcess {
		return errors.New("进程内 MCP 需要配置 HTTP 服务作为宿主")
	}
	if sseServer != nil && sseServer.InProcess {
		return errors.New("进程内 SSE 需要配置 HTTP 服务作为宿主")
	}
	return nil
}

// NewSSEHTTPHandler 创建带业务流解析的 SSE HTTP 处理器。
func NewSSEHTTPHandler(server *sseServer.Server, resolver sseServer.StreamIDResolver) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodOptions {
			server.ServeHTTP(writer, request)
			return
		}
		if resolver == nil {
			serveSSEHTTP(request, func(streamRequest *http.Request) {
				server.ServeHTTP(writer, streamRequest)
			})
			return
		}
		streamID, err := resolver(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		server.CreateStream(sseServer.StreamID(streamID))
		serveSSEHTTP(request, func(streamRequest *http.Request) {
			server.ServeStreamHTTP(writer, streamRequest, sseServer.StreamID(streamID))
		})
	})
}

// serveSSEHTTP 为 SSE 请求移除服务级 deadline，同时保留客户端断开带来的取消信号。
func serveSSEHTTP(request *http.Request, handler func(*http.Request)) {
	streamRequest, cleanup := sse.DetachRequestContext(request)
	defer cleanup()
	handler(streamRequest)
}

// registerLocalSPARoutes 为固定静态根目录下的前端目录注册单页应用路由。
func registerLocalSPARoutes(srv *kratosHTTP.Server, rootDirectory string) {
	entries, err := os.ReadDir(rootDirectory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		directory := filepath.Join(rootDirectory, entry.Name())
		if _, err = os.Stat(filepath.Join(directory, "index.html")); err != nil {
			continue
		}
		prefix := "/" + entry.Name()
		handler := newSPAHandler(os.DirFS(directory), prefix)
		srv.Handle(prefix, handler)
		srv.HandlePrefix(prefix+"/", handler)
	}
}

// registerDataStaticRoute 注册本地 OSS 根目录的只读访问路由。
func registerDataStaticRoute(srv *kratosHTTP.Server, rootDirectory string) {
	dataHandler := http.StripPrefix("/data/", http.FileServer(http.Dir(rootDirectory)))
	srv.HandlePrefix("/data/", dataHandler)
}

// newSPAHandler 为前端路由提供静态文件和 index.html 回退。
func newSPAHandler(webFS fs.FS, prefix string) http.Handler {
	fileHandler := http.StripPrefix(prefix, http.FileServer(http.FS(webFS)))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		relativePath := strings.TrimPrefix(request.URL.Path, prefix)
		relativePath = strings.TrimPrefix(relativePath, "/")
		if relativePath != "" {
			if _, err := fs.Stat(webFS, relativePath); err == nil {
				fileHandler.ServeHTTP(writer, request)
				return
			}
		}
		http.ServeFileFS(writer, request, webFS, "index.html")
	})
}

// newOpenAPIAuthorizer 校验 Swagger 文档请求中的 Bearer Token。
func newOpenAPIAuthorizer(authenticator engine.Authenticator, userToken *data.UserToken) func(*http.Request) bool {
	return func(request *http.Request) bool {
		if authenticator == nil || userToken == nil {
			return true
		}
		parts := strings.Fields(request.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], engine.BearerWord) {
			return false
		}
		claims, err := authenticator.AuthenticateToken(parts[1])
		if err != nil {
			return false
		}
		var userID int64
		userID, err = claims.GetInt64(data.ClaimFieldUserID)
		if err != nil {
			return false
		}
		return userID == 0 || userToken.IsExistAccessToken(userID)
	}
}
