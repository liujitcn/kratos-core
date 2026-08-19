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

	"github.com/liujitcn/kratos-core/resource/docs/dto"
	resourceLocale "github.com/liujitcn/kratos-core/resource/locale"
)

const (
	// maxDocumentContentBytes 限制单篇文档内容的最大字节数，避免异常资源占用过多内存。
	maxDocumentContentBytes = 2 << 20
	// maxDocumentNameBytes 限制本地化文档显示名的最大字节数。
	maxDocumentNameBytes = 512
	// maxDirectoryNameBytes 限制本地化目录显示名的最大字节数。
	maxDirectoryNameBytes = 512
)

var projectKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// catalog 是单个语言 JSON 文件中的项目文档目录结构。
type catalog struct {
	// Documents 保存根目录下直接声明的文档。
	Documents []dto.Document `json:"documents"`
	// Directories 保存根目录下的目录节点。
	Directories []directory `json:"directories"`
}

// directory 表示单个语言 JSON 文件中的递归目录节点。
type directory struct {
	// Name 是当前语言下的目录显示名。
	Name string `json:"name"`
	// Path 是跨语言保持不变的稳定目录路径。
	Path string `json:"path"`
	// Documents 保存当前目录直接声明的文档。
	Documents []dto.Document `json:"documents"`
	// Directories 保存当前目录的子目录。
	Directories []directory `json:"directories"`
}

// catalogSnapshot 保存单个项目、单个语言文件解析后的文档和目录显示名。
type catalogSnapshot struct {
	documents      []dto.Document
	directoryNames map[string]string
}

// registryCatalog 保存一个语言组内所有项目的文档索引。
type registryCatalog struct {
	documents      []dto.Document
	documentsByID  map[string]dto.Document
	directoryNames map[string]string
}

// treeProject 是 Projects 构建过程中使用的项目树节点。
type treeProject struct {
	key         string
	name        string
	documents   []dto.DocumentListItem
	directories map[string]*treeDirectory
}

// treeDirectory 是 Projects 构建过程中使用的目录树节点。
type treeDirectory struct {
	name        string
	path        string
	documents   []dto.DocumentListItem
	directories map[string]*treeDirectory
}

// Registry 按语言组保存宿主和模块项目文档，并提供并发安全的查询与追加注册能力。
type Registry struct {
	mu       sync.RWMutex
	catalogs map[string]*registryCatalog
}

// NewProjectRegistry 从默认语言 docs.json 创建单个项目的文档注册表。
func NewProjectRegistry(data []byte, projectKey, projectName string) (*Registry, error) {
	snapshot, err := parseCatalog(data, projectKey, projectName)
	if err != nil {
		return nil, err
	}
	registry := &Registry{}
	err = registry.registerCatalog("", snapshot)
	if err != nil {
		return nil, fmt.Errorf("注册项目文档目录失败: %w", err)
	}
	return registry, nil
}

// Register 将文档注册到默认语言组，并校验项目名称一致性以及项目内路径唯一性。
func (r *Registry) Register(documents ...dto.Document) error {
	return r.registerCatalog("", catalogSnapshot{documents: documents})
}

