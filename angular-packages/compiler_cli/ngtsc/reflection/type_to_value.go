package reflection

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

// TypeToValue converts a TypeNode to a TypeValueReference that indicates how to use
// the type in a value position.
// Returns an UNAVAILABLE reference if it cannot be statically resolved.
func TypeToValue(typeNode *ast.Node, chk *checker.Checker, isLocalCompilation bool) TypeValueReference {
	if typeNode == nil {
		return missingType()
	}

	if !ast.IsTypeReferenceNode(typeNode) {
		return unsupportedType(typeNode)
	}

	symbols := resolveTypeSymbols(typeNode, chk)
	if symbols == nil {
		return unknownReference(typeNode)
	}

	local := symbols.local
	decl := symbols.decl

	// Check that the declaration has a value component (not just a type).
	if decl.ValueDeclaration == nil || decl.Flags&ast.SymbolFlagsConstEnum != 0 {
		var typeOnlyDecl *ast.Node
		if len(decl.Declarations) > 0 {
			typeOnlyDecl = decl.Declarations[0]
		}

		if !isLocalCompilation || (typeOnlyDecl != nil && isTypeOnlyKind(typeOnlyDecl.Kind)) {
			return noValueDeclaration(typeNode, typeOnlyDecl)
		}
	}

	// The type has a value declaration. Try to rewrite the type reference into a value expression.
	var firstDecl *ast.Node
	if len(local.Declarations) > 0 {
		firstDecl = local.Declarations[0]
	}

	if firstDecl != nil {
		if ast.IsImportClause(firstDecl) {
			clause := firstDecl.AsImportClause()
			if clause.Name() != nil {
				// default import: import Foo from 'foo'
				if clause.PhaseModifier == ast.KindTypeKeyword {
					return typeOnlyImport(typeNode, firstDecl)
				}
				if !ast.IsImportDeclaration(firstDecl.Parent) {
					return unsupportedType(typeNode)
				}
				return LocalTypeValueRef{
					Expression:            clause.Name(),
					DefaultImportStatement: firstDecl.Parent,
				}
			}
		} else if ast.IsImportSpecifier(firstDecl) {
			spec := firstDecl.AsImportSpecifier()
			if spec.IsTypeOnly {
				return typeOnlyImport(typeNode, firstDecl)
			}
			// check parent ImportClause phase modifier
			if ast.IsImportClause(firstDecl.Parent) {
				parentClause := firstDecl.Parent.AsImportClause()
				if parentClause.PhaseModifier == ast.KindTypeKeyword {
					return typeOnlyImport(typeNode, firstDecl.Parent)
				}
			}
			// Determine exported name: use propertyName if it's an aliased import.
			var importedName string
			if spec.PropertyName != nil {
				importedName = spec.PropertyName.Text()
			} else {
				importedName = spec.Name().Text()
			}

			// Build nested path: skip the first symbol name (local alias), take the rest.
			var nestedPath []string
			if len(symbols.symbolNames) > 1 {
				nestedPath = symbols.symbolNames[1:]
			}

			// Walk up to ImportDeclaration: spec → NamedImports → ImportClause → ImportDeclaration
			importDecl := firstDecl.Parent.Parent.Parent
			if !ast.IsImportDeclaration(importDecl) {
				return unsupportedType(typeNode)
			}
			moduleName := extractModuleName(importDecl)
			return ImportedTypeValueRef{
				ModuleName:      moduleName,
				ImportedName:    importedName,
				NestedPath:      nestedPath,
				ValueDeclaration: decl.ValueDeclaration,
			}
		} else if ast.IsNamespaceImport(firstDecl) {
			// import * as Foo from 'foo'
			if ast.IsImportClause(firstDecl.Parent) {
				parentClause := firstDecl.Parent.AsImportClause()
				if parentClause.PhaseModifier == ast.KindTypeKeyword {
					return typeOnlyImport(typeNode, firstDecl.Parent)
				}
			}

			if len(symbols.symbolNames) == 1 {
				// Reference to the namespace itself
				return namespaceImport(typeNode, firstDecl.Parent)
			}

			// Skip namespace name, use next as importedName, rest as nestedPath.
			var importedName string
			var nestedPath []string
			if len(symbols.symbolNames) >= 2 {
				importedName = symbols.symbolNames[1]
				if len(symbols.symbolNames) > 2 {
					nestedPath = symbols.symbolNames[2:]
				}
			}

			// firstDecl (NamespaceImport) → ImportClause → ImportDeclaration
			importDecl := firstDecl.Parent.Parent
			if !ast.IsImportDeclaration(importDecl) {
				return unsupportedType(typeNode)
			}
			moduleName := extractModuleName(importDecl)
			return ImportedTypeValueRef{
				ModuleName:      moduleName,
				ImportedName:    importedName,
				NestedPath:      nestedPath,
				ValueDeclaration: decl.ValueDeclaration,
			}
		}
	}

	// Not imported; convert type reference to expression directly.
	expr := typeNodeToValueExpr(typeNode)
	if expr != nil {
		return LocalTypeValueRef{
			Expression:            expr,
			DefaultImportStatement: nil,
		}
	}
	return unsupportedType(typeNode)
}

