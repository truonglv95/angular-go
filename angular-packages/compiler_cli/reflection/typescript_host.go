package reflection

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

// TypeScriptReflectionHost implements ReflectionHost using the TypeScript type-checker
// and AST (via the typescript-go AST package).
type TypeScriptReflectionHost struct {
	checker *checker.Checker
}

// NewTypeScriptReflectionHost creates a new TypeScriptReflectionHost.
// checker may be nil for cases that only use structural AST traversal
// (e.g., GetDecoratorsOfDeclaration which doesn't need type resolution).
func NewTypeScriptReflectionHost(chk *checker.Checker) *TypeScriptReflectionHost {
	return &TypeScriptReflectionHost{checker: chk}
}

// GetDecoratorsOfDeclaration returns the decorators applied to an AST declaration node.
func (h *TypeScriptReflectionHost) GetDecoratorsOfDeclaration(declaration *ast.Node) []Decorator {
	if declaration == nil {
		return nil
	}
	decoratorNodes := declaration.Decorators()
	if len(decoratorNodes) == 0 {
		return nil
	}
	var result []Decorator
	for _, decoratorNode := range decoratorNodes {
		dec := h.reflectDecorator(decoratorNode)
		if dec != nil {
			result = append(result, *dec)
		}
	}
	return result
}

// reflectDecorator converts a decorator AST node to a Decorator struct.
func (h *TypeScriptReflectionHost) reflectDecorator(node *ast.Node) *Decorator {
	if !ast.IsDecorator(node) {
		return nil
	}
	decoratorData := node.AsDecorator()
	decoratorExpr := decoratorData.Expression
	var args []*ast.Node

	// Unwrap call expressions: @Decorator(...) -> extract expression and args.
	if ast.IsCallExpression(decoratorExpr) {
		call := decoratorExpr.AsCallExpression()
		if call.Arguments != nil {
			for _, arg := range call.Arguments.Nodes {
				args = append(args, arg)
			}
		}
		decoratorExpr = call.Expression
	}

	// The decorator must resolve to an identifier or namespaced identifier.
	if !isDecoratorIdentifier(decoratorExpr) {
		return nil
	}

	var decoratorIdentifier *ast.Node
	if ast.IsIdentifier(decoratorExpr) {
		decoratorIdentifier = decoratorExpr
	} else {
		// Property access expression (e.g. core.Component)
		decoratorIdentifier = decoratorExpr.AsPropertyAccessExpression().Name()
	}

	var importDecl *Import
	if h.checker != nil {
		importDecl = h.GetImportOfIdentifier(decoratorIdentifier)
	}

	return &Decorator{
		Name:       decoratorIdentifier.AsIdentifier().Text,
		Identifier: decoratorExpr,
		Import:     importDecl,
		Node:       node,
		Args:       args,
	}
}

// isDecoratorIdentifier checks if the expression is a valid decorator identifier.
func isDecoratorIdentifier(exp *ast.Node) bool {
	if ast.IsIdentifier(exp) {
		return true
	}
	if ast.IsPropertyAccessExpression(exp) {
		pae := exp.AsPropertyAccessExpression()
		return ast.IsIdentifier(pae.Expression) && ast.IsIdentifier(pae.Name())
	}
	return false
}

// GetMembersOfClass returns all class members of the given class declaration.
func (h *TypeScriptReflectionHost) GetMembersOfClass(clazz *ast.Node) []ClassMember {
	if clazz == nil {
		return nil
	}
	if !ast.IsClassDeclaration(clazz) && !ast.IsClassExpression(clazz) {
		return nil
	}
	var members *ast.ClassElementList
	if ast.IsClassDeclaration(clazz) {
		members = clazz.AsClassDeclaration().Members
	} else {
		members = clazz.AsClassExpression().Members
	}
	if members == nil {
		return nil
	}
	var result []ClassMember
	for _, memberNode := range members.Nodes {
		m := reflectClassMemberNode(memberNode)
		if m == nil {
			continue
		}
		m.Decorators = h.GetDecoratorsOfDeclaration(memberNode)
		result = append(result, *m)
	}
	return result
}