// registerCatalog 将一个项目语言快照原子追加到对应语言组。
func (r *Registry) registerCatalog(locale string, snapshot catalogSnapshot) error {
	if r == nil {
		return fmt.Errorf("项目文档注册表不能为空")
	}
	locale = resourceLocale.Normalize(locale)
	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.catalogs[locale]
	registered := make([]dto.Document, 0, len(snapshot.documents))
	byID := make(map[string]dto.Document)
	directoryNames := make(map[string]string)
	if current != nil {
		registered = append(registered, current.documents...)
		for id, document := range current.documentsByID {
			byID[id] = document
		}
		for key, name := range current.directoryNames {
			directoryNames[key] = name
		}
	}
	projectNames := make(map[string]string)
	paths := make(map[string]struct{})
	for _, document := range registered {
		projectNames[document.ProjectKey] = document.ProjectName
		paths[document.ProjectKey+"\x00"+document.Path] = struct{}{}
	}
	var err error
	for _, document := range snapshot.documents {
		if document.Name == "" {
			document.Name = path.Base(document.Path)
		}
		err = validateDocument(document)
		if err != nil {
			return err
		}
		if name, exists := projectNames[document.ProjectKey]; exists && name != document.ProjectName {
			return fmt.Errorf("项目文档标识 %q 对应多个项目名称", document.ProjectKey)
		}
		key := document.ProjectKey + "\x00" + document.Path
		if _, exists := paths[key]; exists {
			return fmt.Errorf("项目文档路径重复: %s/%s", document.ProjectKey, document.Path)
		}
		document.ID = newDocumentID(document.ProjectKey, document.Path)
		projectNames[document.ProjectKey] = document.ProjectName
		paths[key] = struct{}{}
		registered = append(registered, document)
		byID[document.ID] = document
	}
	for key, name := range snapshot.directoryNames {
		if existing, exists := directoryNames[key]; exists && existing != name {
			return fmt.Errorf("项目文档目录显示名重复: %s", strings.ReplaceAll(key, "\x00", "/"))
		}
		directoryNames[key] = name
	}
	if r.catalogs == nil {
		r.catalogs = make(map[string]*registryCatalog)
	}
	r.catalogs[locale] = &registryCatalog{
		documents:      registered,
		documentsByID:  byID,
		directoryNames: directoryNames,
	}
	return nil
}

// Documents 返回默认语言组中按注册顺序排列的全部文档快照。
func (r *Registry) Documents() []dto.Document {
	return r.documentsForLocale("")
}

// documentsForLocale 返回指定语言覆盖默认语言后的完整文档快照。
func (r *Registry) documentsForLocale(locale string) []dto.Document {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.documentsForLocaleLocked(locale)
}

// documentsForLocaleLocked 在读锁保护下按语言候选覆盖默认文档。
func (r *Registry) documentsForLocaleLocked(locale string) []dto.Document {
	defaults := r.catalogs[""]
	if defaults == nil {
		return nil
	}
	candidates := resourceLocale.Candidates(locale)
	result := make([]dto.Document, 0, len(defaults.documents))
	for _, document := range defaults.documents {
		for _, candidate := range candidates {
			localized := r.catalogs[candidate]
			if localized == nil {
				continue
			}
			if value, exists := localized.documentsByID[document.ID]; exists {
				document = value
				break
			}
		}
		result = append(result, document)
	}
	return result
}

// Projects 将默认语言文档摘要组织为按项目和路径排序的目录树。
func (r *Registry) Projects() []dto.Project {
	return r.ProjectsForLocale("")
}

// ProjectsForLocale 将请求语言下的文档摘要组织为本地化目录树。
func (r *Registry) ProjectsForLocale(locale string) []dto.Project {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	documents := r.documentsForLocaleLocked(locale)
	directoryNames := r.directoryNamesForLocaleLocked(locale)
	return buildProjects(documents, directoryNames)
}

// directoryNamesForLocaleLocked 在读锁保护下选择请求语言对应的目录显示名。
func (r *Registry) directoryNamesForLocaleLocked(locale string) map[string]string {
	defaults := r.catalogs[""]
	if defaults == nil {
		return nil
	}
	candidates := resourceLocale.Candidates(locale)
	result := make(map[string]string, len(defaults.directoryNames))
	for key, name := range defaults.directoryNames {
		for _, candidate := range candidates {
			localized := r.catalogs[candidate]
			if localized == nil {
				continue
			}
			if value, exists := localized.directoryNames[key]; exists {
				name = value
				break
			}
		}
		result[key] = name
	}
	return result
}

