package transform

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/printer"
)

// TraitCompiler là bộ điều phối trung tâm của ngtsc. Nó quản lý tất cả các class
// trong chương trình, chạy qua các bước Analyze -> Resolve -> Compile.
type TraitCompiler struct {
	handlers  []DecoratorHandler
	host      reflection.ReflectionHost
	localHost reflection.ReflectionHost // AST-only host
	oldTc     *TraitCompiler            // Reference to previous compilation

	// classes lưu trữ danh sách các Traits cho mỗi class declaration.
	mu      sync.RWMutex
	classes map[*ast.ClassDeclaration][]*Trait
	files   map[*ast.SourceFile]bool

	// HmrUpdates lưu trữ string HMR generated
	HmrUpdates map[string]string

	// H3 FIX: Cache cwd once at construction to avoid repeated os.Getwd() syscalls
	// inside the hot UpdateSourceFile loop (one call per component class, per compile).
	cwd string

	originalStatements     map[*ast.SourceFile][]*ast.Node
	originalClassMembers   map[*ast.ClassDeclaration][]*ast.Node
	originalClassModifiers map[*ast.ClassDeclaration][]*ast.Node
}

func NewTraitCompiler(handlers []DecoratorHandler, host reflection.ReflectionHost, localHost reflection.ReflectionHost, oldTc *TraitCompiler) *TraitCompiler {
	cwd, _ := os.Getwd()
	tc := &TraitCompiler{
		handlers:               handlers,
		host:                   host,
		localHost:              localHost,
		oldTc:                  oldTc,
		classes:                make(map[*ast.ClassDeclaration][]*Trait),
		files:                  make(map[*ast.SourceFile]bool),
		HmrUpdates:             make(map[string]string),
		cwd:                    cwd,
		originalStatements:     make(map[*ast.SourceFile][]*ast.Node),
		originalClassMembers:   make(map[*ast.ClassDeclaration][]*ast.Node),
		originalClassModifiers: make(map[*ast.ClassDeclaration][]*ast.Node),
	}
	if oldTc != nil {
		tc.originalStatements = oldTc.originalStatements
		tc.originalClassMembers = oldTc.originalClassMembers
		tc.originalClassModifiers = oldTc.originalClassModifiers
	}
	return tc
}

// AnalyzeSync quét qua một source file, phát hiện các class và chạy hàm Analyze của các Handler.
func (tc *TraitCompiler) AnalyzeSync(sf *ast.SourceFile) {
	if sf == nil || sf.Statements == nil {
		return
	}

	var hasTraits bool
	sf.AsNode().ForEachChild(func(node *ast.Node) bool {
		if node.Kind == ast.KindClassDeclaration {
			classDecl := node.AsClassDeclaration()
			if tc.analyzeClass(classDecl) {
				hasTraits = true
			}
		}
		return false
	})
	if hasTraits {
		tc.mu.Lock()
		tc.files[sf] = true
		tc.mu.Unlock()
	}
}

// AnalyzeSyncLocal is the AST-based version of AnalyzeSync.
func (tc *TraitCompiler) AnalyzeSyncLocal(sf *ast.SourceFile) {
	if sf == nil || sf.Statements == nil {
		return
	}

	var hasTraits bool
	sf.AsNode().ForEachChild(func(node *ast.Node) bool {
		if node.Kind == ast.KindClassDeclaration {
			classDecl := node.AsClassDeclaration()
			if tc.analyzeClassLocal(classDecl) {
				hasTraits = true
			}
		}
		return false
	})
	if hasTraits {
		tc.mu.Lock()
		tc.files[sf] = true
		tc.mu.Unlock()
	}
}