// reflectClassMemberNode converts a class element AST node to a ClassMember.
func reflectClassMemberNode(node *ast.Node) *ClassMember {
	var kind ClassMemberKind
	var value *ast.Node
	var name string
	var nameNode *ast.Node

	switch {
	case ast.IsPropertyDeclaration(node):
		kind = Property
		pd := node.AsPropertyDeclaration()
		value = pd.Initializer
	case ast.IsGetAccessorDeclaration(node):
		kind = Getter
	case ast.IsSetAccessorDeclaration(node):
		kind = Setter
	case ast.IsMethodDeclaration(node):
		kind = Method
	case ast.IsConstructorDeclaration(node):
		kind = Constructor
	default:
		return nil
	}

	if ast.IsConstructorDeclaration(node) {
		name = "constructor"
	} else {
		nameNode = node.Name()
		if nameNode == nil {
			return nil
		}
		switch {
		case ast.IsIdentifier(nameNode):
			name = nameNode.AsIdentifier().Text
		case ast.IsStringLiteral(nameNode):
			name = nameNode.AsStringLiteral().Text
		case ast.IsPrivateIdentifier(nameNode):
			name = nameNode.AsPrivateIdentifier().Text
		default:
			return nil
		}
	}

	flags := node.ModifierFlags()
	isStatic := flags&ast.ModifierFlagsStatic != 0
	isReadonly := flags&ast.ModifierFlagsReadonly != 0
	accessLevel := PublicWritable

	if flags&ast.ModifierFlagsPrivate != 0 {
		accessLevel = Private
	} else if flags&ast.ModifierFlagsProtected != 0 {
		accessLevel = Protected
	} else if isReadonly {
		accessLevel = PublicReadonly
	}
	if nameNode != nil && ast.IsPrivateIdentifier(nameNode) {
		accessLevel = EcmaScriptPrivate
	}

	return &ClassMember{
		Node:           node,
		Implementation: node,
		Kind:           kind,
		AccessLevel:    accessLevel,
		Name:           name,
		NameNode:       nameNode,
		Value:          value,
		IsStatic:       isStatic,
	}
}

// GetConstructorParameters returns constructor parameters for the given class.
func (h *TypeScriptReflectionHost) GetConstructorParameters(clazz *ast.Node) []CtorParameter {
	if clazz == nil {
		return nil
	}
	if !ast.IsClassDeclaration(clazz) && !ast.IsClassExpression(clazz) {
		return nil
	}
	var members *ast.ClassElementList
	if ast.IsClassDeclaration(clazz) {
		members = clazz.AsClassDeclaration().Members
	} else {
		members = clazz.AsClassExpression().Members
	}
	if members == nil {
		return nil
	}
	var ctor *ast.Node
	for _, member := range members.Nodes {
		if ast.IsConstructorDeclaration(member) {
			cd := member.AsConstructorDeclaration()
			if cd.Body != nil {
				ctor = member
				break
			}
		}
	}
	if ctor == nil {
		return nil
	}
	ctorDecl := ctor.AsConstructorDeclaration()
	if ctorDecl.Parameters == nil {
		return []CtorParameter{}
	}
	var result []CtorParameter
	for _, paramNode := range ctorDecl.Parameters.Nodes {
		param := paramNode.AsParameterDeclaration()
		name := ""
		if param.Name() != nil && ast.IsIdentifier(param.Name()) {
			name = param.Name().AsIdentifier().Text
		}
		decorators := h.GetDecoratorsOfDeclaration(paramNode)

		// Resolve TypeValueReference
		typeRef := h.typeToValue(param.Type)

		result = append(result, CtorParameter{
			Name:               name,
			NameNode:           param.Name(),
			TypeNode:           param.Type,
			TypeValueReference: typeRef,
			Decorators:         decorators,
		})
	}
	return result
}

