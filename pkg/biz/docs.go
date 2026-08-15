package biz

import "github.com/liujitcn/kratos-core/pkg/dto"

// Docs 定义 Core 提供给宿主业务模块的项目文档查询能力。
type Docs interface {
	// Projects 查询 Core 保存的项目文档树。
	Projects() []dto.Project
	// Get 按稳定 ID 查询 Core 保存的项目文档。
	Get(string) (dto.Document, bool)
}