func (tc *TraitCompiler) analyzeClassLocal(classDecl *ast.ClassDeclaration) bool {
	decorators := tc.localHost.GetDecoratorsOfDeclaration(classDecl.AsNode())
	if len(decorators) == 0 {
		return false
	}

	var traits []*Trait

	// 1. Detect
	for _, handler := range tc.handlers {
		dec := handler.Detect(classDecl, decorators)
		if dec != nil {
			traits = append(traits, &Trait{
				Handler:   handler,
				Decorator: dec,
				State:     TraitStatePending,
			})
		}
	}

	if len(traits) == 0 {
		return false
	}

	tc.mu.Lock()
	tc.classes[classDecl] = traits
	tc.mu.Unlock()

	// 2. Analyze
	for _, trait := range traits {
		analysis, _ := trait.Handler.Analyze(classDecl, trait.Decorator)
		trait.Analysis = analysis
		trait.State = TraitStateAnalyzed
	}
	return true
}

func (tc *TraitCompiler) analyzeClass(classDecl *ast.ClassDeclaration) bool {
	decorators := tc.host.GetDecoratorsOfDeclaration(classDecl.AsNode())
	if len(decorators) == 0 {
		return false
	}

	var traits []*Trait

	// 1. Detect
	for _, handler := range tc.handlers {
		dec := handler.Detect(classDecl, decorators)
		if dec != nil {
			traits = append(traits, &Trait{
				Handler:   handler,
				Decorator: dec,
				State:     TraitStatePending,
			})
		}
	}

	if len(traits) == 0 {
		return false
	}

	// C1 FIX: Add lock when writing to tc.classes — same as analyzeClassLocal.
	// analyzeClass runs in parallel goroutines (after B#1 fix) so concurrent
	// writes to this map would cause a data race detected by go test -race.
	tc.mu.Lock()
	tc.classes[classDecl] = traits
	tc.mu.Unlock()

	// 2. Analyze (outside lock — handler.Analyze is read-only)
	for _, trait := range traits {
		analysis, _ := trait.Handler.Analyze(classDecl, trait.Decorator)
		trait.Analysis = analysis
		trait.State = TraitStateAnalyzed
	}
	return true
}

// Resolve chạy bước giải quyết tham chiếu cho tất cả các traits đã được Analyze (song song).
func (tc *TraitCompiler) Resolve() {
	var wg sync.WaitGroup

	for classDecl, traits := range tc.classes {
		for _, trait := range traits {
			if trait.State == TraitStateAnalyzed || trait.State == TraitStateResolved {
				wg.Add(1)
				go func(c *ast.ClassDeclaration, t *Trait) {
					defer wg.Done()
					t.GetResolution(c)
				}(classDecl, trait)
			}
		}
	}
	wg.Wait()
}

