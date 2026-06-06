package incremental

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type IncrementalBuild interface {
	PriorAnalysisFor(sf *ast.SourceFile) []any
	PriorTypeCheckingResultsFor(fileSf *ast.SourceFile) any
	RecordSuccessfulTypeCheck(results map[string]any) any
}

type DependencyTracker interface {
	AddDependency(from any, on any) any
	AddResourceDependency(from any, on string) any
	RecordDependencyAnalysisFailure(file any) any
}