// typeToValue converts a type node to a TypeValueReference.
// This mirrors the TypeScript type_to_value.ts typeToValue function.
func (h *TypeScriptReflectionHost) typeToValue(typeNode *ast.Node) TypeValueReference {
	if typeNode == nil {
		return &UnavailableTypeValueReference{Reason: "missing type"}
	}

	// Handle nullable union types: Foo|null -> extract Foo
	effectiveTypeNode := typeNode
	if ast.IsUnionTypeNode(typeNode) {
		union := typeNode.AsUnionTypeNode()
		if union.Types == nil {
			return &UnavailableTypeValueReference{Reason: "unsupported type"}
		}
		// Filter out null literal types
		var nonNullTypes []*ast.Node
		for _, t := range union.Types.Nodes {
			if ast.IsLiteralTypeNode(t) {
				lit := t.AsLiteralTypeNode()
				if lit.Literal != nil && lit.Literal.Kind == ast.KindNullKeyword {
					continue
				}
			} else if t.Kind == ast.KindUndefinedKeyword {
				continue
			}
			nonNullTypes = append(nonNullTypes, t)
		}
		if len(nonNullTypes) == 1 {
			effectiveTypeNode = nonNullTypes[0]
		} else {
			return &UnavailableTypeValueReference{Reason: "unsupported type"}
		}
	}

	if !ast.IsTypeReferenceNode(effectiveTypeNode) {
		return &UnavailableTypeValueReference{Reason: "unsupported type"}
	}

	typeRef := effectiveTypeNode.AsTypeReferenceNode()
	typeName := typeRef.TypeName

	// Resolve to symbol and its local origin
	local, decl, symbolNames := h.resolveTypeSymbols(typeName)
	if local == nil || decl == nil {
		if fallback := fallbackResolveImport(leftMostForFallback(typeName), symbolNamesForFallback(typeName)); fallback != nil {
			return fallback
		}
		if fallback := fallbackResolveLocal(leftMostForFallback(typeName)); fallback != nil {
			return fallback
		}
		return &UnavailableTypeValueReference{Reason: "unknown reference"}
	}

	// Check that the symbol has a value declaration
	if decl.ValueDeclaration == nil && decl.Flags&ast.SymbolFlagsConstEnum != 0 {
		return &UnavailableTypeValueReference{Reason: "no value declaration"}
	}
	if decl.ValueDeclaration == nil {
		// Check if it is a type-only decl
		if len(decl.Declarations) == 0 {
			if fallback := fallbackResolveImport(leftMostForFallback(typeName), symbolNamesForFallback(typeName)); fallback != nil {
				return fallback
			}
			if fallback := fallbackResolveLocal(leftMostForFallback(typeName)); fallback != nil {
				return fallback
			}
			return &UnavailableTypeValueReference{Reason: "no value declaration"}
		}
		d := decl.Declarations[0]
		if d.Kind == ast.KindTypeParameter || d.Kind == ast.KindTypeAliasDeclaration || d.Kind == ast.KindInterfaceDeclaration {
			return &UnavailableTypeValueReference{Reason: "no value declaration"}
		}
		// for things like namespace declarations, fall through
	}

	// Look at the local symbol's declarations to see if it was imported
	if len(local.Declarations) > 0 {
		firstDecl := local.Declarations[0]

		if ast.IsImportClause(firstDecl) {
			// Default import: import Foo from './foo'
			clause := firstDecl.AsImportClause()
			if clause.Name() == nil {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			// Check type-only
			if clause.PhaseModifier == ast.KindTypeKeyword {
				return &UnavailableTypeValueReference{Reason: "type-only import"}
			}
			importDecl := firstDecl.Parent
			if !ast.IsImportDeclaration(importDecl) {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			return &LocalTypeValueReference{
				Expression:             clause.Name(),
				DefaultImportStatement: importDecl,
			}
		} else if ast.IsImportSpecifier(firstDecl) {
			// Named import: import {Foo} or import {Foo as Bar}
			spec := firstDecl.AsImportSpecifier()
			if spec.IsTypeOnly {
				return &UnavailableTypeValueReference{Reason: "type-only import"}
			}

			// Get the original exported name
			var importedName string
			if spec.PropertyName != nil {
				importedName = spec.PropertyName.Text()
			} else {
				importedName = spec.Name().Text()
			}

			// Navigate up: ImportSpecifier -> NamedImports -> ImportClause -> ImportDeclaration
			namedImports := firstDecl.Parent
			if namedImports == nil {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			importClause := namedImports.Parent
			if importClause == nil {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			if ast.IsImportClause(importClause) {
				clause := importClause.AsImportClause()
				if clause.PhaseModifier == ast.KindTypeKeyword {
					return &UnavailableTypeValueReference{Reason: "type-only import"}
				}
			}
			importDeclaration := importClause.Parent
			if importDeclaration == nil || !ast.IsImportDeclaration(importDeclaration) {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			moduleSpec := importDeclaration.AsImportDeclaration().ModuleSpecifier
			if moduleSpec == nil || !ast.IsStringLiteral(moduleSpec) {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			moduleName := moduleSpec.AsStringLiteral().Text

			return &ImportedTypeValueReference{
				ModuleName:       moduleName,
				ImportedName:     importedName,
				ValueDeclaration: decl.ValueDeclaration,
			}
		} else if ast.IsNamespaceImport(firstDecl) {
			// Namespace import: import * as Foo from './foo'
			// symbolNames[0] is the namespace, symbolNames[1] is the actual name
			if len(symbolNames) < 2 {
				return &UnavailableTypeValueReference{Reason: "namespace import used directly"}
			}
			importedName := symbolNames[1]

			// Navigate up: NamespaceImport -> ImportClause -> ImportDeclaration
			importClause := firstDecl.Parent
			if importClause == nil {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			importDeclaration := importClause.Parent
			if importDeclaration == nil || !ast.IsImportDeclaration(importDeclaration) {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			moduleSpec := importDeclaration.AsImportDeclaration().ModuleSpecifier
			if moduleSpec == nil || !ast.IsStringLiteral(moduleSpec) {
				return &UnavailableTypeValueReference{Reason: "unsupported type"}
			}
			moduleName := moduleSpec.AsStringLiteral().Text

			return &ImportedTypeValueReference{
				ModuleName:       moduleName,
				ImportedName:     importedName,
				ValueDeclaration: decl.ValueDeclaration,
			}
		}
	}

	// The type is locally declared - convert the typeName to an expression
	expr := entityNameToExpr(typeName)
	if expr != nil {
		return &LocalTypeValueReference{
			Expression:             expr,
			DefaultImportStatement: nil,
		}
	}
	return &UnavailableTypeValueReference{Reason: "unsupported type"}
}

// resolveTypeSymbols resolves a TypeName (entity name) to its local symbol and declaration symbol.
// Returns (local, decl, symbolNames) or (nil, nil, nil) if not resolvable.
func (h *TypeScriptReflectionHost) resolveTypeSymbols(typeName *ast.Node) (*ast.Symbol, *ast.Symbol, []string) {
	if h.checker == nil || typeName == nil {
		return nil, nil, nil
	}

	// Get the symbol at the full typeName location
	typeRefSymbol := h.checker.GetSymbolAtLocation(typeName)
	if typeRefSymbol == nil {
		return nil, nil, nil
	}

	// Walk the qualified name to collect symbol names and find the left-most identifier
	symbolNames := []string{}
	leftMost := typeName
	for ast.IsQualifiedName(leftMost) {
		qn := leftMost.AsQualifiedName()
		symbolNames = append([]string{qn.Right.Text()}, symbolNames...)
		leftMost = qn.Left
	}
	if ast.IsIdentifier(leftMost) {
		symbolNames = append([]string{leftMost.AsIdentifier().Text}, symbolNames...)
	}

	local := typeRefSymbol
	if leftMost != typeName {
		// For qualified names, resolve local to the left-most symbol
		localTmp := h.checker.GetSymbolAtLocation(leftMost)
		if localTmp != nil {
			local = localTmp
		}
	}

	// De-alias to get the declaration symbol
	decl := typeRefSymbol
	if decl.Flags&ast.SymbolFlagsAlias != 0 {
		decl = h.checker.GetAliasedSymbol(decl)
	}

	return local, decl, symbolNames
}

// entityNameToExpr converts an entity name (type position) to an equivalent expression node.
// For qualified names like `i1.Bar`, returns the full QualifiedName node.
// ArgExpressionToString (in ngtsctest) handles QualifiedName nodes.
func entityNameToExpr(node *ast.Node) *ast.Node {
	if ast.IsQualifiedName(node) {
		// Return the full qualified name node - ArgExpressionToString handles it.
		return node
	} else if ast.IsIdentifier(node) {
		return node
	}
	return nil
}

func leftMostForFallback(typeName *ast.Node) *ast.Node {
	leftMost := typeName
	for ast.IsQualifiedName(leftMost) {
		leftMost = leftMost.AsQualifiedName().Left
	}
	return leftMost
}

func symbolNamesForFallback(typeName *ast.Node) []string {
	symbolNames := []string{}
	leftMost := typeName
	for ast.IsQualifiedName(leftMost) {
		qn := leftMost.AsQualifiedName()
		symbolNames = append([]string{qn.Right.Text()}, symbolNames...)
		leftMost = qn.Left
	}
	if ast.IsIdentifier(leftMost) {
		symbolNames = append([]string{leftMost.AsIdentifier().Text}, symbolNames...)
	}
	return symbolNames
}

func fallbackResolveImport(leftMost *ast.Node, symbolNames []string) TypeValueReference {
	if !ast.IsIdentifier(leftMost) {
		return nil
	}
	identName := leftMost.AsIdentifier().Text
	sourceFile := ast.GetSourceFileOfNode(leftMost)
	if sourceFile == nil || sourceFile.Statements == nil {
		return nil
	}

	for _, stmtNode := range sourceFile.Statements.Nodes {
		if !ast.IsImportDeclaration(stmtNode) {
			continue
		}
		importDecl := stmtNode.AsImportDeclaration()
		if importDecl.ImportClause == nil {
			continue
		}
		clause := importDecl.ImportClause.AsImportClause()
		if clause.PhaseModifier == ast.KindTypeKeyword {
			continue
		}

		// check default import
		if clause.Name() != nil && clause.Name().AsIdentifier().Text == identName {
			if len(symbolNames) > 1 {
				return nil
			}
			return &LocalTypeValueReference{
				Expression:             clause.Name(),
				DefaultImportStatement: importDecl.AsNode(),
			}
		}

		// check named imports
		if clause.NamedBindings != nil {
			if ast.IsNamedImports(clause.NamedBindings) {
				namedImports := clause.NamedBindings.AsNamedImports()
				for _, el := range namedImports.Elements.Nodes {
					spec := el.AsImportSpecifier()
					if spec.IsTypeOnly {
						continue
					}
					if spec.Name().Text() == identName {
						var importedName string
						if spec.PropertyName != nil {
							importedName = spec.PropertyName.Text()
						} else {
							importedName = spec.Name().Text()
						}
						moduleSpec := importDecl.ModuleSpecifier
						if moduleSpec != nil && ast.IsStringLiteral(moduleSpec) {
							moduleName := moduleSpec.AsStringLiteral().Text
							
							if len(symbolNames) > 1 {
								return nil
							}

							return &ImportedTypeValueReference{
								ModuleName:       moduleName,
								ImportedName:     importedName,
								ValueDeclaration: nil,
							}
						}
					}
				}
			} else if ast.IsNamespaceImport(clause.NamedBindings) {
				ns := clause.NamedBindings.AsNamespaceImport()
				if ns.Name().Text() == identName {
					if len(symbolNames) < 2 {
						return &UnavailableTypeValueReference{Reason: "namespace import used directly"}
					}
					importedName := symbolNames[1]
					moduleSpec := importDecl.ModuleSpecifier
					if moduleSpec != nil && ast.IsStringLiteral(moduleSpec) {
						moduleName := moduleSpec.AsStringLiteral().Text
						return &ImportedTypeValueReference{
							ModuleName:       moduleName,
							ImportedName:     importedName,
							ValueDeclaration: nil,
						}
					}
				}
			}
		}
	}
	return nil
}

func fallbackResolveLocal(leftMost *ast.Node) TypeValueReference {
	if !ast.IsIdentifier(leftMost) {
		return nil
	}
	identName := leftMost.AsIdentifier().Text
	sourceFile := ast.GetSourceFileOfNode(leftMost)
	if sourceFile == nil || sourceFile.Statements == nil {
		return nil
	}
	for _, stmtNode := range sourceFile.Statements.Nodes {
		if stmtNode.Kind == ast.KindClassDeclaration {
			classDecl := stmtNode.AsClassDeclaration()
			if classDecl.Name() != nil && ast.IsIdentifier(classDecl.Name()) && classDecl.Name().AsIdentifier().Text == identName {
				return &LocalTypeValueReference{
					Expression:             classDecl.Name(),
					DefaultImportStatement: nil,
				}
			}
		} else if stmtNode.Kind == ast.KindVariableStatement {
			varStmt := stmtNode.AsVariableStatement()
			if varStmt.DeclarationList != nil {
				varList := varStmt.DeclarationList.AsVariableDeclarationList()
				if varList.Declarations != nil {
					for _, el := range varList.Declarations.Nodes {
						decl := el.AsVariableDeclaration()
						if decl.Name() != nil && ast.IsIdentifier(decl.Name()) && decl.Name().AsIdentifier().Text == identName {
							return &LocalTypeValueReference{
								Expression:             decl.Name(),
								DefaultImportStatement: nil,
							}
						}
					}
				}
			}
		} else if stmtNode.Kind == ast.KindFunctionDeclaration {
			funcDecl := stmtNode.AsFunctionDeclaration()
			if funcDecl.Name() != nil && ast.IsIdentifier(funcDecl.Name()) && funcDecl.Name().AsIdentifier().Text == identName {
				return &LocalTypeValueReference{
					Expression:             funcDecl.Name(),
					DefaultImportStatement: nil,
				}
			}
		} else if stmtNode.Kind == ast.KindEnumDeclaration {
			enumDecl := stmtNode.AsEnumDeclaration()
			if enumDecl.Name() != nil && ast.IsIdentifier(enumDecl.Name()) && enumDecl.Name().AsIdentifier().Text == identName {
				return &LocalTypeValueReference{
					Expression:             enumDecl.Name(),
					DefaultImportStatement: nil,
				}
			}
		}
	}
	return nil
}




// GetDefinitionOfFunction returns information about a function-like node.
func (h *TypeScriptReflectionHost) GetDefinitionOfFunction(fn *ast.Node) *FunctionDefinition {
	if fn == nil {
		return nil
	}
	if !ast.IsFunctionDeclaration(fn) && !ast.IsMethodDeclaration(fn) &&
		!ast.IsFunctionExpression(fn) && !ast.IsArrowFunction(fn) {
		return nil
	}
	return &FunctionDefinition{Node: fn}
}

// GetImportOfIdentifier resolves an identifier to its import info.
// Handles both direct imports and namespaced imports (e.g., abs.Target).
func (h *TypeScriptReflectionHost) GetImportOfIdentifier(id *ast.Node) *Import {
	if h.checker == nil || id == nil {
		return nil
	}
	// First try a direct import (the identifier itself is directly imported).
	directImport := h.getDirectImportOfIdentifier(id)
	if directImport != nil {
		return directImport
	}
	// Then check if this identifier is the right side of a QualifiedName (type position namespace)
	// e.g. abs.Target where Target is the right side of a QualifiedName.
	if id.Parent != nil && ast.IsQualifiedName(id.Parent) {
		qn := id.Parent.AsQualifiedName()
		if qn.Right == id {
			// Find the left-most identifier in the qualified name chain
			return h.getImportOfNamespacedIdentifier(id, getQualifiedNameRoot(id.Parent))
		}
	}
	// Also handle property access expressions (value position namespace).
	if id.Parent != nil && ast.IsPropertyAccessExpression(id.Parent) {
		pae := id.Parent.AsPropertyAccessExpression()
		if pae.Name() == id {
			return h.getImportOfNamespacedIdentifier(id, getFarLeftIdentifier(id.Parent))
		}
	}
	return nil
}


// getDirectImportOfIdentifier resolves a directly-imported identifier to its import info.
func (h *TypeScriptReflectionHost) getDirectImportOfIdentifier(id *ast.Node) *Import {
	symbol := h.checker.GetSymbolAtLocation(id)
	if symbol == nil || len(symbol.Declarations) != 1 {
		return nil
	}
	decl := symbol.Declarations[0]
	importDecl := getContainingImportDeclaration(decl)
	if importDecl == nil {
		return nil
	}
	moduleSpecifier := importDecl.AsImportDeclaration().ModuleSpecifier
	if moduleSpecifier == nil || !ast.IsStringLiteral(moduleSpecifier) {
		return nil
	}
	name := ""
	if ast.IsImportSpecifier(decl) {
		spec := decl.AsImportSpecifier()
		if spec.PropertyName != nil {
			name = spec.PropertyName.Text()
		} else {
			name = spec.Name().Text()
		}
	} else if id != nil && ast.IsIdentifier(id) {
		name = id.AsIdentifier().Text
	}
	return &Import{
		From: moduleSpecifier.AsStringLiteral().Text,
		Name: name,
		Node: importDecl,
	}
}

// getImportOfNamespacedIdentifier resolves an identifier like `Target` in `ns.Target`
// where `ns` is a namespace import (import * as ns from ...).
func (h *TypeScriptReflectionHost) getImportOfNamespacedIdentifier(id *ast.Node, namespaceId *ast.Node) *Import {
	if namespaceId == nil {
		return nil
	}
	namespaceSymbol := h.checker.GetSymbolAtLocation(namespaceId)
	if namespaceSymbol == nil || len(namespaceSymbol.Declarations) != 1 {
		return nil
	}
	decl := namespaceSymbol.Declarations[0]
	if !ast.IsNamespaceImport(decl) {
		return nil
	}
	// Navigate: NamespaceImport -> ImportClause -> ImportDeclaration
	importClause := decl.Parent
	if importClause == nil {
		return nil
	}
	importDeclaration := importClause.Parent
	if importDeclaration == nil || !ast.IsImportDeclaration(importDeclaration) {
		return nil
	}
	moduleSpec := importDeclaration.AsImportDeclaration().ModuleSpecifier
	if moduleSpec == nil || !ast.IsStringLiteral(moduleSpec) {
		return nil
	}
	name := ""
	if ast.IsIdentifier(id) {
		name = id.AsIdentifier().Text
	}
	return &Import{
		From: moduleSpec.AsStringLiteral().Text,
		Name: name,
		Node: importDeclaration,
	}
}

// getQualifiedNameRoot returns the left-most identifier of a QualifiedName chain.
func getQualifiedNameRoot(node *ast.Node) *ast.Node {
	left := node.AsQualifiedName().Left
	for ast.IsQualifiedName(left) {
		left = left.AsQualifiedName().Left
	}
	if ast.IsIdentifier(left) {
		return left
	}
	return nil
}

// getFarLeftIdentifier returns the far-left identifier in a property access chain.
func getFarLeftIdentifier(node *ast.Node) *ast.Node {
	expr := node.AsPropertyAccessExpression().Expression
	for ast.IsPropertyAccessExpression(expr) {
		expr = expr.AsPropertyAccessExpression().Expression
	}
	if ast.IsIdentifier(expr) {
		return expr
	}
	return nil
}

// GetDeclarationOfIdentifier resolves an identifier to its declaration.
func (h *TypeScriptReflectionHost) GetDeclarationOfIdentifier(id *ast.Node) *Declaration {
	if id == nil {
		return nil
	}
	if h.checker == nil {
		sf := getSourceFileOfNode(id)
		if sf == nil {
			return nil
		}
		var idText string
		if ast.IsIdentifier(id) {
			idText = id.AsIdentifier().Text
		} else if ast.IsPrivateIdentifier(id) {
			idText = id.AsPrivateIdentifier().Text
		} else {
			return nil
		}
		var foundDecl *Declaration
		if stmts := sf.Statements(); stmts != nil {
			for _, n := range stmts {
				if n.Kind == ast.KindClassDeclaration {
					c := n.AsClassDeclaration()
					if c.Name() != nil && c.Name().AsIdentifier().Text == idText {
						foundDecl = &Declaration{Node: n, ViaModule: ""}
						break
					}
				}
			}
		}
		return foundDecl
	}
	symbol := h.checker.GetSymbolAtLocation(id)
	if symbol == nil {
		return nil
	}
	// Get the import info before aliasing (to track viaModule).
	importInfo := h.GetImportOfIdentifier(id)

	// Follow aliases.
	for symbol.Flags&ast.SymbolFlagsAlias != 0 {
		symbol = h.checker.GetAliasedSymbol(symbol)
	}
	var node *ast.Node
	if symbol.ValueDeclaration != nil {
		node = symbol.ValueDeclaration
	} else if len(symbol.Declarations) > 0 {
		node = symbol.Declarations[0]
	}
	if node == nil {
		return nil
	}
	viaModule := h.viaModule(node, id, importInfo)
	return &Declaration{Node: node, ViaModule: viaModule}
}

// viaModule determines the module path for a declaration given the import info.
// Returns the module specifier if the declaration came from a non-relative import,
// or empty string for local declarations.
func (h *TypeScriptReflectionHost) viaModule(declaration *ast.Node, originalId *ast.Node, importInfo *Import) string {
	if importInfo != nil && importInfo.From != "" && !isRelativePath(importInfo.From) {
		return importInfo.From
	}

	// For relative paths or ambient imports, use the absolute file path
	declSF := getSourceFileOfNode(declaration)
	origSF := getSourceFileOfNode(originalId)
	if declSF != nil && origSF != nil && declSF != origSF {
		fileName := declSF.AsSourceFile().FileName()
		if strings.HasSuffix(fileName, ".ts") {
			fileName = fileName[:len(fileName)-3]
		}
		return fileName
	}

	return ""
}

// isRelativePath returns true if the path starts with . or ..
func isRelativePath(p string) bool {
	return len(p) > 0 && (p[0] == '.' || (len(p) > 1 && p[0] == '/' ))
}

// getSourceFileOfNode walks up the parent chain to find the SourceFile.
func getSourceFileOfNode(node *ast.Node) *ast.Node {
	for node != nil {
		if ast.IsSourceFile(node) {
			return node
		}
		node = node.Parent
	}
	return nil
}


// GetExportsOfModule returns all exports from a source file node.
func (h *TypeScriptReflectionHost) GetExportsOfModule(module *ast.Node) map[string]Declaration {
	if h.checker == nil || module == nil || !ast.IsSourceFile(module) {
		return nil
	}
	symbol := h.checker.GetSymbolAtLocation(module)
	if symbol == nil {
		return nil
	}
	result := make(map[string]Declaration)
	h.collectExports(symbol, result, make(map[*ast.Symbol]bool))
	return result
}

// collectExports recursively collects named exports from a module symbol,
// including those from star re-exports (export * from ...).
func (h *TypeScriptReflectionHost) collectExports(symbol *ast.Symbol, result map[string]Declaration, visited map[*ast.Symbol]bool) {
	if symbol == nil || symbol.Exports == nil || visited[symbol] {
		return
	}
	visited[symbol] = true
	for name, exportSymbol := range symbol.Exports {
		// Skip internal TypeScript symbols (like __export from export*, special prefixes).
		// InternalSymbolNamePrefix is \xFE (byte 254).
		if len(name) == 0 || name[0] == ast.InternalSymbolNamePrefix[0] || name == "export=" {
			// But handle the export* star entry specially.
			if name == ast.InternalSymbolNameExportStar {
				// Resolve each star export declaration
				for _, decl := range exportSymbol.Declarations {
					if !ast.IsExportDeclaration(decl) {
						continue
					}
					moduleSpec := decl.AsExportDeclaration().ModuleSpecifier
					if moduleSpec == nil {
						continue
					}
					// Resolve the module symbol
					moduleSym := h.checker.GetSymbolAtLocation(moduleSpec)
					if moduleSym == nil || visited[moduleSym] {
						continue
					}
					// Follow aliases to get the actual module symbol
					for moduleSym.Flags&ast.SymbolFlagsAlias != 0 {
						moduleSym = h.checker.GetAliasedSymbol(moduleSym)
					}
					if visited[moduleSym] {
						continue
					}
					// Recursively collect exports from this module, but only add names not already present
					starResult := make(map[string]Declaration)
					h.collectExports(moduleSym, starResult, visited)
					for starName, starDecl := range starResult {
						if _, exists := result[starName]; !exists {
							result[starName] = starDecl
						}
					}
				}
			}
			continue
		}
		sym := exportSymbol
		for sym.Flags&ast.SymbolFlagsAlias != 0 {
			sym = h.checker.GetAliasedSymbol(sym)
		}
		if sym.ValueDeclaration != nil {
			result[name] = Declaration{Node: sym.ValueDeclaration}
		} else if len(sym.Declarations) > 0 {
			result[name] = Declaration{Node: sym.Declarations[0]}
		}
	}
}


// IsClass returns true if the node is a named class declaration.
func (h *TypeScriptReflectionHost) IsClass(node *ast.Node) bool {
	if node == nil {
		return false
	}
	if !ast.IsClassDeclaration(node) {
		return false
	}
	return node.AsClassDeclaration().Name() != nil
}

// HasBaseClass returns true if the class has a base class via extends.
func (h *TypeScriptReflectionHost) HasBaseClass(clazz *ast.Node) bool {
	return h.GetBaseClassExpression(clazz) != nil
}

// GetBaseClassExpression returns the expression in the extends clause, or nil.
func (h *TypeScriptReflectionHost) GetBaseClassExpression(clazz *ast.Node) *ast.Node {
	if clazz == nil {
		return nil
	}
	if !ast.IsClassDeclaration(clazz) && !ast.IsClassExpression(clazz) {
		return nil
	}
	var heritageClauses *ast.HeritageClauseList
	if ast.IsClassDeclaration(clazz) {
		heritageClauses = clazz.AsClassDeclaration().HeritageClauses
	} else {
		heritageClauses = clazz.AsClassExpression().HeritageClauses
	}
	if heritageClauses == nil {
		return nil
	}
	for _, clause := range heritageClauses.Nodes {
		hc := clause.AsHeritageClause()
		if hc.Token == ast.KindExtendsKeyword {
			if hc.Types != nil && len(hc.Types.Nodes) > 0 {
				return hc.Types.Nodes[0].AsExpressionWithTypeArguments().Expression
			}
		}
	}
	return nil
}

// GetGenericArityOfClass returns the number of type parameters of the class.
func (h *TypeScriptReflectionHost) GetGenericArityOfClass(clazz *ast.Node) int {
	if clazz == nil || !ast.IsClassDeclaration(clazz) {
		return -1
	}
	tp := clazz.AsClassDeclaration().TypeParameters
	if tp == nil {
		return 0
	}
	return len(tp.Nodes)
}

// GetVariableValue returns the initializer of a variable declaration.
func (h *TypeScriptReflectionHost) GetVariableValue(declaration *ast.Node) *ast.Node {
	if declaration == nil || !ast.IsVariableDeclaration(declaration) {
		return nil
	}
	return declaration.AsVariableDeclaration().Initializer
}

// IsStaticallyExported checks if a declaration is statically exported.
func (h *TypeScriptReflectionHost) IsStaticallyExported(decl *ast.Node) bool {
	if decl == nil {
		return false
	}
	topLevel := decl
	if ast.IsVariableDeclaration(decl) {
		parent := decl.Parent
		if parent != nil && ast.IsVariableDeclarationList(parent) {
			topLevel = parent.Parent
		}
	}
	return ast.HasSyntacticModifier(topLevel, ast.ModifierFlagsExport)
}

// getContainingImportDeclaration walks up the AST to find the enclosing ImportDeclaration.
func getContainingImportDeclaration(node *ast.Node) *ast.Node {
	parent := node.Parent
	for parent != nil && !ast.IsSourceFile(parent) {
		if ast.IsImportDeclaration(parent) {
			return parent
		}
		parent = parent.Parent
	}
	return nil
}