// Get 按请求语言和稳定 ID 查询文档，缺少语言版本时回退默认正文。
func (r *Registry) Get(locale, id string) (dto.Document, bool) {
	if r == nil {
		return dto.Document{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, candidate := range resourceLocale.Candidates(locale) {
		localized := r.catalogs[candidate]
		if localized == nil {
			continue
		}
		if document, exists := localized.documentsByID[id]; exists {
			return document, true
		}
	}
	defaults := r.catalogs[""]
	if defaults == nil {
		return dto.Document{}, false
	}
	document, exists := defaults.documentsByID[id]
	return document, exists
}

// parseCatalog 解析并校验单个项目、单个语言的文档目录快照。
func parseCatalog(data []byte, projectKey, projectName string) (catalogSnapshot, error) {
	if len(data) == 0 {
		data = []byte(`{}`)
	}
	var value catalog
	err := json.Unmarshal(data, &value)
	if err != nil {
		return catalogSnapshot{}, fmt.Errorf("解析项目文档目录失败: %w", err)
	}
	snapshot := catalogSnapshot{directoryNames: make(map[string]string)}
	for _, document := range value.Documents {
		document.ProjectKey = projectKey
		document.ProjectName = projectName
		if parentDocumentPath(document.Path) != "" {
			return catalogSnapshot{}, fmt.Errorf("项目根目录包含非根文档: %s", document.Path)
		}
		snapshot.documents = append(snapshot.documents, document)
	}
	for _, item := range value.Directories {
		err = appendDirectorySnapshot(&snapshot, item, "", projectKey, projectName)
		if err != nil {
			return catalogSnapshot{}, err
		}
	}
	return snapshot, nil
}

// appendDirectorySnapshot 校验目录层级并深度优先追加文档和本地化目录名。
func appendDirectorySnapshot(snapshot *catalogSnapshot, item directory, parentPath, projectKey, projectName string) error {
	normalizedPath := path.Clean(strings.ReplaceAll(item.Path, "\\", "/"))
	if item.Path == "" || normalizedPath == "." || normalizedPath == ".." || path.IsAbs(normalizedPath) || strings.HasPrefix(normalizedPath, "../") {
		return fmt.Errorf("项目文档目录路径无效: %q", item.Path)
	}
	if item.Path != normalizedPath || parentDocumentPath(item.Path) != parentPath {
		return fmt.Errorf("项目文档目录层级无效: %q", item.Path)
	}
	if item.Name == "" || !utf8.ValidString(item.Name) || len(item.Name) > maxDirectoryNameBytes {
		return fmt.Errorf("项目文档目录显示名无效: %s", item.Path)
	}
	key := projectKey + "\x00" + item.Path
	if _, exists := snapshot.directoryNames[key]; exists {
		return fmt.Errorf("项目文档目录路径重复: %s", item.Path)
	}
	snapshot.directoryNames[key] = item.Name
	for _, document := range item.Documents {
		document.ProjectKey = projectKey
		document.ProjectName = projectName
		if parentDocumentPath(document.Path) != item.Path {
			return fmt.Errorf("项目文档所在目录与路径不一致: %s", document.Path)
		}
		snapshot.documents = append(snapshot.documents, document)
	}
	var err error
	for _, child := range item.Directories {
		err = appendDirectorySnapshot(snapshot, child, item.Path, projectKey, projectName)
		if err != nil {
			return err
		}
	}
	return nil
}

// validateLocalizedCatalog 校验语言目录与默认目录使用完全相同的稳定路径和更新时间。
func validateLocalizedCatalog(defaults, localized catalogSnapshot, locale string) error {
	defaultDocuments := make(map[string]dto.Document, len(defaults.documents))
	for _, document := range defaults.documents {
		defaultDocuments[document.Path] = document
	}
	if len(localized.documents) != len(defaultDocuments) {
		return fmt.Errorf("项目文档语言 %q 的文档数量与默认语言不一致", locale)
	}
	for _, document := range localized.documents {
		defaultDocument, exists := defaultDocuments[document.Path]
		if !exists {
			return fmt.Errorf("项目文档语言 %q 包含未知路径: %s", locale, document.Path)
		}
		if document.UpdatedAt != defaultDocument.UpdatedAt {
			return fmt.Errorf("项目文档语言 %q 的更新时间与默认语言不一致: %s", locale, document.Path)
		}
	}
	if len(localized.directoryNames) != len(defaults.directoryNames) {
		return fmt.Errorf("项目文档语言 %q 的目录数量与默认语言不一致", locale)
	}
	for key := range localized.directoryNames {
		if _, exists := defaults.directoryNames[key]; !exists {
			return fmt.Errorf("项目文档语言 %q 包含未知目录: %s", locale, strings.ReplaceAll(key, "\x00", "/"))
		}
	}
	return nil
}

// validateDocument 校验文档的项目标识、路径和内容边界。
func validateDocument(document dto.Document) error {
	if !projectKeyPattern.MatchString(document.ProjectKey) {
		return fmt.Errorf("项目文档标识 %q 无效", document.ProjectKey)
	}
	if document.ProjectName == "" {
		return fmt.Errorf("项目文档 %q 缺少项目名称", document.ProjectKey)
	}
	normalized := path.Clean(strings.ReplaceAll(document.Path, "\\", "/"))
	if document.Path == "" || normalized == "." || normalized == ".." || path.IsAbs(normalized) || strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("项目文档路径无效: %q", document.Path)
	}
	if document.Path != normalized {
		return fmt.Errorf("项目文档路径必须使用规范相对路径: %q", document.Path)
	}
	if document.Name == "" || !utf8.ValidString(document.Name) || len(document.Name) > maxDocumentNameBytes {
		return fmt.Errorf("项目文档显示名无效: %s", document.Path)
	}
	if !utf8.ValidString(document.Content) {
		return fmt.Errorf("项目文档不是有效 UTF-8: %s", document.Path)
	}
	if len(document.Content) > maxDocumentContentBytes {
		return fmt.Errorf("项目文档超过 2 MiB: %s", document.Path)
	}
	return nil
}

// buildProjects 将文档快照和本地化目录显示名转换为项目树。
func buildProjects(documents []dto.Document, directoryNames map[string]string) []dto.Project {
	projects := make(map[string]*treeProject)
	for _, document := range documents {
		project := projects[document.ProjectKey]
		if project == nil {
			project = &treeProject{key: document.ProjectKey, name: document.ProjectName, directories: make(map[string]*treeDirectory)}
			projects[document.ProjectKey] = project
		}
		item := dto.DocumentListItem{ID: document.ID, Path: document.Path, Name: document.Name, UpdatedAt: document.UpdatedAt}
		segments := strings.Split(document.Path, "/")
		if len(segments) == 1 {
			project.documents = append(project.documents, item)
			continue
		}
		parent := project.directories
		currentPath := ""
		for index, name := range segments[:len(segments)-1] {
			currentPath = path.Join(currentPath, name)
			current := parent[currentPath]
			if current == nil {
				displayName := directoryNames[document.ProjectKey+"\x00"+currentPath]
				if displayName == "" {
					displayName = name
				}
				current = &treeDirectory{name: displayName, path: currentPath, directories: make(map[string]*treeDirectory)}
				parent[currentPath] = current
			}
			if index == len(segments)-2 {
				current.documents = append(current.documents, item)
			}
			parent = current.directories
		}
	}
	result := make([]dto.Project, 0, len(projects))
	for _, project := range projects {
		result = append(result, convertProject(project))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Key < result[right].Key })
	return result
}

