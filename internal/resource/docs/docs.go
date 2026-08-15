package docs

import (
	"github.com/liujitcn/kratos-core/pkg/biz"
	"github.com/liujitcn/kratos-core/pkg/dto"
)

// Docs 实现 Core 的项目文档查询能力。
type Docs struct {
	registry *Registry
}

var _ biz.Docs = (*Docs)(nil)

// NewDocs 创建项目文档查询服务。
func NewDocs(registry *Registry) *Docs {
	return &Docs{registry: registry}
}

// Projects 查询 Core 保存的项目文档树。
func (d *Docs) Projects() []dto.Project {
	return d.registry.Projects()
}

// Get 按稳定 ID 查询 Core 保存的项目文档。
func (d *Docs) Get(id string) (dto.Document, bool) {
	return d.registry.Get(id)
}
