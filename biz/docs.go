package biz

import (
	"context"

	"github.com/liujitcn/kratos-core/internal/resource/docs"
	"github.com/liujitcn/kratos-core/internal/resource/docs/dto"
)

// Docs 实现 Core 的项目文档查询能力。
type Docs struct {
	registry *docs.Registry
}

// NewDocs 创建项目文档查询服务。
func NewDocs(registry *docs.Registry) *Docs {
	return &Docs{registry: registry}
}

// Projects 查询 Core 保存的项目文档树。
func (d *Docs) Projects(_ context.Context) []dto.Project {
	return d.registry.Projects()
}

// Get 按稳定 ID 查询 Core 保存的项目文档。
func (d *Docs) Get(ctx context.Context, id string) (dto.Document, bool) {
	return d.registry.Get(LocaleFromContext(ctx), id)
}
