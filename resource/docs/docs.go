package docs

import (
	"context"

	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/resource/docs/dto"
)

// Docs 实现 Core 的项目文档查询能力。
type Docs struct {
	registry *Registry
}

// NewDocs 创建项目文档查询服务。
func NewDocs(registry *Registry) *Docs {
	return &Docs{registry: registry}
}

// Projects 按请求语言查询 Core 保存的项目文档树。
func (d *Docs) Projects(ctx context.Context) []dto.Project {
	return d.registry.ProjectsForLocale(biz.LocaleFromContext(ctx))
}

// Get 按稳定 ID 查询 Core 保存的项目文档。
func (d *Docs) Get(ctx context.Context, id string) (dto.Document, bool) {
	return d.registry.Get(biz.LocaleFromContext(ctx), id)
}