// parentDocumentPath 返回文档或目录稳定路径的父目录，根目录统一为空字符串。
func parentDocumentPath(value string) string {
	parent := path.Dir(value)
	if parent == "." {
		return ""
	}
	return parent
}

// newDocumentID 根据项目标识和文档路径生成稳定 ID。
func newDocumentID(projectKey, documentPath string) string {
	sum := sha256.Sum256([]byte(projectKey + "\x00" + documentPath))
	return hex.EncodeToString(sum[:16])
}

// convertProject 将内部项目树转换为对外 DTO，并复制文档摘要切片。
func convertProject(project *treeProject) dto.Project {
	sort.Slice(project.documents, func(left, right int) bool { return project.documents[left].Path < project.documents[right].Path })
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
	paths := make([]string, 0, len(directory.directories))
	for value := range directory.directories {
		paths = append(paths, value)
	}
	sort.Strings(paths)
	directories := make([]dto.Directory, 0, len(paths))
	for _, value := range paths {
		directories = append(directories, convertDirectory(directory.directories[value]))
	}
	sort.Slice(directory.documents, func(left, right int) bool { return directory.documents[left].Path < directory.documents[right].Path })
	return dto.Directory{Name: directory.name, Path: directory.path, Documents: append([]dto.DocumentListItem(nil), directory.documents...), Directories: directories}
}
