package incremental

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type IncrementalStep struct {
	PriorState *IncrementalState
}

// IncrementalCompilation manages the incremental compilation reuse logic.
type IncrementalCompilation struct {
	Versions map[string]string
	State    *IncrementalState
	Step     *IncrementalStep
}

func Fresh(versions map[string]string) *IncrementalCompilation {
	return &IncrementalCompilation{
		Versions: versions,
		State: &IncrementalState{
			Versions:     versions,
			EmittedFiles: make(map[string]bool),
		},
	}
}

func Incremental(program any, newVersions map[string]string, oldProgram any, oldState *IncrementalState, modifiedResourceFiles map[string]bool, perf any) *IncrementalCompilation {
	step := &IncrementalStep{
		PriorState: oldState,
	}
	return &IncrementalCompilation{
		Versions: newVersions,
		State: &IncrementalState{
			Versions:     newVersions,
			EmittedFiles: make(map[string]bool),
		},
		Step: step,
	}
}

func (c *IncrementalCompilation) RecordSuccessfulAnalysis(traitCompiler any) any {
	return nil
}

func (c *IncrementalCompilation) RecordSuccessfulTypeCheck(results map[string]any) any {
	return nil
}

func (c *IncrementalCompilation) RecordSuccessfulEmit(sf *ast.SourceFile) any {
	if sf == nil {
		return nil
	}
	c.State.EmittedFiles[sf.FileName()] = true
	return nil
}

func (c *IncrementalCompilation) PriorAnalysisFor(sf *ast.SourceFile) []any {
	return nil
}

func (c *IncrementalCompilation) PriorTypeCheckingResultsFor(sf *ast.SourceFile) any {
	return nil
}

func (c *IncrementalCompilation) SafeToSkipEmit(sf *ast.SourceFile) bool {
	if sf == nil {
		return false
	}
	fileName := sf.FileName()

	if c.Step != nil && c.Step.PriorState != nil {
		oldVer, existsInOld := c.Step.PriorState.Versions[fileName]
		newVer, existsInNew := c.Versions[fileName]
		if existsInOld && existsInNew && oldVer != newVer {
			return false // Version changed, cannot skip emit
		}
		return c.Step.PriorState.EmittedFiles[fileName]
	}

	return c.State.EmittedFiles[fileName]
}