// UpdateSourceFile áp dụng kết quả compile (CompileResult) vào AST của source file.
func (tc *TraitCompiler) UpdateSourceFile(sf *ast.SourceFile, factory *ast.NodeFactory) {
	if sf == nil || sf.Statements == nil || !tc.files[sf] {
		return
	}

	tc.mu.Lock()
	if origStmts, ok := tc.originalStatements[sf]; ok {
		sf.Statements.Nodes = append([]*ast.Node(nil), origStmts...)
	} else {
		tc.originalStatements[sf] = append([]*ast.Node(nil), sf.Statements.Nodes...)
	}
	tc.mu.Unlock()

	decoratorNames := collectDecoratorNames(sf)
	pool := compiler.NewConstantPool(false)
	config := imports.PresetImportManagerForceNamespaceImports
	importMgr := imports.NewImportManager(&config, factory)
	var hasTransformedClass bool
	var newStatements []*ast.Node

	for _, node := range sf.Statements.Nodes {
		if node.Kind == ast.KindClassDeclaration {
			classDecl := node.AsClassDeclaration()

			tc.mu.Lock()
			if origMembers, ok := tc.originalClassMembers[classDecl]; ok {
				classDecl.Members.Nodes = append([]*ast.Node(nil), origMembers...)
			} else {
				tc.originalClassMembers[classDecl] = append([]*ast.Node(nil), classDecl.Members.Nodes...)
			}
			if classDecl.Modifiers() != nil {
				if origMods, ok := tc.originalClassModifiers[classDecl]; ok {
					classDecl.Modifiers().Nodes = append([]*ast.Node(nil), origMods...)
				} else {
					tc.originalClassModifiers[classDecl] = append([]*ast.Node(nil), classDecl.Modifiers().Nodes...)
				}
			}
			tc.mu.Unlock()

			traits := tc.classes[classDecl]

			var additionalMembers []*ast.Node
			var postStatements []*ast.Node
			for _, trait := range traits {
				if trait.State == TraitStateResolved || trait.State == TraitStateAnalyzed {
					resolution := trait.GetResolution(classDecl)
					results, _ := trait.Handler.CompileFull(classDecl, trait.Analysis, resolution, pool, importMgr, factory)
					for _, result := range results {
						if result.Initializer != nil {
							// Create static property (e.g. ɵcmp, ɵfac)
							modifiers := factory.NewModifierList([]*ast.Node{factory.NewModifier(ast.KindStaticKeyword)})
							name := factory.NewIdentifier(result.PropertyName)

							prop := factory.NewPropertyDeclaration(modifiers, (*ast.PropertyName)(name), nil, nil, (*ast.Expression)(result.Initializer))
							additionalMembers = append(additionalMembers, prop.AsNode())
						}
						if result.Statements != nil {
							for i, s := range result.Statements {
								if s == nil {
									panic(fmt.Sprintf("result.Statements[%d] is nil!", i))
								}
							}
							postStatements = append(postStatements, result.Statements...)
						}
						if result.HmrUpdateNodes != nil {
							// NOTE: HmrUpdateNodes (App_UpdateMetadata function) must NOT
							// be added to postStatements. It belongs ONLY in the virtual
							// HMR module (/@ng/component) — not in the main compiled output.
							// Adding it here caused the printer to emit:
							//   export default App_UpdateMetadata;  ← in app.js
							// and the browser's HMR module had only this reference, causing:
							//   ReferenceError: App_UpdateMetadata is not defined

							// Populate HmrUpdates map with generated code.
							// Key format: "relPath@ClassName" (forward slashes, no encoding)
							// This matches what the Vite plugin sends after URL-decoding the ?c= param.
							className := ""
							if classDecl.Name() != nil {
								className = classDecl.Name().AsIdentifier().Text
							}
							cwd := tc.cwd
							relPath, err := filepath.Rel(cwd, sf.FileName())
							if err != nil || strings.HasPrefix(relPath, "..") {
								relPath = sf.FileName()
							}
							relPath = filepath.ToSlash(relPath)
							key := fmt.Sprintf("%s@%s", relPath, className)

							// Build the self-contained virtual source file for the HMR module.
							//
							// IMPORTANT: use result.HmrImports (the imports generated by the
							// per-component hmrImportMgr inside getHmrUpdateDeclaration) — NOT
							// importMgr.GetAllImports(). The two managers produce different
							// namespace identifiers:
							//   main importMgr  →  import * as i0 from '@angular/core'
							//   hmrImportMgr    →  import * as ɵhmr0 from '@angular/core'
							// The function body uses ɵhmr0 (or i0 when no rename is needed),
							// so only the hmrImportMgr's declarations make the virtual file valid.
							var allNodes []*ast.Node
							allNodes = append(allNodes, result.HmrImports...)
							allNodes = append(allNodes, result.HmrUpdateNodes...)

							opts := printer.PrinterOptions{}
							p := printer.NewPrinter(opts, printer.PrintHandlers{}, nil)
							writer := printer.NewTextWriter("\n", 4)
							virtualSf := factory.NewSourceFile(
								ast.SourceFileParseOptions{FileName: sf.FileName()},
								sf.Text(),
								factory.NewNodeList(allNodes),
								nil,
							).AsSourceFile()
							p.Write(virtualSf.AsNode(), sf, writer, nil)

							// The TypeScript printer doesn't reliably emit 'export default'
							// from AST modifiers set via AsMutable().SetModifiers().
							// Patch the printed output: replace bare `function <Name>(` with
							// `export default function <Name>(` so Angular's ɵɵreplaceMetadata
							// can import the function as the module's default export.
							hmrCode := writer.String()
							fnPrefix := "function " + className + "_UpdateMetadata("
							if strings.Contains(hmrCode, fnPrefix) && !strings.Contains(hmrCode, "export default "+fnPrefix) {
								hmrCode = strings.Replace(hmrCode, fnPrefix, "export default "+fnPrefix, 1)
							}

							tc.mu.Lock()
							tc.HmrUpdates[key] = hmrCode
							tc.mu.Unlock()
						}
					}
				}
			}

			if len(additionalMembers) > 0 {
				hasTransformedClass = true
				var members []*ast.Node
				members = append(members, classDecl.Members.Nodes...)
				members = append(members, additionalMembers...)
				classDecl.Members.Nodes = members

				// Strip decorators (like @Component) from class modifiers so they are not emitted
				if classDecl.Modifiers() != nil {
					var newMods []*ast.Node
					for _, mod := range classDecl.Modifiers().Nodes {
						if mod.Kind != ast.KindDecorator {
							newMods = append(newMods, mod)
						}
					}
					classDecl.Modifiers().Nodes = newMods
				}

				// Fix missing parent pointers in the newly generated AST nodes
				ast.SetParentInChildren(classDecl.AsNode())
			}

			newStatements = append(newStatements, node)
			if len(postStatements) > 0 {
				newStatements = append(newStatements, postStatements...)
			}
		} else {
			newStatements = append(newStatements, node)
		}
	}

	if hasTransformedClass {
		// Filter removed imports
		var filteredStatements []*ast.Node
		for _, stmt := range newStatements {
			keepStmt := true
			if stmt.Kind == ast.KindImportDeclaration {
				importDecl := stmt.AsImportDeclaration()
				if importDecl.ImportClause != nil && importDecl.ImportClause.AsImportClause().NamedBindings != nil {
					namedBindings := importDecl.ImportClause.AsImportClause().NamedBindings
					if namedBindings.Kind == ast.KindNamedImports {
						namedImports := namedBindings.AsNamedImports()
						var keepElements []*ast.Node
						var moduleSpecifier string
						if importDecl.ModuleSpecifier != nil && importDecl.ModuleSpecifier.Kind == ast.KindStringLiteral {
							moduleSpecifier = importDecl.ModuleSpecifier.AsStringLiteral().Text
							if len(moduleSpecifier) >= 2 && (moduleSpecifier[0] == '"' || moduleSpecifier[0] == '\'') {
								moduleSpecifier = moduleSpecifier[1 : len(moduleSpecifier)-1]
							}
						}

						for _, el := range namedImports.Elements.Nodes {
							if el.Kind == ast.KindImportSpecifier {
								importSpecifier := el.AsImportSpecifier()
								name := importSpecifier.Name().AsIdentifier().Text
								if !importMgr.IsRemoved(sf.AsNode(), name, moduleSpecifier) {
									keepElements = append(keepElements, el)
								}
							} else {
								keepElements = append(keepElements, el)
							}
						}
						if len(keepElements) == 0 {
							keepStmt = false
						} else if len(keepElements) != len(namedImports.Elements.Nodes) {
							namedImports.Elements.Nodes = keepElements
						}
					}
				}
			}
			if keepStmt {
				if stmt.Kind == ast.KindImportDeclaration {
					markDecoratorImportSpecifiers(stmt, decoratorNames)
				}
				filteredStatements = append(filteredStatements, stmt)
			}
		}
		newStatements = filteredStatements

		// Translate constant pool statements
		var poolStatements []*ast.Node
		var generatedImports []*ast.Node
		if len(pool.Statements) > 0 {
			// Pre-seed the import manager with i0 so it knows about it!
			// Actually, if we translate statements that use ExternalExpr("@angular/core", "ɵɵdefineComponent"),
			// the ImportManager will automatically generate `import * as i0 from "@angular/core"`!
			// So we do not need to manually generate i0!

			visitor := translator.NewExpressionTranslatorVisitor(factory, importMgr, sf.AsNode(), translator.TranslatorOptions{})
			for _, stmt := range pool.Statements {
				astStmt := stmt.VisitStatement(visitor, translator.Context{IsStatementMode: true})
				if astStmt != nil {
					if n, ok := astStmt.(*ast.Node); ok {
						poolStatements = append(poolStatements, n)
					}
				}
			}

		}
		generatedImports = importMgr.GetAllImports(sf.AsNode())

		// Ensure import is at the top
		var finalStatements []*ast.Node
		var bodyStatements []*ast.Node
		for _, stmt := range newStatements {
			if stmt.Kind == ast.KindImportDeclaration && len(bodyStatements) == 0 {
				finalStatements = append(finalStatements, stmt)
			} else {
				bodyStatements = append(bodyStatements, stmt)
			}
		}

		finalStatements = append(finalStatements, generatedImports...)
		for _, n := range poolStatements {
			if n != nil {
				finalStatements = append(finalStatements, n)
			}
		}
		finalStatements = append(finalStatements, bodyStatements...)
		sf.Statements.Nodes = finalStatements
		ast.SetParentInChildren(sf.AsNode())
	}
}

