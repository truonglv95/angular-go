package metadata

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/internal/ast"
)

// MetaKind defines the kind of metadata.
type MetaKind int

const (
	MetaKindDirective MetaKind = iota
	MetaKindComponent
	MetaKindNgModule
	MetaKindPipe
)

// Reference represents a reference to a declaration node.
type Reference struct {
	Name         string
	Node         *ast.Node
	OwningModule string
}

// DirectiveMeta represents metadata collected for a directive or component.
type DirectiveMeta struct {
	Name         string
	Kind         MetaKind
	Ref          Reference
	Selector     string
	Inputs       map[string]string
	Outputs      map[string]string
	HostBindings map[string]string
	ExportAs     []string
	Standalone   bool
	IsComponent  bool
	Imports      []Reference
}

// NgModuleMeta represents metadata collected for an NgModule.
type NgModuleMeta struct {
	Name         string
	Ref          Reference
	Declarations []Reference
	Imports      []Reference
	Exports      []Reference
	Bootstrap    []Reference
}

// PipeMeta represents metadata collected for a Pipe.
type PipeMeta struct {
	Ref        Reference
	Name       string
	Pure       bool
	Standalone bool
}

// MetadataReader reads metadata for Angular decorators.
type MetadataReader interface {
	GetDirectiveMetadata(node *ast.Node) *DirectiveMeta
	GetNgModuleMetadata(node *ast.Node) *NgModuleMeta
	GetPipeMetadata(node *ast.Node) *PipeMeta
}

// MetadataRegistry registers metadata for Angular decorators.
type MetadataRegistry interface {
	RegisterDirective(node *ast.Node, meta *DirectiveMeta)
	RegisterNgModule(node *ast.Node, meta *NgModuleMeta)
	RegisterPipe(node *ast.Node, meta *PipeMeta)
}

type SemanticMetadataReader interface {
	GetSemanticSymbol(node *ast.Node) *semantic_graph.SemanticSymbol
}
