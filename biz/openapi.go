package biz

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/internal/resource/openapi"
	"github.com/liujitcn/kratos-core/internal/resource/openapi/dto"
	"gopkg.in/yaml.v3"
)

// OpenAPI 实现 Core 的 OpenAPI 查询能力。
type OpenAPI struct {
	registry *openapi.Registry
}

// NewOpenAPI 创建 OpenAPI 查询服务。
func NewOpenAPI(registry *openapi.Registry) *OpenAPI {
	return &OpenAPI{registry: registry}
}

// Services 查询 Core 保存的 OpenAPI 服务。
func (o *OpenAPI) Services(ctx context.Context, serviceCode string) ([]dto.OpenAPIService, error) {
	documents := o.registry.DocumentsForLocale(LocaleFromContext(ctx))
	services := make([]dto.OpenAPIService, 0, len(documents))
	var err error
	for _, document := range documents {
		if serviceCode != "" && document.Key != serviceCode {
			continue
		}
		var api *openAPIDocument
		api, err = parseOpenAPIDocument(document.Data)
		if err != nil {
			return nil, errorsx.Internal("解析OpenAPI文档失败").WithCause(fmt.Errorf("解析 OpenAPI 文档 %q: %w", document.Key, err))
		}
		services = append(services, dto.OpenAPIService{
			Key:        document.Key,
			Name:       document.Name,
			Operations: openAPIDocumentOperations(api),
		})
	}
	return services, nil
}

// Service 按 HTTP 操作查询所属 OpenAPI 服务。
func (o *OpenAPI) Service(ctx context.Context, path, method string) (dto.OpenAPIService, bool) {
	document, found := o.registry.DocumentByOperationForLocale(LocaleFromContext(ctx), path, method)
	if !found {
		return dto.OpenAPIService{}, false
	}
	return dto.OpenAPIService{Key: document.Key, Name: document.Name}, true
}

// GetOperation 按 HTTP 操作查询 OpenAPI 接口文档。
func (o *OpenAPI) GetOperation(ctx context.Context, path, method string) (*dto.OpenAPIOperationDocument, error) {
	if len(o.registry.DocumentsForLocale(LocaleFromContext(ctx))) == 0 {
		return nil, errorsx.ResourceNotFound("OpenAPI文档不存在")
	}
	document, found := o.registry.DocumentByOperationForLocale(LocaleFromContext(ctx), path, method)
	if !found {
		return nil, errorsx.ResourceNotFound("OpenAPI文档不存在").WithCause(fmt.Errorf("%s %s", method, path))
	}
	var api *openAPIDocument
	var err error
	api, err = parseOpenAPIDocument(document.Data)
	if err != nil {
		return nil, errorsx.Internal("解析OpenAPI文档失败").WithCause(err)
	}
	operation := api.operation(path, method)
	if operation == nil {
		return nil, errorsx.ResourceNotFound("OpenAPI接口不存在").WithCause(fmt.Errorf("%s %s", method, path))
	}
	return &dto.OpenAPIOperationDocument{
		Summary:     operation.Summary,
		Description: operation.Description,
		Parameters:  buildOpenAPIParameters(api, operation.Parameters),
		RequestBody: buildOpenAPIRequestBody(api, operation.RequestBody),
		Responses:   buildOpenAPIResponses(api, operation.Responses),
	}, nil
}

const (
	openAPIDocumentJSONMediaType = "application/json"
	openAPIDocumentMaxDepth      = 8
)

type openAPIDocument struct {
	Paths      map[string]openAPIPathItem `yaml:"paths"`
	Components openAPIComponents          `yaml:"components"`
}

type openAPIPathItem struct {
	Get     *openAPIOperation `yaml:"get,omitempty"`
	Post    *openAPIOperation `yaml:"post,omitempty"`
	Put     *openAPIOperation `yaml:"put,omitempty"`
	Delete  *openAPIOperation `yaml:"delete,omitempty"`
	Head    *openAPIOperation `yaml:"head,omitempty"`
	Options *openAPIOperation `yaml:"options,omitempty"`
	Patch   *openAPIOperation `yaml:"patch,omitempty"`
	Trace   *openAPIOperation `yaml:"trace,omitempty"`
}

