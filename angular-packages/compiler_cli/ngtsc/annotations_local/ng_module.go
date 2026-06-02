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
		case "imports":
			analysis.Imports = extractR3References(initExpr)
			analysis.ImportsExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		case "exports":
			analysis.Exports = extractR3References(initExpr)
		case "bootstrap":
			analysis.Bootstrap = extractR3References(initExpr)
		case "providers":
			analysis.ProvidersExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		}
	}

	// Register module metadata
	if h.metaRegistry != nil {
		var decls, imps, exps []metadata.Reference
		for _, d := range analysis.Declarations {
			name := ""
			if rv, ok := d.Value.(*output.ReadVarExpr); ok {
				name = rv.Name
			}
			decls = append(decls, metadata.Reference{Name: name})
		}
		for _, i := range analysis.Imports {
			name := ""
			if rv, ok := i.Value.(*output.ReadVarExpr); ok {
				name = rv.Name
			}
			imps = append(imps, metadata.Reference{Name: name})
		}
		for _, e := range analysis.Exports {
			name := ""
			if rv, ok := e.Value.(*output.ReadVarExpr); ok {
				name = rv.Name
			}
			exps = append(exps, metadata.Reference{Name: name})
		}
		h.metaRegistry.RegisterNgModule(node.AsNode(), &metadata.NgModuleMeta{
			Name:         node.Name().AsIdentifier().Text,
			Declarations: decls,
			Imports:      imps,
			Exports:      exps,
		})
	}

	return analysis, nil
}

func (h *NgModuleLocalDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	localAnalysis := analysisData.(*annotations.NgModuleAnalysis)
	globalHandler := annotations.NewNgModuleDecoratorHandler(h.host, h.metaRegistry, h.scopeRegistry)
	return globalHandler.Resolve(node, localAnalysis)
}

func (h *NgModuleLocalDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	globalHandler := annotations.NewNgModuleDecoratorHandler(h.host, h.metaRegistry, h.scopeRegistry)
	return globalHandler.CompileFull(node, analysisData, resolutionData, pool, importMgr, factory)
}
