package incremental

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// Phase 0: Stable Identity Types
type FilePath = string
type ResourcePath = string
type DeclarationKey = string
type SemanticSymbolKey = string
type BuildVersion = string

// Phase 3: Analysis State Reuse Types
type AnalyzedTrait struct {
	HandlerName string
	Decorator   any // Stores reflection.Decorator (cannot import due to circular dep)
	Analysis    any
	Diagnostics []ast.Diagnostic
	State       int // Stores transform.TraitState
}

type FileAnalysisState struct {
	FilePath string
	Version  string
	Classes  map[string][]AnalyzedTrait // ClassName -> AnalyzedTraits
}

// Phase 7: Type Check State Reuse Types
type TypeCheckState struct {
	FilePath    string
	Diagnostics []ast.Diagnostic
}

// IncrementalState represents the persisted state between compilations.
type IncrementalState struct {
	Versions      map[string]string
	EmittedFiles  map[string]bool
	Analysis      map[string]FileAnalysisState
	TypeCheck     map[string]TypeCheckState
	DepGraph      *FileDependencyGraph
	SemanticGraph any
}