type openAPIPathOperation struct {
	method    string
	operation *openAPIOperation
}

type openAPIOperation struct {
	Summary     string                     `yaml:"summary"`
	Description string                     `yaml:"description"`
	OperationID string                     `yaml:"operationId"`
	Parameters  []openAPIParameter         `yaml:"parameters"`
	RequestBody *openAPIRequestBody        `yaml:"requestBody"`
	Responses   map[string]openAPIResponse `yaml:"responses"`
}

type openAPIComponents struct {
	Schemas map[string]openAPISchema `yaml:"schemas"`
}

type openAPIParameter struct {
	Name        string        `yaml:"name"`
	In          string        `yaml:"in"`
	Description string        `yaml:"description"`
	Required    bool          `yaml:"required"`
	Schema      openAPISchema `yaml:"schema"`
}

type openAPIRequestBody struct {
	Description string                      `yaml:"description"`
	Required    bool                        `yaml:"required"`
	Content     map[string]openAPIMediaType `yaml:"content"`
}

type openAPIResponse struct {
	Description string                      `yaml:"description"`
	Content     map[string]openAPIMediaType `yaml:"content"`
}

type openAPIMediaType struct {
	Schema openAPISchema `yaml:"schema"`
}

type openAPISchema struct {
	Ref         string                   `yaml:"$ref"`
	Type        string                   `yaml:"type"`
	Format      string                   `yaml:"format"`
	Description string                   `yaml:"description"`
	Enum        []string                 `yaml:"enum"`
	Required    []string                 `yaml:"required"`
	Properties  map[string]openAPISchema `yaml:"properties"`
	Items       *openAPISchema           `yaml:"items"`
}

// operation 按 HTTP 路径和方法查询 OpenAPI 接口定义。
func (d *openAPIDocument) operation(path, method string) *openAPIOperation {
	if d == nil {
		return nil
	}
	item, found := d.Paths[path]
	if !found {
		return nil
	}
	for _, operation := range openAPIPathOperations(item) {
		if operation.method == strings.ToUpper(method) {
			return operation.operation
		}
	}
	return nil
}

// parseOpenAPIDocument 解析查询所需的 OpenAPI YAML 或 JSON 字段。
func parseOpenAPIDocument(data []byte) (*openAPIDocument, error) {
	var document openAPIDocument
	err := yaml.Unmarshal(data, &document)
	if err != nil {
		return nil, err
	}
	return &document, nil
}

// openAPIPathOperations 返回路径下支持的全部 HTTP 操作。
func openAPIPathOperations(item openAPIPathItem) []openAPIPathOperation {
	return []openAPIPathOperation{
		{method: "GET", operation: item.Get},
		{method: "POST", operation: item.Post},
		{method: "PUT", operation: item.Put},
		{method: "DELETE", operation: item.Delete},
		{method: "HEAD", operation: item.Head},
		{method: "OPTIONS", operation: item.Options},
		{method: "PATCH", operation: item.Patch},
		{method: "TRACE", operation: item.Trace},
	}
}

// openAPIDocumentOperations 返回文档包含的 HTTP 接口并稳定排序。
func openAPIDocumentOperations(document *openAPIDocument) []dto.OpenAPIOperation {
	operations := make([]dto.OpenAPIOperation, 0)
	for path, item := range document.Paths {
		for _, operation := range openAPIPathOperations(item) {
			if operation.operation != nil {
				operations = append(operations, dto.OpenAPIOperation{Path: path, Method: operation.method})
			}
		}
	}
	sort.Slice(operations, func(left, right int) bool {
		if operations[left].Path == operations[right].Path {
			return operations[left].Method < operations[right].Method
		}
		return operations[left].Path < operations[right].Path
	})
	return operations
}

// buildOpenAPIParameters 构建请求参数文档。
func buildOpenAPIParameters(document *openAPIDocument, parameters []openAPIParameter) []*dto.OpenAPISchema {
	items := make([]*dto.OpenAPISchema, 0, len(parameters))
	for _, parameter := range parameters {
		item := buildOpenAPISchema(document, parameter.Name, parameter.Name, parameter.In, parameter.Required, parameter.Schema, 0)
		if parameter.Description != "" {
			item.Description = parameter.Description
		}
		items = append(items, item)
	}
	return items
}

