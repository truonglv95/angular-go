package annotations

import (
	"fmt"

	ngtscdiag "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/internal/ast"
)

// CreateValueHasWrongTypeError creates a FatalDiagnosticError for a node that did not evaluate to the expected type.
func CreateValueHasWrongTypeError(node *ast.Node, value partial_evaluator.ResolvedValue, messageText string) *ngtscdiag.FatalDiagnosticError {
	var chainedMessage string
	var relatedInformation []*ast.Diagnostic

	switch v := value.(type) {
	case *partial_evaluator.DynamicValue:
		chainedMessage = "Value could not be determined statically."
		relatedInformation = partial_evaluator.TraceDynamicValue(node, v)
	case *imports.Reference:
		target := "an anonymous declaration"
		if v.DebugName() != "" {
			target = fmt.Sprintf("'%s'", v.DebugName())
		}
		chainedMessage = fmt.Sprintf("Value is a reference to %s.", target)

		refNode := v.Node
		if name := identifierOfNode(v.Node); name != nil {
			refNode = name
		}
		relatedInformation = []*ast.Diagnostic{
			ngtscdiag.MakeRelatedInformation(refNode, "Reference is declared here."),
		}
	default:
		chainedMessage = fmt.Sprintf("Value is of type '%s'.", partial_evaluator.DescribeResolvedType(v, 1))
	}

	chain := ngtscdiag.MakeDiagnosticChain(messageText, []*ast.Diagnostic{
		ngtscdiag.MakeDiagnosticChain(chainedMessage, nil),
	})

	return ngtscdiag.NewFatalDiagnosticError(
		ngtscdiag.ErrorCode_VALUE_HAS_WRONG_TYPE,
		node,
		chain,
		relatedInformation,
	)
}

func identifierOfNode(node *ast.Node) *ast.Node {
	if node == nil {
		return nil
	}
	if ast.IsIdentifier(node) {
		return node
	}
	if node.Kind == ast.KindClassDeclaration {
		return node.AsClassDeclaration().Name()
	}
	if node.Kind == ast.KindFunctionDeclaration {
		return node.AsFunctionDeclaration().Name()
	}
	if node.Kind == ast.KindVariableDeclaration {
		return node.AsVariableDeclaration().Name()
	}
	return nil
}