func collectDecoratorNames(sf *ast.SourceFile) map[string]bool {
	names := make(map[string]bool)
	if sf == nil || sf.Statements == nil {
		return names
	}
	for _, stmt := range sf.Statements.Nodes {
		if stmt.Kind != ast.KindClassDeclaration {
			continue
		}
		collectDecoratorsFromList(stmt.Decorators(), names)
		classDecl := stmt.AsClassDeclaration()
		if classDecl.Members != nil {
			for _, member := range classDecl.Members.Nodes {
				collectDecoratorsFromList(member.Decorators(), names)
			}
		}
	}
	return names
}

func collectDecoratorsFromList(decorators []*ast.Node, names map[string]bool) {
	for _, decorator := range decorators {
		name := decoratorLocalName(decorator)
		if name != "" {
			names[name] = true
		}
	}
}

func decoratorLocalName(decorator *ast.Node) string {
	if decorator == nil || !ast.IsDecorator(decorator) {
		return ""
	}
	expr := decorator.AsDecorator().Expression
	if expr == nil {
		return ""
	}
	if ast.IsCallExpression(expr) {
		expr = expr.AsCallExpression().Expression
	}
	if ast.IsIdentifier(expr) {
		return expr.AsIdentifier().Text
	}
	if ast.IsPropertyAccessExpression(expr) {
		pa := expr.AsPropertyAccessExpression()
		if pa.Expression != nil && ast.IsIdentifier(pa.Expression) {
			return pa.Expression.AsIdentifier().Text
		}
	}
	return ""
}

