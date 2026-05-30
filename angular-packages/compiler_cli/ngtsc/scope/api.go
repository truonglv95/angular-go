package scope

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/internal/ast"
)

// CompilationScope represents the set of directives, components, and pipes visible in a component's template.
type CompilationScope struct {
	Directives []metadata.DirectiveMeta
	Pipes      []metadata.PipeMeta
}

// ScopeReader defines the reader interface for compilation scopes.
type ScopeReader interface {
	GetCompilationScope(component *ast.Node) *CompilationScope
}
