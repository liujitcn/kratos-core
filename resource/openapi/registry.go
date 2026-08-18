package openapi

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	resourceLocale "github.com/liujitcn/kratos-core/resource/locale"
	"gopkg.in/yaml.v3"
)

var (
	documentKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
)

var httpMethods = map[string]struct{}{
	"delete":  {},
	"get":     {},
	"head":    {},
	"options": {},
	"patch":   {},
	"post":    {},
	"put":     {},
	"trace":   {},
}

// Document 描述一份可独立访问的 OpenAPI 文档。
type Document struct {
	// Key 是文档的稳定唯一标识。
	Key string
	// Name 是文档的展示名称。
	Name string
	// Locale 是文档语言，空字符串表示默认文档。
	Locale string
	// Data 是未经合并的 OpenAPI 文档内容。
	Data []byte
}

// Registry 保存 OpenAPI 文档及 HTTP 操作与文档的归属关系。
type Registry struct {
	mu                sync.RWMutex
	documents         []Document
	documentsByKey    map[string]Document
	documentKeysByAPI map[string]string
}

// Register 注册 OpenAPI 文档，重复 key 仅允许名称和内容完全一致。
func (r *Registry) Register(documents ...Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	registeredDocuments := append([]Document(nil), r.documents...)
	documentsByKey := cloneDocumentMap(r.documentsByKey)
	documentKeysByAPI := cloneStringMap(r.documentKeysByAPI)
	var err error
	for _, document := range documents {
		document.Locale = resourceLocale.Normalize(document.Locale)
		err = validateDocument(document)
		if err != nil {
			return err
		}

		documentKey := newDocumentKey(document.Key, document.Locale)
		existing, exists := documentsByKey[documentKey]
		if exists {
			if existing.Name == document.Name && bytes.Equal(existing.Data, document.Data) {
				continue
			}
			return fmt.Errorf("OpenAPI 文档 key %q、语言 %q 重复", document.Key, document.Locale)
		}

		var apiKeys []string
		apiKeys, err = documentAPIKeys(document.Data)
		if err != nil {
			return fmt.Errorf("解析 OpenAPI 文档 %q 失败: %w", document.Key, err)
		}
		for _, apiKey := range apiKeys {
			apiIndexKey := newLocaleAPIKey(document.Locale, apiKey)
			ownerKey, found := documentKeysByAPI[apiIndexKey]
			if found {
				return fmt.Errorf("OpenAPI 文档 %q 与 %q 在语言 %q 包含重复接口 %s", document.Key, ownerKey, document.Locale, displayAPIKey(apiKey))
			}
			documentKeysByAPI[apiIndexKey] = documentKey
		}

		document.Data = append([]byte(nil), document.Data...)
		registeredDocuments = append(registeredDocuments, document)
		documentsByKey[documentKey] = document
	}

	r.documents = registeredDocuments
	r.documentsByKey = documentsByKey
	r.documentKeysByAPI = documentKeysByAPI
	return nil
}

// DocumentsByLocale 返回指定语言的精确 OpenAPI 文档，不执行默认语言回退。
func (r *Registry) DocumentsByLocale(locale string) []Document {
	r.mu.RLock()
	defer r.mu.RUnlock()

	locale = resourceLocale.Normalize(locale)
	documents := make([]Document, 0, len(r.documents))
	for _, document := range r.documents {
		if document.Locale != locale {
			continue
		}
		document.Data = append([]byte(nil), document.Data...)
		documents = append(documents, document)
	}
	return documents
}

