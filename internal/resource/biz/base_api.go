package biz

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/data/models"
	"github.com/liujitcn/kratos-core/internal/resource/biz/dto"
	kitutils "github.com/liujitcn/kratos-kit/utils"
	"gopkg.in/yaml.v3"
)

// BaseAPICase 接口业务实例。
type BaseAPICase struct {
	*data.BaseAPIRepository
	*data.BaseAPII18nRepository
}

// NewBaseAPICase 创建接口业务实例。
func NewBaseAPICase(baseAPIRepo *data.BaseAPIRepository, baseAPII18nRepo *data.BaseAPII18nRepository) *BaseAPICase {
	return &BaseAPICase{BaseAPIRepository: baseAPIRepo, BaseAPII18nRepository: baseAPII18nRepo}
}

// OpenAPIDataToBaseAPI 将 OpenAPI 文档转换为待持久化的接口模型。
//
// 此方法只负责内存转换，不访问数据库。转换分为两阶段：先按“终端 + 服务名”
// 推断完整 protobuf 包名，再生成每个 HTTP operation，避免 shop 与 system 的同名服务互相覆盖。
func (c *BaseAPICase) OpenAPIDataToBaseAPI(openAPIData []byte) ([]*models.BaseAPI, error) {
	api, err := ParseOpenAPI(openAPIData)
	if err != nil {
		return nil, err
	}

	// OpenAPI operationId 不含 protobuf 包名，必须先从请求和响应 schema 建立服务包名索引。
	servicePackages := buildServicePackageMap(api.Paths)
	// tag 只提供展示描述，索引键仍使用完整服务名以消除同名 tag 的歧义。
	tagsMap := buildTagsMap(api.Tags, servicePackages)

	baseAPIList := make([]*models.BaseAPI, 0)
	for path, item := range api.Paths {
		for _, operation := range pathOperations(item) {
			var baseAPI *models.BaseAPI
			baseAPI, err = parseOperation(path, operation.Method, operation.Operation, tagsMap, servicePackages)
			if err != nil {
				return nil, err
			}
			// 当前路径存在对应 HTTP 操作时，写入接口权限与 MCP 元数据。
			if baseAPI != nil {
				baseAPIList = append(baseAPIList, baseAPI)
			}
		}
	}

	// 数据库存储和后续 Casbin 重建都以 operation 为标识，排序保证每次生成顺序稳定。
	sort.Slice(baseAPIList, func(i, j int) bool {
		return baseAPIList[i].Operation < baseAPIList[j].Operation
	})
	return baseAPIList, nil
}

// OpenAPIDataToBaseAPII18n 将指定语言的 OpenAPI 文档转换为 API 国际化记录。
func (c *BaseAPICase) OpenAPIDataToBaseAPII18n(openAPIData []byte, locale string) ([]*models.BaseAPII18n, error) {
	if locale == "" {
		return nil, nil
	}
	items, err := c.OpenAPIDataToBaseAPI(openAPIData)
	if err != nil {
		return nil, err
	}
	result := make([]*models.BaseAPII18n, 0, len(items))
	for _, item := range items {
		if item == nil || item.Operation == "" {
			continue
		}
		result = append(result, &models.BaseAPII18n{
			Operation:   item.Operation,
			Locale:      locale,
			ToolPrompts: item.ToolPrompts,
			ServiceDesc: item.ServiceDesc,
			Desc:        item.Desc,
		})
	}
	return result, nil
}

// ParseOpenAPI 解析 OpenAPI YAML 文档。
//
// 该方法只做 YAML 解码并完整保留 paths、tags 与 schemas，业务层可同时复用同一份文档
// 生成权限数据和展示接口文档。
func ParseOpenAPI(openAPIData []byte) (*dto.OpenAPI, error) {
	var api dto.OpenAPI
	err := yaml.Unmarshal(openAPIData, &api)
	if err != nil {
		return nil, err
	}
	return &api, nil
}

