package annotations_local

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// InjectableLocalDecoratorHandler handles @Injectable in Local Compilation mode.
// It explicitly avoids invoking the Type Checker and only uses AST structural analysis.
type InjectableLocalDecoratorHandler struct {
	host reflection.ReflectionHost
}

func NewInjectableLocalDecoratorHandler(host reflection.ReflectionHost) *InjectableLocalDecoratorHandler {
	return &InjectableLocalDecoratorHandler{
		host: host,
	}
}

func (h *InjectableLocalDecoratorHandler) Name() string {
	return "InjectableLocalDecoratorHandler"
}

func (h *InjectableLocalDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	if decorators == nil {
		return nil
	}
	// In local compilation, we only look at the string name of the decorator.
	// Since there is no type checker, we don't strictly verify if it's imported from '@angular/core'.
	// In a real robust implementation, we would write a simple syntax-based AST import resolver here.
	for _, dec := range decorators {
		if dec.Name == "Injectable" {
			d := dec
			return &d
		}
	}
	return nil
}

func (h *InjectableLocalDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &annotations.InjectableAnalysis{
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

		switch name {
		case "providedIn":
			// AST-only extraction of providedIn!
			analysis.ProvidedIn = output.NewWrappedNodeExpr(assign.Initializer, nil, nil, nil)
		}
	}

	return analysis, nil
}

func (h *InjectableLocalDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	// Delegate to the global handler, which is safe since Resolve/Compile run sequentially.
	globalHandler := annotations.NewInjectableDecoratorHandler(h.host)
	return globalHandler.Resolve(node, analysisData)
}

func (h *InjectableLocalDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	// Delegate to the global handler, which is safe since Resolve/Compile run sequentially.
	globalHandler := annotations.NewInjectableDecoratorHandler(h.host)
	return globalHandler.CompileFull(node, analysisData, resolutionData, pool, importMgr, factory)
}

func (h *InjectableLocalDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
}
