package incremental

import (
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
)

type IncrementalStep struct {
	PriorState *IncrementalState
}

// IncrementalCompilation manages the incremental compilation reuse logic.
type IncrementalCompilation struct {
	Versions      map[string]string
	State         *IncrementalState
	Step          *IncrementalStep
	mu            sync.RWMutex
	modifiedFiles map[string]bool
	affectedFiles map[string]bool
}

func Fresh(versions map[string]string) *IncrementalCompilation {
	return &IncrementalCompilation{
		Versions: versions,
		State: &IncrementalState{
			Versions:     versions,
			EmittedFiles: make(map[string]bool),
			Analysis:     make(map[string]FileAnalysisState),
			TypeCheck:    make(map[string]TypeCheckState),
			DepGraph:     NewFileDependencyGraph(),
		},
	}
}

func Incremental(
	program any,
	newVersions map[string]string,
	oldProgram any,
	oldState *IncrementalState,
	modifiedResourceFiles map[string]bool,
	perf any,
) *IncrementalCompilation {
	step := &IncrementalStep{
		PriorState: oldState,
	}
	comp := &IncrementalCompilation{
		Versions: newVersions,
		State: &IncrementalState{
			Versions:     newVersions,
			EmittedFiles: make(map[string]bool),
			Analysis:     make(map[string]FileAnalysisState),
			TypeCheck:    make(map[string]TypeCheckState),
			DepGraph:     NewFileDependencyGraph(),
		},
		Step:          step,
		modifiedFiles: modifiedResourceFiles,
	}

	// Compute affected files transitively using the dependency graph
	if oldState != nil && oldState.DepGraph != nil {
		changedTs := make(map[string]bool)
		deletedTs := make(map[string]bool)
		changedResources := make(map[string]bool)

		// 1. Detect changed/deleted TS files
		for file, oldVer := range oldState.Versions {
			if newVer, ok := newVersions[file]; !ok {
				deletedTs[file] = true
			} else if oldVer != newVer {
				changedTs[file] = true
			}
		}
		for file := range newVersions {
			if _, ok := oldState.Versions[file]; !ok {
				changedTs[file] = true
			}
		}

		// 2. Detect changed resources
		for file := range modifiedResourceFiles {
			if !strings.HasSuffix(file, ".ts") {
				changedResources[file] = true
			} else {
				changedTs[file] = true
			}
		}

		// Update our new DepGraph with the changes and compute affected files
		comp.affectedFiles = comp.State.DepGraph.UpdateWithPhysicalChanges(
			*oldState.DepGraph,
			changedTs,
			deletedTs,
			changedResources,
		)
	}

	return comp
}

func (c *IncrementalCompilation) RecordSuccessfulAnalysis(traitCompiler any) any {
	collector, ok := traitCompiler.(interface {
		CollectAnalysis() map[string]FileAnalysisState
	})
	if !ok || collector == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.State.Analysis = collector.CollectAnalysis()
	if tcGraph, ok := traitCompiler.(interface {
		GetSemanticGraph() any
	}); ok {
		c.State.SemanticGraph = tcGraph.GetSemanticGraph()
	}

	return nil
}

func (c *IncrementalCompilation) RecordSuccessfulTypeCheck(results map[string]any) any {
	c.mu.Lock()
	defer c.mu.Unlock()

	for file, res := range results {
		diags, _ := res.([]ast.Diagnostic)
		c.State.TypeCheck[file] = TypeCheckState{
			FilePath:    file,
			Diagnostics: diags,
		}
	}
	return nil
}

func (c *IncrementalCompilation) RecordSuccessfulEmit(sf *ast.SourceFile) any {
	if sf == nil {
		return nil
	}
	c.mu.Lock()
	c.State.EmittedFiles[sf.FileName()] = true
	c.mu.Unlock()
	return nil
}

