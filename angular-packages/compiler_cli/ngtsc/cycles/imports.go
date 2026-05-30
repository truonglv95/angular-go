package cycles

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/perf"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
)

type ImportGraph struct {
	program *compiler.Program
	imports map[*ast.SourceFile]map[*ast.SourceFile]struct{}
	perf    perf.PerfRecorder
}

func NewImportGraph(program *compiler.Program, perfRecorder perf.PerfRecorder) *ImportGraph {
	return &ImportGraph{
		program: program,
		imports: make(map[*ast.SourceFile]map[*ast.SourceFile]struct{}),
		perf:    perfRecorder,
	}
}

func (g *ImportGraph) ImportsOf(sf *ast.SourceFile) map[*ast.SourceFile]struct{} {
	if _, ok := g.imports[sf]; !ok {
		g.imports[sf] = g.scanImports(sf)
	}
	return g.imports[sf]
}

func (g *ImportGraph) FindPath(start *ast.SourceFile, end *ast.SourceFile) []*ast.SourceFile {
	if start == end {
		return []*ast.SourceFile{start}
	}

	foundMap := make(map[*ast.SourceFile]bool)
	foundMap[start] = true

	queue := []*Found{{sourceFile: start, parent: nil}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		imports := g.ImportsOf(current.sourceFile)
		for importedFile := range imports {
			if !foundMap[importedFile] {
				next := &Found{sourceFile: importedFile, parent: current}
				if next.sourceFile == end {
					return next.toPath()
				}
				foundMap[importedFile] = true
				queue = append(queue, next)
			}
		}
	}
	return nil
}

func (g *ImportGraph) AddSyntheticImport(sf *ast.SourceFile, imported *ast.SourceFile) {
	if isLocalFile(imported) {
		g.ImportsOf(sf)[imported] = struct{}{}
	}
}

func (g *ImportGraph) scanImports(sf *ast.SourceFile) map[*ast.SourceFile]struct{} {
	res := g.perf.InPhase(perf.PerfPhase_CycleDetection, func() interface{} {
		imports := make(map[*ast.SourceFile]struct{})
		if sf == nil || sf.Statements == nil {
			return imports
		}

		for _, stmt := range sf.Statements.Nodes {
			if stmt == nil {
				continue
			}

			var moduleSpecifier *ast.Node

			if ast.IsImportDeclaration(stmt) {
				decl := stmt.AsImportDeclaration()

				// Skip type-only imports - they are always elided and don't contribute to cycles.
				if decl.ImportClause != nil && isTypeOnlyImportClause(decl.ImportClause) {
					continue
				}

				moduleSpecifier = decl.ModuleSpecifier
			} else if ast.IsExportDeclaration(stmt) {
				decl := stmt.AsExportDeclaration()

				// Skip type-only re-exports.
				if decl.IsTypeOnly {
					continue
				}

				moduleSpecifier = decl.ModuleSpecifier
			} else {
				continue
			}

			if moduleSpecifier == nil {
				continue
			}

			if !ast.IsStringLiteralLike(moduleSpecifier) {
				continue
			}

			// Resolve the module specifier to a source file using the program.
			resolved := g.program.GetResolvedModuleFromModuleSpecifier(sf, moduleSpecifier)
			if resolved == nil || !resolved.IsResolved() {
				continue
			}

			importedSF := g.program.GetSourceFileForResolvedModule(resolved.ResolvedFileName)
			if importedSF != nil && isLocalFile(importedSF) {
				imports[importedSF] = struct{}{}
			}
		}
		return imports
	})
	return res.(map[*ast.SourceFile]struct{})
}

func isLocalFile(sf *ast.SourceFile) bool {
	return sf != nil && !sf.IsDeclarationFile
}

// isTypeOnlyImportClause checks whether an import clause is type-only.
// This mirrors the TypeScript isTypeOnlyImportClause function.
func isTypeOnlyImportClause(importClause *ast.ImportClauseNode) bool {
	if importClause == nil {
		return false
	}

	clause := importClause.AsImportClause()

	// The clause itself is type-only (e.g. `import type {foo} from '...'`)
	if clause.PhaseModifier == ast.KindTypeKeyword {
		return true
	}

	// Check if all specifiers in named bindings are type-only
	// (e.g. `import {type a, type b} from '...'`)
	if clause.NamedBindings != nil {
		namedBindings := clause.NamedBindings
		if ast.IsNamedImports(namedBindings) {
			namedImports := namedBindings.AsNamedImports()
			elements := namedImports.Elements.Nodes
			if len(elements) > 0 {
				allTypeOnly := true
				for _, elem := range elements {
					if elem != nil {
						spec := elem.AsImportSpecifier()
						if !spec.IsTypeOnly {
							allTypeOnly = false
							break
						}
					}
				}
				if allTypeOnly {
					return true
				}
			}
		}
	}

	return false
}

type Found struct {
	sourceFile *ast.SourceFile
	parent     *Found
}

func (f *Found) toPath() []*ast.SourceFile {
	var array []*ast.SourceFile
	current := f
	for current != nil {
		array = append(array, current.sourceFile)
		current = current.parent
	}
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}
	return array
}