// buildOpenAPISchema 展开 OpenAPI Schema 为字段树。
func buildOpenAPISchema(document *openAPIDocument, name, path, in string, required bool, schema openAPISchema, depth int) *dto.OpenAPISchema {
	schema, refName := dereferenceOpenAPISchema(document, schema)
	item := &dto.OpenAPISchema{
		Name: name, Path: path, In: in, Type: schema.Type, Format: schema.Format, Required: required,
		Description: schema.Description, Ref: refName, Enum: append([]string(nil), schema.Enum...),
	}
	if item.Type == "" {
		item.Type = inferOpenAPISchemaType(schema)
	}
	if depth >= openAPIDocumentMaxDepth {
		return item
	}
	if schema.Items != nil {
		item.Children = []*dto.OpenAPISchema{buildOpenAPISchema(document, name+"[]", path+"[]", in, false, *schema.Items, depth+1)}
	}
	if len(schema.Properties) > 0 {
		requiredFields := make(map[string]bool, len(schema.Required))
		for _, field := range schema.Required {
			requiredFields[field] = true
		}
		fieldNames := make([]string, 0, len(schema.Properties))
		for fieldName := range schema.Properties {
			fieldNames = append(fieldNames, fieldName)
		}
		sort.Strings(fieldNames)
		item.Children = make([]*dto.OpenAPISchema, 0, len(fieldNames))
		for _, fieldName := range fieldNames {
			fieldPath := fieldName
			if path != "" {
				fieldPath = path + "." + fieldName
			}
			item.Children = append(item.Children, buildOpenAPISchema(document, fieldName, fieldPath, in, requiredFields[fieldName], schema.Properties[fieldName], depth+1))
		}
	}
	return item
}

// dereferenceOpenAPISchema 解析本地组件引用。
func dereferenceOpenAPISchema(document *openAPIDocument, schema openAPISchema) (openAPISchema, string) {
	refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	if refName == "" || document == nil {
		return schema, refName
	}
	refSchema, found := document.Components.Schemas[refName]
	if !found {
		return schema, refName
	}
	if refSchema.Description == "" {
		refSchema.Description = schema.Description
	}
	return refSchema, refName
}

// inferOpenAPISchemaType 推断缺省 Schema 类型。
func inferOpenAPISchemaType(schema openAPISchema) string {
	if len(schema.Properties) > 0 || schema.Ref != "" {
		return "object"
	}
	if schema.Items != nil {
		return "array"
	}
	return "string"
}

// buildOpenAPIRequestBody 构建请求体文档。
func buildOpenAPIRequestBody(document *openAPIDocument, requestBody *openAPIRequestBody) *dto.OpenAPISchema {
	if requestBody == nil {
		return nil
	}
	schema := selectOpenAPIContentSchema(requestBody.Content)
	if schema == nil {
		return nil
	}
	item := buildOpenAPISchema(document, "body", "body", "body", requestBody.Required, *schema, 0)
	if requestBody.Description != "" {
		item.Description = requestBody.Description
	}
	return item
}

// selectOpenAPIContentSchema 选择可展示的 JSON Schema。
func selectOpenAPIContentSchema(content map[string]openAPIMediaType) *openAPISchema {
	if len(content) == 0 {
		return nil
	}
	media, found := content[openAPIDocumentJSONMediaType]
	if found {
		return &media.Schema
	}
	for _, media = range content {
		return &media.Schema
	}
	return nil
}

// buildOpenAPIResponses 构建响应文档。
func buildOpenAPIResponses(document *openAPIDocument, responses map[string]openAPIResponse) []*dto.OpenAPIResponse {
	statuses := make([]string, 0, len(responses))
	for status := range responses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	items := make([]*dto.OpenAPIResponse, 0, len(statuses))
	for _, status := range statuses {
		response := responses[status]
		item := &dto.OpenAPIResponse{Status: status, Description: response.Description}
		schema := selectOpenAPIContentSchema(response.Content)
		if schema != nil {
			item.Body = buildOpenAPISchema(document, "body", "body", "body", false, *schema, 0)
		}
		items = append(items, item)
	}
	return items
}
