package component

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/internal/ast"
)

// LegacyAnimationTriggerResolver resolves Angular animation 'trigger()' calls.
func legacyAnimationTriggerResolver(fn *ast.Node, callExpr *ast.CallExpression, resolve func(expr *ast.Node) partial_evaluator.ResolvedValue, unresolvable *partial_evaluator.DynamicValue) partial_evaluator.ResolvedValue {
	if fn.Kind == ast.KindIdentifier {
		ident := fn.AsIdentifier()
		if ident.Text == "trigger" && callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
			res := make(partial_evaluator.ResolvedValueMap)
			res["name"] = resolve(callExpr.Arguments.Nodes[0])
			return res
		}
	} else if fn.Kind == ast.KindPropertyAccessExpression {
		prop := fn.AsPropertyAccessExpression()
		if prop.Name().Kind == ast.KindIdentifier && prop.Name().AsIdentifier().Text == "trigger" && callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
			res := make(partial_evaluator.ResolvedValueMap)
			res["name"] = resolve(callExpr.Arguments.Nodes[0])
			return res
		}
	}
	return unresolvable
}
