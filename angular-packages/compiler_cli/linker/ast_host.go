package linker

import (
	"strconv"

	"github.com/microsoft/typescript-go/internal/ast"
)

type Range struct {
	StartPos  int
	StartLine int
	StartCol  int
	EndPos    int
}

// AstHost is the AST abstraction used by the linker. This mirrors ngtsc's
// generic AstHost, specialized to typescript-go AST nodes.
type AstHost interface {
	GetSymbolName(node *ast.Node) string
	IsStringLiteral(node *ast.Node) bool
	ParseStringLiteral(node *ast.Node) (string, error)
	IsNumericLiteral(node *ast.Node) bool
	ParseNumericLiteral(node *ast.Node) (float64, error)
	IsBooleanLiteral(node *ast.Node) bool
	ParseBooleanLiteral(node *ast.Node) (bool, error)
	IsNull(node *ast.Node) bool
	IsArrayLiteral(node *ast.Node) bool
	ParseArrayLiteral(node *ast.Node) ([]*ast.Node, error)
	IsObjectLiteral(node *ast.Node) bool
	ParseObjectLiteral(node *ast.Node) (map[string]*ast.Node, error)
	IsFunctionExpression(node *ast.Node) bool
	ParseReturnValue(node *ast.Node) (*ast.Node, error)
	ParseParameters(node *ast.Node) ([]*ast.Node, error)
	IsCallExpression(node *ast.Node) bool
	ParseCallee(node *ast.Node) (*ast.Node, error)
	ParseArguments(node *ast.Node) ([]*ast.Node, error)
	GetRange(node *ast.Node) Range
}

type TypeScriptAstHost struct{}

func NewTypeScriptAstHost() *TypeScriptAstHost {
	return &TypeScriptAstHost{}
}

func (h *TypeScriptAstHost) GetSymbolName(node *ast.Node) string {
	if node == nil {
		return ""
	}
	if ast.IsIdentifier(node) {
		return node.Text()
	}
	if ast.IsPropertyAccessExpression(node) {
		return node.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

func (h *TypeScriptAstHost) IsStringLiteral(node *ast.Node) bool {
	return node != nil && (ast.IsStringLiteral(node) || ast.IsNoSubstitutionTemplateLiteral(node))
}

func (h *TypeScriptAstHost) ParseStringLiteral(node *ast.Node) (string, error) {
	if !h.IsStringLiteral(node) {
		return "", linkerError(node, "Expected string literal.")
	}
	return node.Text(), nil
}

func (h *TypeScriptAstHost) IsNumericLiteral(node *ast.Node) bool {
	return node != nil && ast.IsNumericLiteral(node)
}

func (h *TypeScriptAstHost) ParseNumericLiteral(node *ast.Node) (float64, error) {
	if !h.IsNumericLiteral(node) {
		return 0, linkerError(node, "Expected numeric literal.")
	}
	return strconv.ParseFloat(node.Text(), 64)
}

func (h *TypeScriptAstHost) IsBooleanLiteral(node *ast.Node) bool {
	if node == nil {
		return false
	}
	return node.Kind == ast.KindTrueKeyword || node.Kind == ast.KindFalseKeyword
}

func (h *TypeScriptAstHost) ParseBooleanLiteral(node *ast.Node) (bool, error) {
	if node == nil {
		return false, linkerError(node, "Expected boolean literal.")
	}
	switch node.Kind {
	case ast.KindTrueKeyword:
		return true, nil
	case ast.KindFalseKeyword:
		return false, nil
	default:
		return false, linkerError(node, "Expected boolean literal.")
	}
}

func (h *TypeScriptAstHost) IsNull(node *ast.Node) bool {
	return node != nil && node.Kind == ast.KindNullKeyword
}

func (h *TypeScriptAstHost) IsArrayLiteral(node *ast.Node) bool {
	return node != nil && ast.IsArrayLiteralExpression(node)
}

func (h *TypeScriptAstHost) ParseArrayLiteral(node *ast.Node) ([]*ast.Node, error) {
	if !h.IsArrayLiteral(node) {
		return nil, linkerError(node, "Expected array literal.")
	}
	if node.AsArrayLiteralExpression().Elements == nil {
		return nil, nil
	}
	return node.AsArrayLiteralExpression().Elements.Nodes, nil
}

func (h *TypeScriptAstHost) IsObjectLiteral(node *ast.Node) bool {
	return node != nil && ast.IsObjectLiteralExpression(node)
}

func (h *TypeScriptAstHost) ParseObjectLiteral(node *ast.Node) (map[string]*ast.Node, error) {
	if !h.IsObjectLiteral(node) {
		return nil, linkerError(node, "Expected object literal.")
	}
	result := map[string]*ast.Node{}
	props := node.AsObjectLiteralExpression().Properties
	if props == nil {
		return result, nil
	}
	for _, prop := range props.Nodes {
		if !ast.IsPropertyAssignment(prop) {
			continue
		}
		name := prop.Name()
		if name == nil {
			continue
		}
		result[name.Text()] = prop.AsPropertyAssignment().Initializer
	}
	return result, nil
}

func (h *TypeScriptAstHost) IsFunctionExpression(node *ast.Node) bool {
	return node != nil && (ast.IsFunctionExpression(node) || ast.IsArrowFunction(node))
}

func (h *TypeScriptAstHost) ParseReturnValue(node *ast.Node) (*ast.Node, error) {
	if !h.IsFunctionExpression(node) {
		return nil, linkerError(node, "Expected function expression.")
	}
	if node.Body() != nil && !ast.IsBlock(node.Body()) {
		return node.Body(), nil
	}
	if node.Body() != nil && ast.IsBlock(node.Body()) {
		for _, stmt := range node.Body().AsBlock().Statements.Nodes {
			if ast.IsReturnStatement(stmt) {
				return stmt.AsReturnStatement().Expression, nil
			}
		}
	}
	return nil, linkerError(node, "Function does not have a statically extractable return value.")
}

func (h *TypeScriptAstHost) ParseParameters(node *ast.Node) ([]*ast.Node, error) {
	if !h.IsFunctionExpression(node) {
		return nil, linkerError(node, "Expected function expression.")
	}
	if node.ParameterList() == nil {
		return nil, nil
	}
	return node.ParameterList().Nodes, nil
}

func (h *TypeScriptAstHost) IsCallExpression(node *ast.Node) bool {
	return node != nil && ast.IsCallExpression(node)
}

func (h *TypeScriptAstHost) ParseCallee(node *ast.Node) (*ast.Node, error) {
	if !h.IsCallExpression(node) {
		return nil, linkerError(node, "Expected call expression.")
	}
	return node.AsCallExpression().Expression, nil
}

func (h *TypeScriptAstHost) ParseArguments(node *ast.Node) ([]*ast.Node, error) {
	if !h.IsCallExpression(node) {
		return nil, linkerError(node, "Expected call expression.")
	}
	if node.AsCallExpression().Arguments == nil {
		return nil, nil
	}
	return node.AsCallExpression().Arguments.Nodes, nil
}

func (h *TypeScriptAstHost) GetRange(node *ast.Node) Range {
	if node == nil {
		return Range{}
	}
	return Range{StartPos: node.Pos(), EndPos: node.End()}
}
