package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	resourceLocale "github.com/liujitcn/kratos-core/internal/resource/locale"
	"github.com/liujitcn/kratos-core/pkg/dto"
)

// maxDocumentContentBytes 限制单篇文档内容的最大字节数，避免异常资源占用过多内存。
const maxDocumentContentBytes = 2 << 20

// projectKeyPattern 约束项目标识只能以字母或数字开头，并继续使用字母、数字、点、下划线和连字符。
var (
	projectKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
)

// catalog 是 docs.json 的根目录结构，文档可以直接声明或嵌套在目录中。
type catalog struct {
	// Documents 保存根目录下直接声明的文档。
	Documents []dto.Document `json:"documents"`
	// Directories 保存根目录下的目录节点。
	Directories []directory `json:"directories"`
}

// directory 表示 docs.json 中可递归嵌套的目录节点。
type directory struct {
	// Documents 保存当前目录直接声明的文档。
	Documents []dto.Document `json:"documents"`
	// Directories 保存当前目录的子目录。
	Directories []directory `json:"directories"`
}

// treeProject 是 Projects 构建过程中使用的项目树节点。
type treeProject struct {
	// key 是项目唯一标识。
	key string
	// name 是项目展示名称。
	name string
	// documents 保存项目根目录下的文档摘要。
	documents []dto.DocumentListItem
	// directories 以目录完整相对路径为 key，避免同名目录在不同层级发生混淆。
	directories map[string]*treeDirectory
}

// treeDirectory 是 Projects 构建过程中使用的目录树节点。
type treeDirectory struct {
	// name 是当前目录名称，不包含父级路径。
	name string
	// path 是从项目根目录开始的规范相对路径。
	path string
	// documents 保存当前目录直接包含的文档摘要。
	documents []dto.DocumentListItem
	// directories 保存当前目录的子目录，key 为子目录的完整相对路径。
	directories map[string]*treeDirectory
}

// Registry 保存宿主和模块合并后的项目文档，并提供并发安全的查询与追加注册能力。
type Registry struct {
	// mu 保护 documents 和 documentsByID，禁止在持锁期间把内部切片或 map 暴露给调用方。
	mu sync.RWMutex
	// documents 保存已注册的完整文档，顺序与成功注册顺序一致。
	documents []dto.Document
	// documentsByID 保存文档 ID 到完整文档的索引。
	documentsByID map[string]dto.Document
}

// NewProjectRegistry 从 docs.json 创建项目文档注册表，并为所有文档注入项目标识。
func NewProjectRegistry(data []byte, projectKey, projectName string) (*Registry, error) {
	// 未配置文档资源时按空目录处理，保证调用方无需区分 nil 和空 JSON。
	if len(data) == 0 {
		data = []byte(`{}`)
	}
	var value catalog
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, fmt.Errorf("解析项目文档目录失败: %w", err)
	}
	// 将递归目录展开为顺序文档列表，后续统一走注册校验流程。
	documents := append([]dto.Document(nil), value.Documents...)
	documents = appendDirectoryDocuments(documents, value.Directories)
	// docs.json 只描述文档本身，项目归属由宿主资源统一注入。
	for index := range documents {
		documents[index].ProjectKey = projectKey
		documents[index].ProjectName = projectName
	}
	registry := &Registry{}
	err = registry.Register(documents...)
	if err != nil {
		return nil, fmt.Errorf("注册项目文档目录失败: %w", err)
	}
	return registry, nil
}