// buildServicePackageMap 从全部 OpenAPI operation 的 schema 引用推断服务所属 protobuf 包。
//
// 返回键为 servicePackageKey(path, operationId服务名)，值为唯一候选包。一个服务的多个
// operation 可以共同提供 schema 线索；只要候选包不唯一就不返回该服务，宁可跳过错误记录，
// 也不能把权限 operation 绑定到另一个模块。
func buildServicePackageMap(paths map[string]dto.PathItem) map[string]string {
	candidatesByService := make(map[string]map[string]struct{})
	for path, item := range paths {
		for _, pathOperation := range pathOperations(item) {
			// 当前 HTTP 方法未声明 operation 时，没有可供推断的 operationId 或 schema。
			if pathOperation.Operation == nil {
				continue
			}
			serviceTag, _ := parseOperationID(pathOperation.Operation.OperationID)
			serviceKey := servicePackageKey(path, serviceTag)
			// operationId 或主版本路径无效时，不参与服务包名推断。
			if serviceKey == "" {
				continue
			}
			// 同一个服务的每个 HTTP operation 都为所属包提供候选，集合自动消除重复引用。
			for packageName := range operationProtoPackages(path, pathOperation.Operation) {
				if candidatesByService[serviceKey] == nil {
					candidatesByService[serviceKey] = make(map[string]struct{})
				}
				candidatesByService[serviceKey][packageName] = struct{}{}
			}
		}
	}

	servicePackages := make(map[string]string, len(candidatesByService))
	for serviceKey, candidates := range candidatesByService {
		// 同一终端服务只允许归属一个包，避免共享 DTO 导致权限 operation 绑定到错误模块。
		if len(candidates) != 1 {
			continue
		}
		for packageName := range candidates {
			servicePackages[serviceKey] = packageName
		}
	}
	return servicePackages
}

// pathOperations 返回参与接口权限同步的 HTTP 操作。
//
// 当前生成链路只把 GET、POST、PUT、DELETE 写入 base_api；未列出的 HEAD、OPTIONS、PATCH
// 不会生成权限记录，新增受支持方法时必须同时扩展此处和 OpenAPI.Operation。
func pathOperations(item dto.PathItem) []dto.PathOperation {
	return []dto.PathOperation{
		{Method: "GET", Operation: item.Get},
		{Method: "POST", Operation: item.Post},
		{Method: "PUT", Operation: item.Put},
		{Method: "DELETE", Operation: item.Delete},
	}
}

