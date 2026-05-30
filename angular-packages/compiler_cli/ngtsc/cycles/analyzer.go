package cycles

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type CycleAnalyzer struct {
	importGraph   *ImportGraph
	cachedResults *CycleResults
}

func NewCycleAnalyzer(importGraph *ImportGraph) *CycleAnalyzer {
	return &CycleAnalyzer{
		importGraph: importGraph,
	}
}

func (a *CycleAnalyzer) WouldCreateCycle(from *ast.SourceFile, to *ast.SourceFile) *Cycle {
	if a.cachedResults == nil || a.cachedResults.from != from {
		a.cachedResults = NewCycleResults(from, a.importGraph)
	}

	if a.cachedResults.WouldBeCyclic(to) {
		return NewCycle(a.importGraph, from, to)
	}
	return nil
}

func (a *CycleAnalyzer) RecordSyntheticImport(from *ast.SourceFile, to *ast.SourceFile) {
	a.cachedResults = nil
	a.importGraph.AddSyntheticImport(from, to)
}

type CycleResults struct {
	from        *ast.SourceFile
	importGraph *ImportGraph
	results     map[*ast.SourceFile]bool
}

func NewCycleResults(from *ast.SourceFile, importGraph *ImportGraph) *CycleResults {
	return &CycleResults{
		from:        from,
		importGraph: importGraph,
		results:     make(map[*ast.SourceFile]bool),
	}
}

func (r *CycleResults) WouldBeCyclic(sf *ast.SourceFile) bool {
	if val, ok := r.results[sf]; ok {
		return val
	}

	if sf == r.from {
		return true
	}

	r.results[sf] = false

	imports := r.importGraph.ImportsOf(sf)
	for imported := range imports {
		if r.WouldBeCyclic(imported) {
			r.results[sf] = true
			return true
		}
	}
	return false
}

type Cycle struct {
	importGraph *ImportGraph
	From        *ast.SourceFile
	To          *ast.SourceFile
}

func NewCycle(importGraph *ImportGraph, from *ast.SourceFile, to *ast.SourceFile) *Cycle {
	return &Cycle{
		importGraph: importGraph,
		From:        from,
		To:          to,
	}
}

func (c *Cycle) GetPath() []*ast.SourceFile {
	path := c.importGraph.FindPath(c.To, c.From)
	res := make([]*ast.SourceFile, 0, len(path)+1)
	res = append(res, c.From)
	res = append(res, path...)
	return res
}

type CycleHandlingStrategy int

const (
	CycleHandlingStrategy_UseRemoteScoping CycleHandlingStrategy = iota
	CycleHandlingStrategy_Error
)