// Register 注册一批文档，并校验项目名称一致性以及项目内路径唯一性。
func (r *Registry) Register(documents ...dto.Document) error {
	if r == nil {
		return fmt.Errorf("项目文档注册表不能为空")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// 从已有文档重建冲突索引，同时复核注册表中的历史数据。
	projectNames := make(map[string]string)
	paths := make(map[string]struct{})
	for _, document := range r.documents {
		if err := validateDocument(document); err != nil {
			return err
		}
		projectNames[document.ProjectKey] = document.ProjectName
		paths[document.ProjectKey+"\x00"+document.Path] = struct{}{}
	}
	// 在副本上组装新状态，确保批量注册失败时原注册表保持不变。
	registered := append([]dto.Document(nil), r.documents...)
	byID := make(map[string]dto.Document, len(registered)+len(documents))
	// 重建 ID 索引，保证顺序存储与查询索引使用相同的 ID 生成规则。
	for _, document := range registered {
		document.ID = newDocumentID(document.ProjectKey, document.Path)
		byID[document.ID] = document
	}
	// 逐篇校验新增文档，并同步更新临时冲突索引以发现批次内重复。
	for _, document := range documents {
		if err := validateDocument(document); err != nil {
			return err
		}
		document.Locale = cloneLocalizedContents(document.Locale)
		if name, exists := projectNames[document.ProjectKey]; exists && name != document.ProjectName {
			return fmt.Errorf("项目文档标识 %q 对应多个项目名称", document.ProjectKey)
		}
		key := document.ProjectKey + "\x00" + document.Path
		if _, exists := paths[key]; exists {
			return fmt.Errorf("项目文档路径重复: %s/%s", document.ProjectKey, document.Path)
		}
		// ID 完全由项目和路径决定，覆盖调用方可能传入的不可信 ID。
		document.ID = newDocumentID(document.ProjectKey, document.Path)
		projectNames[document.ProjectKey] = document.ProjectName
		paths[key] = struct{}{}
		registered = append(registered, document)
		byID[document.ID] = document
	}
	// 所有文档校验通过后一次性提交两个内部索引。
	r.documents = registered
	r.documentsByID = byID
	return nil
}

// Documents 返回按注册顺序排列的全部文档快照。
func (r *Registry) Documents() []dto.Document {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 复制底层切片，避免调用方修改注册表内部顺序或文档值。
	result := make([]dto.Document, 0, len(r.documents))
	for _, document := range r.documents {
		document.Locale = cloneLocalizedContents(document.Locale)
		result = append(result, document)
	}
	return result
}

// Projects 将文档摘要组织为按项目和路径排序的目录树。
func (r *Registry) Projects() []dto.Project {
	projects := make(map[string]*treeProject)
	for _, document := range r.Documents() {
		// 同一项目的文档共用一个根节点，项目名称已在注册阶段保证一致。
		project := projects[document.ProjectKey]
		if project == nil {
			project = &treeProject{key: document.ProjectKey, name: document.ProjectName, directories: make(map[string]*treeDirectory)}
			projects[document.ProjectKey] = project
		}
		// 列表树仅保留查询和展示所需字段，不携带文档正文。
		item := dto.DocumentListItem{ID: document.ID, Path: document.Path, UpdatedAt: document.UpdatedAt}
		// Path 已校验为规范相对路径，可以直接按正斜杠拆分。
		segments := strings.Split(document.Path, "/")
		// 单段路径表示文档直接位于项目根目录。
		if len(segments) == 1 {
			project.documents = append(project.documents, item)
			continue
		}
		// 逐段下钻并复用已有节点，最后将文档挂到直接父目录。
		parent := project.directories
		currentPath := ""
		for index, name := range segments[:len(segments)-1] {
			currentPath = path.Join(currentPath, name)
			current := parent[currentPath]
			if current == nil {
				// 首次遇到该路径时创建对应层级的目录节点。
				current = &treeDirectory{name: name, path: currentPath, directories: make(map[string]*treeDirectory)}
				parent[currentPath] = current
			}
			if index == len(segments)-2 {
				current.documents = append(current.documents, item)
			}
			parent = current.directories
		}
	}
	// 递归转换目录节点，并在最后稳定项目的输出顺序。
	result := make([]dto.Project, 0, len(projects))
	for _, project := range projects {
		result = append(result, convertProject(project))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Key < result[right].Key })
	return result
}

