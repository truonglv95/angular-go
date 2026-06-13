package annotations_local

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

type NgModuleLocalDecoratorHandler struct {
	host          reflection.ReflectionHost
	metaRegistry  *metadata.LocalMetadataRegistry
	scopeRegistry *scope.LocalModuleScopeRegistry
}

func NewNgModuleLocalDecoratorHandler(host reflection.ReflectionHost, metaRegistry *metadata.LocalMetadataRegistry, scopeRegistry *scope.LocalModuleScopeRegistry) *NgModuleLocalDecoratorHandler {
	return &NgModuleLocalDecoratorHandler{
		host:          host,
		metaRegistry:  metaRegistry,
		scopeRegistry: scopeRegistry,
	}
}

func (h *NgModuleLocalDecoratorHandler) Name() string {
	return "NgModuleLocalDecoratorHandler"
}

func (h *NgModuleLocalDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	if decorators == nil {
		return nil
	}
	for _, dec := range decorators {
		if dec.Name == "NgModule" {
			d := dec
			return &d
		}
	}
	return nil
}

func extractR3References(expr *ast.Expression) []render3.R3Reference {
	var refs []render3.R3Reference
	if expr.Kind == ast.KindArrayLiteralExpression {
		arr := expr.AsArrayLiteralExpression()
		if arr.Elements != nil {
			for _, elem := range arr.Elements.Nodes {
				if elem.Kind == ast.KindIdentifier {
					ident := elem.AsIdentifier().Text
					refs = append(refs, render3.R3Reference{
						Value: output.NewReadVarExpr(ident, nil, nil, nil),
					})
				} else if baseIdent := getModuleScopeReferenceIdentifier(elem); baseIdent != nil {
					ident := baseIdent.AsIdentifier().Text
					refs = append(refs, render3.R3Reference{
						Value: output.NewReadVarExpr(ident, nil, nil, nil),
					})
				} else {
					refs = append(refs, render3.R3Reference{
						Value: output.NewWrappedNodeExpr(elem, nil, nil, nil),
					})
				}
			}
		}
	}
	return refs
}

func extractASTNodes(expr *ast.Expression) []*ast.Node {
	var nodes []*ast.Node
	if expr == nil {
		return nodes
	}
	if expr.Kind == ast.KindArrayLiteralExpression {
		arr := expr.AsArrayLiteralExpression()
		if arr.Elements != nil {
			for _, elem := range arr.Elements.Nodes {
				nodes = append(nodes, elem)
			}
		}
		return nodes
	}
	return append(nodes, expr.AsNode())
}

func getModuleScopeReferenceIdentifier(node *ast.Node) *ast.Node {
	if node == nil || node.Kind != ast.KindCallExpression {
		return nil
	}
	return getBaseIdentifier(node)
}

func getBaseIdentifier(node *ast.Node) *ast.Node {
	if node == nil {
		return nil
	}
	if node.Kind == ast.KindIdentifier {
		return node
	}
	if node.Kind == ast.KindCallExpression {
		callExpr := node.AsCallExpression()
		if callExpr.Expression != nil {
			if callExpr.Expression.Kind == ast.KindPropertyAccessExpression {
				return getBaseIdentifier(callExpr.Expression.AsPropertyAccessExpression().Expression)
			}
			return getBaseIdentifier(callExpr.Expression)
		}
	}
	if node.Kind == ast.KindPropertyAccessExpression {
		return getBaseIdentifier(node.AsPropertyAccessExpression().Expression)
	}
	return nil
}

func topLevelClassByName(sf *ast.SourceFile) map[string]*ast.Node {
	classes := make(map[string]*ast.Node)
	if sf == nil {
		return classes
	}
	for _, stmt := range sf.Statements.Nodes {
		if stmt.Kind != ast.KindClassDeclaration {
			continue
		}
		classDecl := stmt.AsClassDeclaration()
		if classDecl.Name() != nil && classDecl.Name().Kind == ast.KindIdentifier {
			classes[classDecl.Name().AsIdentifier().Text] = stmt
		}
	}
	return classes
}

