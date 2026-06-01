package transform

import (
	"fmt"
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
)

// TraitCompiler là bộ điều phối trung tâm của ngtsc. Nó quản lý tất cả các class
// trong chương trình, chạy qua các bước Analyze -> Resolve -> Compile.
type TraitCompiler struct {
	handlers []DecoratorHandler
	host     reflection.ReflectionHost
	
	// classes lưu trữ danh sách các Traits cho mỗi class declaration.
	classes map[*ast.ClassDeclaration][]*Trait
}

func NewTraitCompiler(handlers []DecoratorHandler, host reflection.ReflectionHost) *TraitCompiler {
	return &TraitCompiler{
		handlers: handlers,
		host:     host,
		classes:  make(map[*ast.ClassDeclaration][]*Trait),
	}
}

// AnalyzeSync quét qua một source file, phát hiện các class và chạy hàm Analyze của các Handler.
func (tc *TraitCompiler) AnalyzeSync(sf *ast.SourceFile) {
	if sf == nil || sf.Statements == nil {
		return
	}
	
	sf.AsNode().ForEachChild(func(node *ast.Node) bool {
		if node.Kind == ast.KindClassDeclaration {
			classDecl := node.AsClassDeclaration()
			tc.analyzeClass(classDecl)
		}
		return false
	})
}

func (tc *TraitCompiler) analyzeClass(classDecl *ast.ClassDeclaration) {
	decorators := tc.host.GetDecoratorsOfDeclaration(classDecl.AsNode())
	if len(decorators) == 0 {
		return
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
		return
	}
	
	tc.classes[classDecl] = traits
	
	// 2. Analyze
	for _, trait := range traits {
		analysis, _ := trait.Handler.Analyze(classDecl, trait.Decorator)
		trait.Analysis = analysis
		trait.State = TraitStateAnalyzed
	}
}

// Resolve chạy bước giải quyết tham chiếu cho tất cả các traits đã được Analyze.
func (tc *TraitCompiler) Resolve() {
	for classDecl, traits := range tc.classes {
		for _, trait := range traits {
			if trait.State == TraitStateAnalyzed {
				resolution, _ := trait.Handler.Resolve(classDecl, trait.Analysis)
				trait.Resolution = resolution
				trait.State = TraitStateResolved
			}
		}
	}
}

// UpdateSourceFile áp dụng kết quả compile (CompileResult) vào AST của source file.
func (tc *TraitCompiler) UpdateSourceFile(sf *ast.SourceFile, factory *ast.NodeFactory) {
	if sf == nil || sf.Statements == nil {
		return
	}
	
	pool := compiler.NewConstantPool(false)
	config := imports.PresetImportManagerForceNamespaceImports
	importMgr := imports.NewImportManager(&config, factory)
	var hasTransformedClass bool
	var newStatements []*ast.Node
	
	for _, node := range sf.Statements.Nodes {
		if node.Kind == ast.KindClassDeclaration {
			classDecl := node.AsClassDeclaration()
			traits := tc.classes[classDecl]
			
			var additionalMembers []*ast.Node
			var postStatements []*ast.Node
			for _, trait := range traits {
				if trait.State == TraitStateResolved {
					results, _ := trait.Handler.CompileFull(classDecl, trait.Analysis, trait.Resolution, pool, importMgr, factory)
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
		// Translate constant pool statements
		var poolStatements []*ast.Node
		var generatedImports []*ast.Node
		if len(pool.Statements) > 0 {
			// Pre-seed the import manager with i0 so it knows about it!
			// Actually, if we translate statements that use ExternalExpr("@angular/core", "ɵɵdefineComponent"),
			// the ImportManager will automatically generate `import * as i0 from "@angular/core"`!
			// So we do not need to manually generate i0!
			
			fmt.Printf("Visitor sf.AsNode()=%p\n", sf.AsNode())
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
