package indexer

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/internal/ast"
)

type ComponentTemplateMeta struct {
	IsInline bool
	File     *parse_util.ParseSourceFile
}

type ComponentInfo struct {
	Declaration   *ast.Node
	Selector      *string
	BoundTemplate AbstractBoundTemplate
	TemplateMeta  ComponentTemplateMeta
}

type IndexingContext struct {
	Components []*ComponentInfo
}

func NewIndexingContext() *IndexingContext {
	return &IndexingContext{
		Components: []*ComponentInfo{},
	}
}

func (ctx *IndexingContext) AddComponent(info *ComponentInfo) {
	ctx.Components = append(ctx.Components, info)
}
