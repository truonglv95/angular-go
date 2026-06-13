package annotations

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/internal/ast"
)

func extractDITokenFromNode(node *ast.Node) output.Expression {
	if node == nil {
		return output.NewLiteralExpr(nil, nil, nil, nil)
	}
	if ast.IsStringLiteral(node) {
		return output.NewLiteralExpr(node.AsStringLiteral().Text, nil, nil, nil)
	}
	if ast.IsIdentifier(node) {
		return output.NewReadVarExpr(node.AsIdentifier().Text, nil, nil, nil)
	}
	if ast.IsPropertyAccessExpression(node) {
		pa := node.AsPropertyAccessExpression()
		recv := extractDITokenFromNode(pa.Expression)
		if pa.Name().Kind == ast.KindIdentifier {
			return output.NewReadPropExpr(recv, pa.Name().AsIdentifier().Text, nil, nil, nil, false)
		}
	}
	if ast.IsParenthesizedExpression(node) {
		return extractDITokenFromNode(node.AsParenthesizedExpression().Expression)
	}
	if ast.IsAsExpression(node) {
		return extractDITokenFromNode(node.AsAsExpression().Expression)
	}
	if ast.IsSatisfiesExpression(node) {
		return extractDITokenFromNode(node.AsSatisfiesExpression().Expression)
	}
	if ast.IsNonNullExpression(node) {
		return extractDITokenFromNode(node.AsNonNullExpression().Expression)
	}
	if ast.IsCallExpression(node) {
		fn := extractDITokenFromNode(node.Expression())
		args := make([]output.Expression, 0, len(node.Arguments()))
		for _, arg := range node.Arguments() {
			args = append(args, extractDITokenFromNode(arg))
		}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	}
	if ast.IsArrowFunction(node) {
		return extractFunctionLikeTokenBody(node.Body())
	}
	if ast.IsFunctionExpression(node) {
		return extractFunctionLikeTokenBody(node.AsFunctionExpression().Body)
	}
	return output.NewLiteralExpr("UNKNOWN_TOKEN", nil, nil, nil)
}

func extractFunctionLikeTokenBody(body *ast.Node) output.Expression {
	if body == nil {
		return output.NewFunctionExpr(nil, nil, nil, nil, nil, nil)
	}
	if ast.IsBlock(body) {
		block := body.AsBlock()
		if block.Statements != nil {
			for _, stmt := range block.Statements.Nodes {
				if ast.IsReturnStatement(stmt) {
					returnExpr := stmt.AsReturnStatement().Expression
					if returnExpr != nil {
						return output.NewFunctionExpr(nil, []output.Statement{
							output.NewReturnStatement(extractDITokenFromNode(returnExpr), nil, nil),
						}, nil, nil, nil, nil)
					}
				}
			}
		}
		return output.NewFunctionExpr(nil, nil, nil, nil, nil, nil)
	}
	return output.NewFunctionExpr(nil, []output.Statement{
		output.NewReturnStatement(extractDITokenFromNode(body), nil, nil),
	}, nil, nil, nil, nil)
}
