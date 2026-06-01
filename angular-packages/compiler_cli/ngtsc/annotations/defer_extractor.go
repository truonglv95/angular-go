package annotations

import (
	"path/filepath"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
)

func extractDeferBlocks(nodes []render3.Node, compScope *scope.CompilationScope, componentNode *ast.Node, imports []ComponentImport) (map[any]render3.Expression, []render3.R3DeferPerComponentDependency) {
	if compScope == nil {
		return nil, nil
	}

	blocks := make(map[any]render3.Expression)
	var componentDeps []render3.R3DeferPerComponentDependency
	seenComponentDeps := make(map[string]bool)

	var visitNodes func(nodes []render3.Node)
	var visitDeferBlock func(block *render3.DeferredBlock)

	visitDeferBlock = func(block *render3.DeferredBlock) {
		var deps []render3.R3DeferPerBlockDependency

		// Simple extractor: find all Element nodes inside the defer block
		// and match them against compScope.Directives
		seenElements := make(map[string]bool)

		var findElements func(ns []render3.Node)
		findElements = func(ns []render3.Node) {
			for _, n := range ns {
				if el, ok := n.(*render3.Element); ok {
					seenElements[el.Name] = true
					findElements(el.Children)
				} else if tpl, ok := n.(*render3.Template); ok {
					findElements(tpl.Children)
				} else if db, ok := n.(*render3.DeferredBlock); ok {
					// Nested defer blocks: we DO NOT collect from their main Children,
					// because the inner defer block will lazy-load those itself.
					// We DO collect from their Placeholder/Loading/Error blocks,
					// because those must be rendered eagerly by the *outer* block.
					if db.Placeholder != nil {
						findElements(db.Placeholder.Children)
					}
					if db.Loading != nil {
						findElements(db.Loading.Children)
					}
					if db.Error != nil {
						findElements(db.Error.Children)
					}
				}
			}
		}
		findElements(block.Children)

		for _, dir := range compScope.Directives {
			// Very naive selector matching for tags (e.g. "app-heavy")
			matched := false
			if dir.Selector != "" {
				// Strip brackets and dots for a naive match
				sel := dir.Selector
				if !strings.Contains(sel, "[") && !strings.Contains(sel, ".") {
					if seenElements[sel] {
						matched = true
					}
				}
			}

			if matched {
				importPath := dir.Ref.OwningModule
				symbolName := dir.Name
				localName := dir.Name
				if imp, ok := findDeferredComponentImport(imports, dir); ok {
					localName = imp.Name
					if imp.ImportName != "" {
						symbolName = imp.ImportName
					}
					if imp.ImportPath != "" {
						importPath = imp.ImportPath
					}
				}
				if importPath == "" {
					if dir.Ref.Node != nil && componentNode != nil {
						// Check if they are in different files
						sourceFile := ast.GetSourceFileOfNode(componentNode)
						targetFile := ast.GetSourceFileOfNode(dir.Ref.Node)
						if sourceFile != nil && targetFile != nil && sourceFile != targetFile {
							// Compute relative path
							fromDir := filepath.Dir(sourceFile.FileName())
							toPath := targetFile.FileName()
							rel, err := filepath.Rel(fromDir, toPath)
							if err == nil {
								// Strip extension
								if ext := filepath.Ext(rel); ext != "" {
									rel = rel[:len(rel)-len(ext)]
								}
								if !strings.HasPrefix(rel, ".") {
									rel = "./" + rel
								}
								// Always use forward slashes for TS imports
								importPath = strings.ReplaceAll(rel, "\\", "/")
							}
						}
					}
				}

				dep := render3.R3DeferPerBlockDependency{
					SymbolName:   symbolName,
					IsDeferrable: importPath != "", // Deferrable if it comes from an import
				}
				if importPath != "" {
					dep.ImportPath = &importPath
				}

				// fallback type reference
				if importPath != "" {
					dep.TypeReference = output.NewExternalExpr(output.ExternalReference{
						ModuleName: &importPath,
						Name:       &symbolName,
					}, nil, nil, nil, nil)
				} else {
					dep.TypeReference = output.NewReadVarExpr(localName, nil, nil, nil)
				}

				deps = append(deps, dep)

				componentDepKey := importPath + "#" + symbolName + "#" + localName
				if importPath != "" && !seenComponentDeps[componentDepKey] {
					seenComponentDeps[componentDepKey] = true
					componentDeps = append(componentDeps, render3.R3DeferPerComponentDependency{
						SymbolName: symbolName,
						ImportPath: importPath,
						LocalName:  localName,
					})
				}
			}
		}

		if len(deps) > 0 {
			meta := render3.R3DeferResolverFunctionMetadata{
				Mode:         render3.DeferBlockDepsEmitMode_PerBlock,
				Dependencies: deps,
			}
			blocks[block] = render3.CompileDeferResolverFunction(meta)
		} else {
			blocks[block] = nil // Emits null for dependencies
		}

		// Continue visiting children for nested defer blocks
		visitNodes(block.Children)
	}

	visitNodes = func(nodes []render3.Node) {
		for _, n := range nodes {
			if block, ok := n.(*render3.DeferredBlock); ok {
				visitDeferBlock(block)
			} else if el, ok := n.(*render3.Element); ok {
				visitNodes(el.Children)
			} else if tpl, ok := n.(*render3.Template); ok {
				visitNodes(tpl.Children)
			}
		}
	}

	visitNodes(nodes)

	return blocks, componentDeps
}

func findDeferredComponentImport(imports []ComponentImport, dir metadata.DirectiveMeta) (ComponentImport, bool) {
	for _, imp := range imports {
		if imp.Decl.Node != nil && dir.Ref.Node != nil && imp.Decl.Node == dir.Ref.Node {
			return imp, true
		}
		if imp.Name == dir.Ref.Name || imp.Name == dir.Name || imp.ImportName == dir.Ref.Name || imp.ImportName == dir.Name {
			return imp, true
		}
	}
	return ComponentImport{}, false
}
