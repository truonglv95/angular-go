package annotations

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/internal/ast"
)

// parseSignalInput checks if the initializer is a call to input() or input.required()
// and extracts the corresponding R3InputMetadata.
func parseSignalInput(initializer *ast.Node, propName string) (render3.R3InputMetadata, bool) {
	if initializer == nil || initializer.Kind != ast.KindCallExpression {
		return render3.R3InputMetadata{}, false
	}
	callExpr := initializer.AsCallExpression()
	expr := callExpr.Expression
	isSignalInput := false
	isRequired := false
	alias := propName

	if expr.Kind == ast.KindIdentifier && expr.AsIdentifier().Text == "input" {
		isSignalInput = true
	} else if expr.Kind == ast.KindPropertyAccessExpression {
		pae := expr.AsPropertyAccessExpression()
		if pae.Expression.Kind == ast.KindIdentifier && pae.Expression.AsIdentifier().Text == "input" {
			if pae.Name() != nil && pae.Name().AsIdentifier().Text == "required" {
				isSignalInput = true
				isRequired = true
			}
		}
	}

	if !isSignalInput {
		return render3.R3InputMetadata{}, false
	}

	// Parse alias from options:
	// For input.required(options?), options is at index 0
	// For input(initialValue?, options?), options is at index 1
	var optionsNode *ast.Node
	if isRequired {
		if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
			optionsNode = callExpr.Arguments.Nodes[0]
		}
	} else {
		if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 1 {
			optionsNode = callExpr.Arguments.Nodes[1]
		}
	}

	if optionsNode != nil && optionsNode.Kind == ast.KindObjectLiteralExpression {
		obj := optionsNode.AsObjectLiteralExpression()
		if obj.Properties != nil {
			for _, prop := range obj.Properties.Nodes {
				if prop.Kind == ast.KindPropertyAssignment {
					pa := prop.AsPropertyAssignment()
					if pa.Name() != nil && pa.Name().Kind == ast.KindIdentifier && pa.Name().AsIdentifier().Text == "alias" {
						if pa.Initializer != nil && pa.Initializer.Kind == ast.KindStringLiteral {
							alias = pa.Initializer.AsStringLiteral().Text
						}
					}
				}
			}
		}
	}

	return render3.R3InputMetadata{
		ClassPropertyName:   propName,
		BindingPropertyName: alias,
		Required:            isRequired,
		IsSignal:            true,
	}, true
}