// parseOperationID 从 OpenAPI operationId 拆出 RPC 服务名与方法名。
//
// protoc-gen-openapi 按 Service_Method 生成 operationId；两段分别用于构造 service_name
// 与 Kratos 运行时 operation。格式不合法的 operation 不写入权限表，避免生成无法匹配的策略。
func parseOperationID(operationID string) (string, string) {
	parts := strings.SplitN(operationID, "_", 2)
	// operationId 必须满足 Service_Method 格式才能还原 Kratos 运行时 operation。
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// servicePackageKey 生成“终端 + 服务名”唯一索引键。
//
// 同一个 Service 可以同时出现在 shop.admin 和 system.admin，因此不能仅用服务名定位 protobuf 包。
func servicePackageKey(path, serviceTag string) string {
	terminal := openAPITerminal(path)
	// 非主版本路径或缺少服务名时，没有可用于包名推断的稳定索引。
	if terminal == "" || serviceTag == "" {
		return ""
	}
	return terminal + "\x00" + serviceTag
}

// openAPITerminal 获取 /api/v1/{terminal}/... 路径中的终端段。
//
// 仅主版本路径参与权限同步；additional_bindings 产生的兼容旧路径不应重复生成同一个 RPC 权限。
func openAPITerminal(path string) string {
	paths := strings.Split(strings.Trim(path, "/"), "/")
	// 路径必须同时包含 api、版本和终端三个固定段。
	if len(paths) < 3 || paths[0] != "api" || paths[1] != "v1" {
		return ""
	}
	return paths[2]
}

// operationProtoPackages 收集一个 HTTP operation 中属于当前终端的 protobuf 包。
//
// 请求参数、请求体和响应可能引用不同的 schema；统一收集后由 buildServicePackageMap
// 判断服务是否只有一个候选包，防止共享 DTO 或错误引用污染权限 operation。
func operationProtoPackages(path string, operation *dto.Operation) map[string]struct{} {
	packages := make(map[string]struct{})
	// 缺少 operation 时没有 schema 可以用于包名推断。
	if operation == nil {
		return packages
	}
	for _, parameter := range operation.Parameters {
		collectSchemaProtoPackages(path, parameter.Schema, packages)
	}
	// 请求体的每种媒体类型都可能定义独立 schema，需要全部检查。
	if operation.RequestBody != nil {
		for _, mediaType := range operation.RequestBody.Content {
			collectSchemaProtoPackages(path, mediaType.Schema, packages)
		}
	}
	// 响应按状态码和媒体类型分组，任一分支都可提供所属 protobuf 包线索。
	for _, response := range operation.Responses {
		for _, mediaType := range response.Content {
			collectSchemaProtoPackages(path, mediaType.Schema, packages)
		}
	}
	return packages
}

// collectSchemaProtoPackages 递归收集 schema 及其内嵌字段中属于当前终端的 protobuf 包。
//
// 共享 schema（如 common.v1）会出现在多种业务 operation 中，只有包名终端与 HTTP 路径终端
// 一致时才成为候选，避免将共享类型误认为服务所属包。
func collectSchemaProtoPackages(path string, schema dto.Schema, packages map[string]struct{}) {
	packageName := protoPackageFromSchemaRef(schema.Ref)
	// schema 引用的包与当前 HTTP 终端一致时，才能作为服务所属包名候选。
	if packageName != "" && protoPackageTerminal(packageName) == openAPITerminal(path) {
		packages[packageName] = struct{}{}
	}
	for _, property := range schema.Properties {
		collectSchemaProtoPackages(path, property, packages)
	}
	// 数组元素也可能直接引用另一个 protobuf message，不能只检查数组外层 schema。
	if schema.Items != nil {
		collectSchemaProtoPackages(path, *schema.Items, packages)
	}
}

// protoPackageFromSchemaRef 从 OpenAPI schema 引用提取完整 protobuf 包名。
//
// 在 fq_schema_naming=true 时，引用形如 #/components/schemas/shop.admin.v1.GoodsInfo。
// 项目约定 protobuf 包以版本段结束，因此第一个 vN 分段及其前缀就是包名，后续部分是 message 名。
func protoPackageFromSchemaRef(ref string) string {
	const schemaRefPrefix = "#/components/schemas/"
	schemaName := strings.TrimPrefix(ref, schemaRefPrefix)
	// 非组件 schema 引用不能可靠还原 protobuf 包名。
	if schemaName == ref {
		return ""
	}

	parts := strings.Split(schemaName, ".")
	for index, part := range parts {
		// 版本段后必须仍有 message 名，避免把不完整的 schema 名误判为包名。
		if isProtoPackageVersion(part) && index > 0 && index < len(parts)-1 {
			return strings.Join(parts[:index+1], ".")
		}
	}
	return ""
}

// isProtoPackageVersion 判断包名分段是否为以数字开头的版本段。
//
// 除 v1 外，也兼容 v2、v1beta1 等版本命名；只要 v 后首字符为数字，即可作为包名边界。
func isProtoPackageVersion(value string) bool {
	return len(value) > 1 && value[0] == 'v' && value[1] >= '0' && value[1] <= '9'
}

// protoPackageTerminal 返回 protobuf 包版本段前的终端名称。
//
// 例如 shop.admin.v1 返回 admin，base.v1 返回 base；该规则不依赖 shop、system 等模块列表。
func protoPackageTerminal(packageName string) string {
	parts := strings.Split(packageName, ".")
	// 包名必须以版本段结尾，版本段前一段即为 HTTP 终端。
	if len(parts) < 2 || !isProtoPackageVersion(parts[len(parts)-1]) {
		return ""
	}
	return parts[len(parts)-2]
}

// buildTagsMap 构建“完整服务名 -> 服务描述”的索引。
//
// OpenAPI tag 名可能在 admin 与 app 重复，因此先由 tag 描述确定终端，再结合服务包索引生成完整键。
func buildTagsMap(tags []dto.TagsItem, servicePackages map[string]string) map[string]string {
	tagsMap := make(map[string]string, len(tags))
	for _, item := range tags {
		terminal := terminalByTagDescription(item.Description)
		packageName := servicePackages[terminal+"\x00"+item.Name]
		// 标签描述可确定终端和服务所属包时，才写入完整服务名索引。
		if packageName != "" {
			tagsMap[fmt.Sprintf("%s.%s", packageName, item.Name)] = item.Description
		}
	}
	return tagsMap
}

// terminalByTagDescription 根据生成器写入的 tag 描述前缀识别 HTTP 终端。
//
// 该值仅用于查找展示描述，不参与 service_name 或 operation 的生成。
func terminalByTagDescription(description string) string {
	// Admin 标签描述对应管理后台服务。
	if strings.HasPrefix(description, "Admin") {
		return "admin"
	}
	// App 标签描述对应商城端服务。
	if strings.HasPrefix(description, "App") {
		return "app"
	}
	// Base 标签描述对应公共基础服务。
	if strings.HasPrefix(description, "Base") {
		return "base"
	}
	return ""
}

// parseOperation 将一个 HTTP operation 转换为 base_api 记录。
//
// service_name 固定为 {protobuf包}.{operationId服务名}，operation 固定为
// /{protobuf包}.{operationId服务名}/{operationId方法名}，两者与 Kratos 运行时鉴权值保持一致。
func parseOperation(path, method string, op *dto.Operation, tagsMap map[string]string, servicePackages map[string]string) (*models.BaseAPI, error) {
	// 操作项为空时，当前请求方法无需生成接口权限数据。
	if op == nil {
		return nil, nil
	}

	serviceTag, methodName := parseOperationID(op.OperationID)
	// operationId 缺少服务或方法时无法还原运行时 operation，直接跳过异常数据。
	if serviceTag == "" || methodName == "" {
		return nil, nil
	}
	packageName := servicePackages[servicePackageKey(path, serviceTag)]
	// 未能从当前终端和服务名确定唯一 protobuf 包时，跳过异常数据，避免写入错误权限。
	if packageName == "" {
		return nil, nil
	}

	serviceName := fmt.Sprintf("%s.%s", packageName, serviceTag)
	operation := fmt.Sprintf("/%s/%s", serviceName, methodName)
	serviceDesc := tagsMap[serviceName]
	// tag 只作为服务描述回退来源，不能替换 serviceName，否则会和 operation 的服务段不一致。
	if len(op.Tags) > 0 {
		tagServiceName := fmt.Sprintf("%s.%s", packageName, op.Tags[0])
		if value, ok := tagsMap[tagServiceName]; ok {
			serviceDesc = value
		}
	}

	baseAPI := &models.BaseAPI{
		ToolName:    kitutils.ToolNameFromRPCPath(operation),
		ServiceName: serviceName,
		ServiceDesc: serviceDesc,
		Desc:        operationDescription(op),
		Operation:   operation,
		Method:      method,
		Path:        path,
		McpStatus:   _const.STATUS_STATUS_ENABLE,
		AgentStatus: _const.STATUS_STATUS_ENABLE,
	}
	baseAPI.ToolPrompts = encodeToolPrompts(defaultToolPrompts(baseAPI.ToolName, baseAPI.ServiceName, baseAPI.ServiceDesc, baseAPI.Desc, baseAPI.Operation, baseAPI.Method, baseAPI.Path))
	return baseAPI, nil
}

// operationDescription 获取接口面向管理端和工具目录的说明文本。
// description 用于完整说明，缺失时才使用简短的 summary。
func operationDescription(op *dto.Operation) string {
	if op.Description != "" {
		return op.Description
	}
	return op.Summary
}

// defaultToolPrompts 根据接口元数据生成 MCP 与 Agent 的默认检索提示词。
//
// 每条提示词只表达一个稳定维度：服务/接口业务语义、终端语义、工具名、RPC operation 或 HTTP
// 路由。最后统一去重，既便于自然语言命中，也让管理端能看出一条接口对应的实际调用标识。
func defaultToolPrompts(toolName, serviceName, serviceDesc, desc, operation, method, path string) []string {
	values := make([]string, 0, 6)
	// 服务描述与接口描述同时存在时，保留组合提示，增强完整语义命中。
	if serviceDesc != "" && desc != "" {
		values = append(values, serviceDesc+"："+desc)
	}
	if desc != "" {
		values = append(values, desc)
	}
	if serviceDesc != "" && desc == "" {
		values = append(values, serviceDesc)
	}
	terminal := terminalToolPrompt(serviceName)
	if terminal != "" && desc != "" {
		values = append(values, terminal+"："+desc)
	}
	if toolName != "" {
		values = append(values, "工具名："+toolName)
	}
	if operation != "" {
		values = append(values, "RPC："+operation)
	}
	if method != "" || path != "" {
		values = append(values, strings.TrimSpace("HTTP："+method+" "+path))
	}
	return uniqueToolPrompts(values)
}

// terminalToolPrompt 根据完整服务名生成面向用户的终端提示词。
//
// 服务名中的版本段前一段就是终端，例如 shop.admin.v1.GoodsService 的 admin，
// 这个解析与包名推断相同，不依赖具体模块名称。
func terminalToolPrompt(serviceName string) string {
	parts := strings.Split(serviceName, ".")
	for index, part := range parts {
		// 服务名中的版本段前一段为终端，兼容新增模块和版本。
		if !isProtoPackageVersion(part) || index == 0 || index == len(parts)-1 {
			continue
		}
		// 根据 HTTP 终端生成面向用户的提示词。
		switch parts[index-1] {
		case "admin":
			return "管理后台"
		case "app":
			return "商城移动端"
		case "base":
			return "公共基础"
		}
	}
	return ""
}

// uniqueToolPrompts 按首次出现顺序去除空提示词和重复提示词。
//
// 保持顺序可使默认提示词在每次 OpenAPI 同步后稳定，避免无意义的数据库内容变化。
func uniqueToolPrompts(prompts []string) []string {
	values := make([]string, 0, len(prompts))
	seen := make(map[string]struct{}, len(prompts))
	for _, item := range prompts {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		values = append(values, item)
	}
	return values
}

// encodeToolPrompts 将提示词切片编码为 base_api.tool_prompts 的 JSON 数组。
//
// JSON 编码理论上不应失败；发生异常时返回空数组，保证同步流程仍能写入有效的字段格式。
func encodeToolPrompts(prompts []string) string {
	raw, err := json.Marshal(prompts)
	if err != nil {
		return "[]"
	}
	return string(raw)
}