// DocumentsForLocale 返回指定语言的 OpenAPI 文档，并在缺少翻译时回退默认文档。
func (r *Registry) DocumentsForLocale(locale string) []Document {
	r.mu.RLock()
	defer r.mu.RUnlock()

	locales := append(resourceLocale.Candidates(locale), "")
	selected := make(map[string]Document)
	for _, currentLocale := range locales {
		for _, document := range r.documents {
			if document.Locale == currentLocale {
				if _, exists := selected[document.Key]; !exists {
					selected[document.Key] = document
				}
			}
		}
	}
	for _, document := range r.documents {
		if document.Locale == "" {
			if _, exists := selected[document.Key]; !exists {
				selected[document.Key] = document
			}
		}
	}
	documents := make([]Document, 0, len(selected))
	for _, document := range r.documents {
		if selectedDocument, exists := selected[document.Key]; !exists || selectedDocument.Locale != document.Locale {
			continue
		}
		document.Data = append([]byte(nil), document.Data...)
		documents = append(documents, document)
	}
	return documents
}

// Locales 返回已注册的非默认 OpenAPI 语言列表。
func (r *Registry) Locales() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, document := range r.documents {
		if document.Locale != "" {
			seen[document.Locale] = struct{}{}
		}
	}
	locales := make([]string, 0, len(seen))
	for locale := range seen {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}

// DocumentByOperationForLocale 按语言查询 HTTP 操作所属 OpenAPI 文档。
func (r *Registry) DocumentByOperationForLocale(locale, path, method string) (Document, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiKey := newAPIKey(path, method)
	locales := append(resourceLocale.Candidates(locale), "")
	for _, currentLocale := range locales {
		documentKey, exists := r.documentKeysByAPI[newLocaleAPIKey(currentLocale, apiKey)]
		if !exists {
			continue
		}
		document := r.documentsByKey[documentKey]
		document.Data = append([]byte(nil), document.Data...)
		return document, true
	}
	return Document{}, false
}

// validateDocument 校验 OpenAPI 文档注册信息。
func validateDocument(document Document) error {
	if !documentKeyPattern.MatchString(document.Key) {
		return fmt.Errorf("OpenAPI 文档 key %q 只能包含字母、数字、点、下划线和连字符", document.Key)
	}
	if document.Name == "" {
		return fmt.Errorf("OpenAPI 文档 %q 的名称不能为空", document.Key)
	}
	if len(document.Data) == 0 {
		return fmt.Errorf("OpenAPI 文档 %q 的内容不能为空", document.Key)
	}
	if document.Locale != "" && resourceLocale.Normalize(document.Locale) != document.Locale {
		return fmt.Errorf("OpenAPI 文档 %q 的语言标识必须规范化", document.Key)
	}
	return nil
}

// documentAPIKeys 解析文档中的全部 HTTP 操作索引。
func documentAPIKeys(data []byte) ([]string, error) {
	var document struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	err := yaml.Unmarshal(data, &document)
	if err != nil {
		return nil, err
	}
	apiKeys := make([]string, 0)
	for path, pathItem := range document.Paths {
		for method := range pathItem {
			if _, supported := httpMethods[strings.ToLower(method)]; !supported {
				continue
			}
			apiKeys = append(apiKeys, newAPIKey(path, method))
		}
	}
	return apiKeys, nil
}

// newAPIKey 构造 HTTP 操作的内部唯一索引。
func newAPIKey(path, method string) string {
	return strings.ToUpper(method) + "\x00" + path
}

// newDocumentKey 生成项目和语言组成的 OpenAPI 文档索引键。
func newDocumentKey(key, locale string) string {
	return key + "\x00" + locale
}

// newLocaleAPIKey 生成语言和 HTTP 操作组成的 OpenAPI 索引键。
func newLocaleAPIKey(locale, apiKey string) string {
	return locale + "\x00" + apiKey
}

// displayAPIKey 将内部 HTTP 操作索引转换为可读文本。
func displayAPIKey(apiKey string) string {
	return strings.Replace(apiKey, "\x00", " ", 1)
}

// cloneDocumentMap 复制文档索引。
func cloneDocumentMap(source map[string]Document) map[string]Document {
	target := make(map[string]Document, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

// cloneStringMap 复制字符串索引。
func cloneStringMap(source map[string]string) map[string]string {
	target := make(map[string]string, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}