func (c *IncrementalCompilation) PriorAnalysisFor(sf *ast.SourceFile) []any {
	if sf == nil {
		return nil
	}
	fileName := sf.FileName()

	// If the file is affected by code or resource changes, we cannot reuse its prior analysis
	if c.affectedFiles != nil && c.affectedFiles[canonicalizePath(fileName)] {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Step == nil || c.Step.PriorState == nil {
		return nil
	}

	priorState := c.Step.PriorState
	oldVer, existsInOld := priorState.Versions[fileName]
	newVer, existsInNew := c.Versions[fileName]
	if !existsInOld || !existsInNew || oldVer != newVer {
		return nil
	}

	analysisState, found := priorState.Analysis[fileName]
	if !found {
		return nil
	}

	var results []any
	for className, traits := range analysisState.Classes {
		results = append(results, map[string]any{
			"className": className,
			"traits":    traits,
		})
	}
	return results
}

func (c *IncrementalCompilation) PriorTypeCheckingResultsFor(sf *ast.SourceFile) any {
	if sf == nil {
		return nil
	}
	fileName := sf.FileName()

	if c.affectedFiles != nil && c.affectedFiles[canonicalizePath(fileName)] {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Step == nil || c.Step.PriorState == nil {
		return nil
	}

	priorState := c.Step.PriorState
	oldVer, existsInOld := priorState.Versions[fileName]
	newVer, existsInNew := c.Versions[fileName]
	if !existsInOld || !existsInNew || oldVer != newVer {
		return nil
	}

	if typeCheck, ok := priorState.TypeCheck[fileName]; ok {
		return typeCheck.Diagnostics
	}
	return nil
}

func (c *IncrementalCompilation) SafeToSkipEmit(sf *ast.SourceFile) bool {
	if sf == nil {
		return false
	}
	fileName := sf.FileName()

	c.mu.RLock()
	defer c.mu.RUnlock()

	canonicalName := canonicalizePath(fileName)
	if c.modifiedFiles != nil && c.modifiedFiles[canonicalName] {
		return false
	}

	if c.affectedFiles != nil && c.affectedFiles[canonicalName] {
		return false
	}

	if c.Step != nil && c.Step.PriorState != nil {
		if c.Step.PriorState.SemanticGraph == nil {
			return false
		}
		if graph, ok := c.Step.PriorState.SemanticGraph.(interface{ HasReferences() bool }); ok && !graph.HasReferences() {
			return false
		}
		oldVer, existsInOld := c.Step.PriorState.Versions[fileName]
		newVer, existsInNew := c.Versions[fileName]
		if existsInOld && existsInNew && oldVer != newVer {
			return false
		}
		return c.Step.PriorState.EmittedFiles[fileName]
	}

	return c.State.EmittedFiles[fileName]
}

// AddDependency implements DependencyTracker
func (c *IncrementalCompilation) AddDependency(from any, on any) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State.DepGraph == nil {
		c.State.DepGraph = NewFileDependencyGraph()
	}
	c.State.DepGraph.AddDependency(from, on)
	return nil
}

// AddResourceDependency implements DependencyTracker
func (c *IncrementalCompilation) AddResourceDependency(from any, resource string) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State.DepGraph == nil {
		c.State.DepGraph = NewFileDependencyGraph()
	}
	c.State.DepGraph.AddResourceDependency(from, resource)
	return nil
}

// RecordDependencyAnalysisFailure implements DependencyTracker
func (c *IncrementalCompilation) RecordDependencyAnalysisFailure(file any) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State.DepGraph == nil {
		c.State.DepGraph = NewFileDependencyGraph()
	}
	c.State.DepGraph.RecordDependencyAnalysisFailure(file)
	return nil
}

func (c *IncrementalCompilation) IsFileAffected(fileName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.affectedFiles == nil {
		return true
	}
	return c.affectedFiles[fileName]
}

func (c *IncrementalCompilation) AffectedFiles() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.affectedFiles
}

