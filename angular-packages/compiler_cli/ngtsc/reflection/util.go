package reflection

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// IsNamedClassDeclaration checks if a node is a class declaration with a name.
// Anonymous default exports (`export default class { ... }`) are not considered named.
func IsNamedClassDeclaration(node *ast.Node) bool {
	if !ast.IsClassDeclaration(node) {
		return false
	}
	return node.AsClassDeclaration().Name() != nil
}

// IsNamedFunctionDeclaration checks if a node is a function declaration with a name.
func IsNamedFunctionDeclaration(node *ast.Node) bool {
	if !ast.IsFunctionDeclaration(node) {
		return false
	}
	return node.Name() != nil
}

// IsNamedVariableDeclaration checks if a node is a variable declaration with an identifier name.
func IsNamedVariableDeclaration(node *ast.Node) bool {
	if !ast.IsVariableDeclaration(node) {
		return false
	}
	name := node.AsVariableDeclaration().Name()
	return name != nil && ast.IsIdentifier(name)
}

// ClassMemberAccessLevelToString converts a ClassMemberAccessLevel to its string representation.
func ClassMemberAccessLevelToString(level ClassMemberAccessLevel) string {
	switch level {
	case PublicWritable:
		return "public"
	case PublicReadonly:
		return "public readonly"
	case Protected:
		return "protected"
	case Private:
		return "private"
	case EcmaScriptPrivate:
		return "#private"
	default:
		return "unknown"
	}
}
