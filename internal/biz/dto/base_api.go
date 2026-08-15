package dto

// OpenAPI 描述 OpenAPI 文档结构。
type OpenAPI struct {
	Paths      map[string]PathItem `yaml:"paths"`
	Tags       []TagsItem          `yaml:"tags"`
	Components Components          `yaml:"components"`
}

// PathItem 描述单个路径的请求方法。
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// PathOperation 描述单个路径下的 HTTP 操作。
type PathOperation struct {
	Method    string
	Operation *Operation
}

// TagsItem 描述 OpenAPI 标签信息。
type TagsItem struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Operation 描述单个接口操作项。
type Operation struct {
	Tags        []string            `yaml:"tags"`
	Summary     string              `yaml:"summary"`
	Description string              `yaml:"description"`
	OperationID string              `yaml:"operationId"`
	Parameters  []Parameter         `yaml:"parameters"`
	RequestBody *RequestBody        `yaml:"requestBody"`
	Responses   map[string]Response `yaml:"responses"`
}

// Components 描述 OpenAPI 组件定义。
type Components struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

// Parameter 描述 OpenAPI 请求参数。
type Parameter struct {
	Name        string `yaml:"name"`
	In          string `yaml:"in"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Schema      Schema `yaml:"schema"`
}

// RequestBody 描述 OpenAPI 请求体。
type RequestBody struct {
	Description string               `yaml:"description"`
	Required    bool                 `yaml:"required"`
	Content     map[string]MediaType `yaml:"content"`
}

// Response 描述 OpenAPI 响应。
type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content"`
}

// MediaType 描述 OpenAPI 媒体类型。
type MediaType struct {
	Schema Schema `yaml:"schema"`
}

// Schema 描述 OpenAPI Schema。
type Schema struct {
	Ref         string            `yaml:"$ref"`
	Type        string            `yaml:"type"`
	Format      string            `yaml:"format"`
	Description string            `yaml:"description"`
	Enum        []string          `yaml:"enum"`
	Required    []string          `yaml:"required"`
	Properties  map[string]Schema `yaml:"properties"`
	Items       *Schema           `yaml:"items"`
}

// Operation 按精确的 HTTP path 和 method 查询 OpenAPI 操作定义。
//
// 未找到路径、文档为空或 method 不在当前权限同步支持的 GET、POST、PUT、DELETE 范围时返回 nil。
func (api *OpenAPI) Operation(path, method string) *Operation {
	if api == nil {
		return nil
	}
	item, ok := api.Paths[path]
	if !ok {
		return nil
	}
	// HTTP 方法决定同一路径下需要返回的操作定义。
	switch method {
	case "GET":
		return item.Get
	case "POST":
		return item.Post
	case "PUT":
		return item.Put
	case "DELETE":
		return item.Delete
	default:
		return nil
	}
}