func (h *NgModuleLocalDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &annotations.NgModuleAnalysis{
		DecoratorNode: decorator.Node,
	}

	if len(decorator.Args) == 0 {
		return analysis, nil
	}

	arg := decorator.Args[0]
	if arg.Kind != ast.KindObjectLiteralExpression {
		return analysis, nil
	}

	obj := arg.AsObjectLiteralExpression()
	if obj.Properties == nil {
		return analysis, nil
	}

	for _, prop := range obj.Properties.Nodes {
		if prop.Kind != ast.KindPropertyAssignment {
			continue
		}

		assign := prop.AsPropertyAssignment()
		name := ""
		if assign.Name().Kind == ast.KindIdentifier {
			name = assign.Name().AsIdentifier().Text
		}

		initExpr := assign.Initializer

		switch name {
		case "declarations":
			analysis.Declarations = extractR3References(initExpr)
			analysis.DeclarationNodes = extractASTNodes(initExpr)
		case "imports":
			analysis.Imports = extractR3References(initExpr)
			analysis.ImportNodes = extractASTNodes(initExpr)
			analysis.ImportsExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		case "exports":
			analysis.Exports = extractR3References(initExpr)
			analysis.ExportNodes = extractASTNodes(initExpr)
		case "bootstrap":
			analysis.Bootstrap = extractR3References(initExpr)
			analysis.BootstrapNodes = extractASTNodes(initExpr)
		case "providers":
			analysis.ProvidersExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		}
	}

	// Register module metadata
	if h.metaRegistry != nil {
		classes := topLevelClassByName(ast.GetSourceFileOfNode(node.AsNode()))
		resolveRefNode := func(elem *ast.Node) metadata.Reference {
			if elem == nil {
				return metadata.Reference{}
			}
			baseIdent := getBaseIdentifier(elem)
			if baseIdent != nil {
				decl := h.host.GetDeclarationOfIdentifier(baseIdent)
				if decl != nil && decl.Node != nil {
					return metadata.Reference{
						Name:         baseIdent.AsIdentifier().Text,
						Node:         decl.Node,
						OwningModule: decl.ViaModule,
					}
				}
				name := baseIdent.AsIdentifier().Text
				if cNode, ok := classes[name]; ok {
					return metadata.Reference{Name: name, Node: cNode}
				}
			}
			return metadata.Reference{}
		}

		var decls, imps, exps []metadata.Reference
		for _, d := range analysis.DeclarationNodes {
			decls = append(decls, resolveRefNode(d))
		}
		for _, i := range analysis.ImportNodes {
			imps = append(imps, resolveRefNode(i))
		}
		for _, e := range analysis.ExportNodes {
			exps = append(exps, resolveRefNode(e))
		}
		h.metaRegistry.RegisterNgModule(node.AsNode(), &metadata.NgModuleMeta{
			Name:         node.Name().AsIdentifier().Text,
			Ref:          metadata.Reference{Name: node.Name().AsIdentifier().Text, Node: node.AsNode()},
			Declarations: decls,
			Imports:      imps,
			Exports:      exps,
		})

		if h.scopeRegistry != nil {
			for _, ref := range decls {
				if ref.Node != nil {
					h.scopeRegistry.RegisterComponentDeclaration(ref.Node, node.AsNode())
				}
			}
		}
	}

	return analysis, nil
}

func (h *NgModuleLocalDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	// Local compilation cannot reliably validate imported/exported NgModule
	// references because there is no type checker or d.ts metadata reader.
	// ngtsc still emits ɵmod/ɵinj in this mode and leaves cross-file semantics to
	// runtime/linking, so keep resolve non-blocking here.
	return &annotations.NgModuleResolution{}, nil
}

func (h *NgModuleLocalDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	globalHandler := annotations.NewNgModuleDecoratorHandler(h.host, h.metaRegistry, h.scopeRegistry)
	return globalHandler.CompileFull(node, analysisData, resolutionData, pool, importMgr, factory)
}

func (h *NgModuleLocalDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
	analysis := analysisData.(*annotations.NgModuleAnalysis)
	globalHandler := annotations.NewNgModuleDecoratorHandler(h.host, h.metaRegistry, h.scopeRegistry)
	globalHandler.Register(node, analysis)
}