// typeSymbolsResult holds the symbols and names from resolveTypeSymbols.
type typeSymbolsResult struct {
	local       *ast.Symbol
	decl        *ast.Symbol
	symbolNames []string
}

// resolveTypeSymbols resolves the symbols behind a TypeReferenceNode.
func resolveTypeSymbols(typeRef *ast.Node, chk *checker.Checker) *typeSymbolsResult {
	typeRefNode := typeRef.AsTypeReferenceNode()
	typeName := typeRefNode.TypeName

	typeRefSymbol := chk.GetSymbolAtLocation(typeName)
	if typeRefSymbol == nil {
		return nil
	}

	local := typeRefSymbol

	// Destructure qualified names like foo.X.Y.Z
	leftMost := typeName
	var symbolNames []string
	for ast.IsQualifiedName(leftMost) {
		qn := leftMost.AsQualifiedName()
		symbolNames = append([]string{qn.Right.Text()}, symbolNames...)
		leftMost = qn.Left
	}
	symbolNames = append([]string{leftMost.AsIdentifier().Text}, symbolNames...)

	if leftMost != typeName {
		localTmp := chk.GetSymbolAtLocation(leftMost)
		if localTmp != nil {
			local = localTmp
		}
	}

	decl := typeRefSymbol
	if decl.Flags&ast.SymbolFlagsAlias != 0 {
		decl = chk.GetAliasedSymbol(decl)
	}

	return &typeSymbolsResult{local: local, decl: decl, symbolNames: symbolNames}
}

// TypeNodeToValueExpr attempts to convert a TypeNode to an equivalent Expression.
func TypeNodeToValueExpr(typeNode *ast.Node) *ast.Node {
	if ast.IsTypeReferenceNode(typeNode) {
		return EntityNameToValue(typeNode.AsTypeReferenceNode().TypeName)
	}
	return nil
}

// typeNodeToValueExpr is the internal version of TypeNodeToValueExpr.
func typeNodeToValueExpr(typeNode *ast.Node) *ast.Node {
	return TypeNodeToValueExpr(typeNode)
}

// EntityNameToValue converts an entity name (qualified or simple identifier) to an expression.
func EntityNameToValue(node *ast.Node) *ast.Node {
	if ast.IsQualifiedName(node) {
		qn := node.AsQualifiedName()
		left := EntityNameToValue(qn.Left)
		if left == nil {
			return nil
		}
		// Return the right identifier as a property access expression would be
		// built by a factory. For our purposes we return the right node directly
		// as the caller uses this for import resolution, not code generation.
		return qn.Right
	} else if ast.IsIdentifier(node) {
		return node
	}
	return nil
}

func extractModuleName(importDecl *ast.Node) string {
	moduleSpecifier := importDecl.AsImportDeclaration().ModuleSpecifier
	if moduleSpecifier != nil && ast.IsStringLiteral(moduleSpecifier) {
		return moduleSpecifier.AsStringLiteral().Text
	}
	return ""
}

// isTypeOnlyKind returns true if the given kind represents a type-only declaration.
func isTypeOnlyKind(kind ast.Kind) bool {
	switch kind {
	case ast.KindTypeParameter, ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration:
		return true
	}
	return false
}

// --- Unavailable reason constructors ---

func missingType() TypeValueReference {
	return UnavailableTypeValueRef{Kind: MissingTypeKind}
}

func unsupportedType(node *ast.Node) TypeValueReference {
	return UnavailableTypeValueRef{Kind: UnsupportedKind, Node: node}
}

func noValueDeclaration(typeNode *ast.Node, decl *ast.Node) TypeValueReference {
	return UnavailableTypeValueRef{Kind: NoValueDeclarationKind, Node: typeNode, ExtraNode: decl}
}

func typeOnlyImport(typeNode *ast.Node, clauseOrSpecifier *ast.Node) TypeValueReference {
	return UnavailableTypeValueRef{Kind: TypeOnlyImportKind, Node: typeNode, ExtraNode: clauseOrSpecifier}
}

func unknownReference(typeNode *ast.Node) TypeValueReference {
	return UnavailableTypeValueRef{Kind: UnknownReferenceKind, Node: typeNode}
}

func namespaceImport(typeNode *ast.Node, importClause *ast.Node) TypeValueReference {
	return UnavailableTypeValueRef{Kind: NamespaceKind, Node: typeNode, ExtraNode: importClause}
}

// --- AmbientImport sentinel ---

// AmbientImport is a sentinel value used in Declaration.ViaModule when the
// declaration comes from a different source file via an ambient module.
const AmbientImport = "\x00ambient"

// IsAbsoluteModuleName returns true if the module name is not a relative path.
func IsAbsoluteModuleName(name string) bool {
	return !strings.HasPrefix(name, ".")
}