// Get 按语言和文档 ID 查询文档内容，缺少翻译时回退默认正文。
func (r *Registry) Get(locale, id string) (dto.Document, bool) {
	if r == nil {
		return dto.Document{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	document, exists := r.documentsByID[id]
	if !exists {
		return dto.Document{}, false
	}
	document.Locale = cloneLocalizedContents(document.Locale)
	document.Content = localizedContent(document.Locale, locale, document.Content)
	return document, exists
}

// validateDocument 校验文档的项目标识、路径和内容边界。
func validateDocument(document dto.Document) error {
	// 项目标识会参与文档 ID 和跨模块分组，只允许使用稳定且可移植的字符。
	if !projectKeyPattern.MatchString(document.ProjectKey) {
		return fmt.Errorf("项目文档标识 %q 无效", document.ProjectKey)
	}
	if document.ProjectName == "" {
		return fmt.Errorf("项目文档 %q 缺少项目名称", document.ProjectKey)
	}
	// 先按统一分隔符清理路径，用于识别绝对路径和父目录穿越。
	normalized := path.Clean(strings.ReplaceAll(document.Path, "\\", "/"))
	if document.Path == "" || normalized == "." || normalized == ".." || path.IsAbs(normalized) || strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("项目文档路径无效: %q", document.Path)
	}
	// 只接受规范原值，避免同一路径因分隔符或冗余片段产生多个表示。
	if document.Path != normalized {
		return fmt.Errorf("项目文档路径必须使用规范相对路径: %q", document.Path)
	}
	// 文档内容直接作为文本返回，必须是合法 UTF-8 且大小受控。
	if !utf8.ValidString(document.Content) {
		return fmt.Errorf("项目文档不是有效 UTF-8: %s", document.Path)
	}
	if len(document.Content) > maxDocumentContentBytes {
		return fmt.Errorf("项目文档超过 2 MiB: %s", document.Path)
	}
	locales := make(map[string]struct{}, len(document.Locale))
	for locale, content := range document.Locale {
		normalizedLocale := resourceLocale.Normalize(locale)
		if normalizedLocale == "" {
			return fmt.Errorf("项目文档语言标识不能为空: %s", document.Path)
		}
		if _, exists := locales[normalizedLocale]; exists {
			return fmt.Errorf("项目文档语言版本重复: %s (%s)", document.Path, locale)
		}
		locales[normalizedLocale] = struct{}{}
		if !utf8.ValidString(content) {
			return fmt.Errorf("项目文档翻译不是有效 UTF-8: %s (%s)", document.Path, locale)
		}
		if len(content) > maxDocumentContentBytes {
			return fmt.Errorf("项目文档翻译超过 2 MiB: %s (%s)", document.Path, locale)
		}
	}
	return nil
}

// localizedContent 按精确语言、基础语言和默认正文顺序选择文档内容。
func localizedContent(contents map[string]string, locale, fallback string) string {
	if locale == "" || len(contents) == 0 {
		return fallback
	}
	for _, candidate := range resourceLocale.Candidates(locale) {
		for currentLocale, content := range contents {
			if resourceLocale.Normalize(currentLocale) == candidate {
				return content
			}
		}
	}
	return fallback
}

// cloneLocalizedContents 复制文档语言内容，避免调用方修改注册表内部状态。
func cloneLocalizedContents(contents map[string]string) map[string]string {
	if len(contents) == 0 {
		return nil
	}
	result := make(map[string]string, len(contents))
	for locale, content := range contents {
		result[locale] = content
	}
	return result
}

// appendDirectoryDocuments 按声明顺序深度优先展开目录中的全部文档。
func appendDirectoryDocuments(documents []dto.Document, directories []directory) []dto.Document {
	for _, item := range directories {
		// 先保留当前目录文档顺序，再递归展开它的子目录。
		documents = append(documents, item.Documents...)
		documents = appendDirectoryDocuments(documents, item.Directories)
	}
	return documents
}

// newDocumentID 根据项目标识和文档路径生成稳定 ID。
func newDocumentID(projectKey, documentPath string) string {
	// NUL 分隔字段避免拼接边界歧义，截取前 16 字节得到紧凑的 32 位十六进制 ID。
	sum := sha256.Sum256([]byte(projectKey + "\x00" + documentPath))
	return hex.EncodeToString(sum[:16])
}

// convertProject 将内部项目树转换为对外 DTO，并复制文档摘要切片。
func convertProject(project *treeProject) dto.Project {
	// 根目录文档按完整相对路径排序。
	sort.Slice(project.documents, func(left, right int) bool { return project.documents[left].Path < project.documents[right].Path })
	// map 不保证遍历顺序，先提取并排序目录路径再递归转换。
	paths := make([]string, 0, len(project.directories))
	for value := range project.directories {
		paths = append(paths, value)
	}
	sort.Strings(paths)
	directories := make([]dto.Directory, 0, len(paths))
	for _, value := range paths {
		directories = append(directories, convertDirectory(project.directories[value]))
	}
	return dto.Project{Key: project.key, Name: project.name, Documents: append([]dto.DocumentListItem(nil), project.documents...), Directories: directories}
}

// convertDirectory 递归转换目录树，并对同级目录和文档分别排序。
func convertDirectory(directory *treeDirectory) dto.Directory {
	// 子目录以完整相对路径排序，保证每次查询的树结构顺序一致。
	paths := make([]string, 0, len(directory.directories))
	for value := range directory.directories {
		paths = append(paths, value)
	}
	sort.Strings(paths)
	directories := make([]dto.Directory, 0, len(paths))
	for _, value := range paths {
		directories = append(directories, convertDirectory(directory.directories[value]))
	}
	// 当前目录文档独立排序，不受注册顺序影响。
	sort.Slice(directory.documents, func(left, right int) bool { return directory.documents[left].Path < directory.documents[right].Path })
	return dto.Directory{Name: directory.name, Path: directory.path, Documents: append([]dto.DocumentListItem(nil), directory.documents...), Directories: directories}
}
