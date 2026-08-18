package dto

// OpenAPIService 描述一份 OpenAPI 文档及其 HTTP 接口归属。
type OpenAPIService struct {
	// Key 是 OpenAPI 文档稳定标识。
	Key string
	// Name 是 OpenAPI 文档展示名称。
	Name string
	// Operations 是当前文档包含的 HTTP 接口。
	Operations []OpenAPIOperation
}

// OpenAPIOperation 描述 OpenAPI 文档中的一个 HTTP 接口。
type OpenAPIOperation struct {
	// Path 是 HTTP 请求路径。
	Path string
	// Method 是 HTTP 请求方法。
	Method string
}

// OpenAPIOperationDocument 描述一个 HTTP 接口的 OpenAPI 文档。
type OpenAPIOperationDocument struct {
	// Summary 是接口摘要。
	Summary string
	// Description 是接口描述。
	Description string
	// Parameters 是请求参数。
	Parameters []*OpenAPISchema
	// RequestBody 是请求体。
	RequestBody *OpenAPISchema
	// Responses 是响应定义。
	Responses []*OpenAPIResponse
}

// OpenAPISchema 描述 OpenAPI 字段结构。
type OpenAPISchema struct {
	// Name 是字段名。
	Name string
	// Path 是字段路径。
	Path string
	// In 是参数位置。
	In string
	// Type 是字段类型。
	Type string
	// Format 是字段格式。
	Format string
	// Required 表示字段是否必填。
	Required bool
	// Description 是字段描述。
	Description string
	// Ref 是引用类型。
	Ref string
	// Enum 是枚举值。
	Enum []string
	// Children 是子字段。
	Children []*OpenAPISchema
}

// OpenAPIResponse 描述 OpenAPI 响应。
type OpenAPIResponse struct {
	// Status 是响应状态码。
	Status string
	// Description 是响应描述。
	Description string
	// Body 是响应体。
	Body *OpenAPISchema
}
