package dto

// Document 表示一篇可查询的项目文档。
type Document struct {
	// ID 是文档稳定标识。
	ID string `json:"id"`
	// ProjectKey 是项目唯一标识。
	ProjectKey string `json:"-"`
	// ProjectName 是项目展示名称。
	ProjectName string `json:"-"`
	// Path 是项目内相对路径。
	Path string `json:"path"`
	// Name 是当前语言下的文档显示名。
	Name string `json:"name"`
	// Content 是 Markdown 文档内容。
	Content string `json:"content"`
	// UpdatedAt 是文档更新时间。
	UpdatedAt string `json:"updated_at"`
}

// DocumentListItem 表示文档树中的文档摘要。
type DocumentListItem struct {
	// ID 是文档稳定标识。
	ID string `json:"id"`
	// Path 是项目内相对路径。
	Path string `json:"path"`
	// Name 是当前语言下的文档显示名。
	Name string `json:"name"`
	// UpdatedAt 是文档更新时间。
	UpdatedAt string `json:"updated_at"`
}

// Project 表示一个项目的文档树。
type Project struct {
	// Key 是项目唯一标识。
	Key string `json:"key"`
	// Name 是项目展示名称。
	Name string `json:"name"`
	// Documents 是项目根目录下的文档。
	Documents []DocumentListItem `json:"documents"`
	// Directories 是项目根目录下的目录。
	Directories []Directory `json:"directories"`
}

// Directory 表示文档树中的目录节点。
type Directory struct {
	// Name 是目录名称。
	Name string `json:"name"`
	// Path 是项目内相对目录路径。
	Path string `json:"path"`
	// Documents 是当前目录下的文档。
	Documents []DocumentListItem `json:"documents"`
	// Directories 是当前目录下的子目录。
	Directories []Directory `json:"directories"`
}
