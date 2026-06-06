package incremental

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

var NOOP_INCREMENTAL_BUILD IncrementalBuild

type noopIncrementalBuild struct{}

func (n noopIncrementalBuild) PriorAnalysisFor(sf *ast.SourceFile) []any {
	return nil
}

func (n noopIncrementalBuild) PriorTypeCheckingResultsFor(fileSf *ast.SourceFile) any {
	return nil
}

func (n noopIncrementalBuild) RecordSuccessfulTypeCheck(results map[string]any) any {
	return nil
}

func init() {
	NOOP_INCREMENTAL_BUILD = noopIncrementalBuild{}
}
