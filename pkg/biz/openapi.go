package biz

import "github.com/liujitcn/kratos-core/pkg/dto"

// OpenAPI 定义 Core 提供给宿主业务模块的 OpenAPI 查询能力。
type OpenAPI interface {
	// Services 查询 Core 保存的 OpenAPI 服务。
	Services(string) ([]dto.OpenAPIService, error)
	// Service 按 HTTP 操作查询所属 OpenAPI 服务。
	Service(string, string) (dto.OpenAPIService, bool)
	// GetOperation 按 HTTP 操作查询 OpenAPI 接口文档。
	GetOperation(string, string) (*dto.OpenAPIOperationDocument, error)
}