func markDecoratorImportSpecifiers(importDeclNode *ast.Node, decoratorNames map[string]bool) {
	if len(decoratorNames) == 0 || importDeclNode == nil || importDeclNode.Kind != ast.KindImportDeclaration {
		return
	}
	importDecl := importDeclNode.AsImportDeclaration()
	if importDecl.ImportClause == nil {
		return
	}
	importClause := importDecl.ImportClause.AsImportClause()
	if importClause.Name() != nil {
		name := importClause.Name().AsIdentifier().Text
		if decoratorNames[name] {
			importClause.Name().Flags |= ast.NodeFlagsAmbient
		}
	}
	namedBindings := importClause.NamedBindings
	if namedBindings == nil {
		return
	}
	if namedBindings.Kind == ast.KindNamespaceImport {
		name := namedBindings.AsNamespaceImport().Name().AsIdentifier().Text
		if decoratorNames[name] {
			namedBindings.Flags |= ast.NodeFlagsAmbient
		}
		return
	}
	if namedBindings.Kind != ast.KindNamedImports {
		return
	}
	for _, element := range namedBindings.AsNamedImports().Elements.Nodes {
		if element.Kind != ast.KindImportSpecifier {
			continue
		}
		importName := element.AsImportSpecifier().Name().AsIdentifier().Text
		if decoratorNames[importName] {
			element.Flags |= ast.NodeFlagsAmbient
		}
	}
}

func (tc *TraitCompiler) GetHmrUpdate(componentId string) string {
	// C2 FIX: Use RLock — HmrUpdates is read-only here. Using exclusive Lock()
	// unnecessarily serialises concurrent HMR requests from the dev server.
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.HmrUpdates[componentId]
}

func (tc *TraitCompiler) GetHmrComponentIds() []string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	ids := make([]string, 0, len(tc.HmrUpdates))
	for id := range tc.HmrUpdates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
