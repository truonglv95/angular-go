package scope

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/internal/ast"
)

type ComponentScopeKind int

const (
	ComponentScopeKindNgModule ComponentScopeKind = iota
	ComponentScopeKindStandalone
)

type ScopeDependency struct {
	Kind      metadata.MetaKind
	Directive *metadata.DirectiveMeta
	Pipe      *metadata.PipeMeta
	NgModule  *metadata.NgModuleMeta
}

type ExportScope struct {
	Dependencies []ScopeDependency
	IsPoisoned   bool
}

type StandaloneScope struct {
	Component            *ast.Node
	Dependencies         []ScopeDependency
	DeferredDependencies []ScopeDependency
	IsPoisoned           bool
}

type NgModuleScope struct {
	NgModule    *ast.Node
	Compilation ExportScope
	Exported    ExportScope
}

// CompilationScope represents the set of directives, components, and pipes visible in a component's template.
// Keeping this for backward compatibility temporarily.
type CompilationScope struct {
	Directives []metadata.DirectiveMeta
	Pipes      []metadata.PipeMeta
}

// ScopeReader defines the reader interface for compilation scopes.
type ScopeReader interface {
	GetCompilationScope(component *ast.Node) *CompilationScope
	
	GetStandaloneScope(node *ast.Node) *StandaloneScope
}
