package annotations_local

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/internal/ast"
)

func parseLocalSignalInput(initializer *ast.Node, propName string) (render3.R3InputMetadata, bool) {
	return parseLocalInitializerInput(initializer, propName, "input")
}

func parseLocalModelInput(initializer *ast.Node, propName string) (render3.R3InputMetadata, bool) {
	return parseLocalInitializerInput(initializer, propName, "model")
}

func parseLocalInitializerInput(initializer *ast.Node, propName string, apiName string) (render3.R3InputMetadata, bool) {
	if initializer == nil || initializer.Kind != ast.KindCallExpression {
		return render3.R3InputMetadata{}, false
	}

	callExpr := initializer.AsCallExpression()
	expr := callExpr.Expression
	isInitializerApi := false
	isRequired := false
	alias := propName

	if expr.Kind == ast.KindIdentifier && expr.AsIdentifier().Text == apiName {
		isInitializerApi = true
	} else if expr.Kind == ast.KindPropertyAccessExpression {
		pae := expr.AsPropertyAccessExpression()
		if pae.Expression.Kind == ast.KindIdentifier && pae.Expression.AsIdentifier().Text == apiName {
			if pae.Name() != nil && pae.Name().Kind == ast.KindIdentifier && pae.Name().AsIdentifier().Text == "required" {
				isInitializerApi = true
				isRequired = true
			}
		}
	}

	if !isInitializerApi {
		return render3.R3InputMetadata{}, false
	}

	var optionsNode *ast.Node
	if isRequired {
		if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
			optionsNode = callExpr.Arguments.Nodes[0]
		}
	} else if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 1 {
		optionsNode = callExpr.Arguments.Nodes[1]
	}

	if optionsNode != nil && optionsNode.Kind == ast.KindObjectLiteralExpression {
		obj := optionsNode.AsObjectLiteralExpression()
		if obj.Properties != nil {
			for _, prop := range obj.Properties.Nodes {
				if prop.Kind != ast.KindPropertyAssignment {
					continue
				}
				pa := prop.AsPropertyAssignment()
				if pa.Name() != nil && pa.Name().Kind == ast.KindIdentifier && pa.Name().AsIdentifier().Text == "alias" {
					if pa.Initializer != nil && pa.Initializer.Kind == ast.KindStringLiteral {
						alias = pa.Initializer.AsStringLiteral().Text
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
