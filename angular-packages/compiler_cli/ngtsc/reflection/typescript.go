package reflection

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

// TypeScriptReflectionHost implements ReflectionHost using the TypeScript type-checker
// and AST (via the typescript-go AST package). No regex is used.
type TypeScriptReflectionHost struct {
	checker            *checker.Checker
	isLocalCompilation bool
	// Cache of local exported declarations per SourceFile (keyed by SourceFile node pointer).
	localExportsCache map[*ast.SourceFile]map[*ast.Node]bool
}

// NewTypeScriptReflectionHost creates a new TypeScriptReflectionHost.
func NewTypeScriptReflectionHost(chk *checker.Checker) *TypeScriptReflectionHost {
	return &TypeScriptReflectionHost{
		checker:           chk,
		localExportsCache: make(map[*ast.SourceFile]map[*ast.Node]bool),
	}
}

// NewTypeScriptReflectionHostWithOptions creates a new TypeScriptReflectionHost with options.
func NewTypeScriptReflectionHostWithOptions(chk *checker.Checker, isLocalCompilation bool) *TypeScriptReflectionHost {
	return &TypeScriptReflectionHost{
		checker:            chk,
		isLocalCompilation: isLocalCompilation,
		localExportsCache:  make(map[*ast.SourceFile]map[*ast.Node]bool),
	}
}

// GetDecoratorsOfDeclaration returns the decorators applied to a declaration node.
// Returns nil if the node cannot have decorators or has none.
func (h *TypeScriptReflectionHost) GetDecoratorsOfDeclaration(declaration *ast.Node) []Decorator {
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

	// Unwrap call expressions: @Decorator(...) -> take the expression and args.
	if ast.IsCallExpression(decoratorExpr) {
		call := decoratorExpr.AsCallExpression()
		for _, arg := range call.Arguments.Nodes {
			args = append(args, arg)
		}
		decoratorExpr = call.Expression
	}

	// The final resolved decorator must be an identifier or namespaced identifier (a.B).
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

	importDecl := h.GetImportOfIdentifier(decoratorIdentifier)

	return &Decorator{
		Name:       decoratorIdentifier.AsIdentifier().Text,
		Identifier: decoratorExpr,
		Import:     importDecl,
		Node:       node,
		Args:       args,
	}
}

// GetMembersOfClass returns all class members of the given class declaration.
func (h *TypeScriptReflectionHost) GetMembersOfClass(clazz ClassDeclaration) []ClassMember {
	if !ast.IsClassDeclaration(clazz) && !ast.IsClassExpression(clazz) {
		return nil
	}

	var classData interface {
		GetMembers() *ast.ClassElementList
	}

	if ast.IsClassDeclaration(clazz) {
		cd := clazz.AsClassDeclaration()
		classData = &classDeclarationAdapter{cd}
	} else {
		ce := clazz.AsClassExpression()
		classData = &classExpressionAdapter{ce}
	}

	members := classData.GetMembers()
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

// classDeclarationAdapter adapts ClassDeclaration to a common interface.
type classDeclarationAdapter struct{ *ast.ClassDeclaration }

func (a *classDeclarationAdapter) GetMembers() *ast.ClassElementList { return a.Members }

// classExpressionAdapter adapts ClassExpression to a common interface.
type classExpressionAdapter struct{ *ast.ClassExpression }

func (a *classExpressionAdapter) GetMembers() *ast.ClassElementList { return a.Members }

// reflectClassMemberNode converts a class element AST node to a ClassMember.
// Returns nil for unsupported member kinds.
func reflectClassMemberNode(node *ast.Node) *ClassMember {
	var kind ClassMemberKind
	var value *ast.Node
	var name string
	var nameNode *ast.Node

	switch {
	case ast.IsPropertyDeclaration(node):
		kind = Property
		pd := node.AsPropertyDeclaration()
		if pd.Initializer != nil {
			value = pd.Initializer
		}
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

	accessLevel, isStatic := extractModifiersOfMember(node)

	return &ClassMember{
		Node:           node,
		Implementation: node,
		Kind:           kind,
		Type:           typeAnnotationOf(node),
		AccessLevel:    accessLevel,
		Name:           name,
		NameNode:       nameNode,
		Value:          value,
		IsStatic:       isStatic,
	}
}

// typeAnnotationOf returns the type annotation node for a class member, or nil.
func typeAnnotationOf(node *ast.Node) *ast.Node {
	switch {
	case ast.IsPropertyDeclaration(node):
		return node.AsPropertyDeclaration().Type
	case ast.IsMethodDeclaration(node):
		return node.AsMethodDeclaration().Type
	case ast.IsGetAccessorDeclaration(node):
		return node.AsGetAccessorDeclaration().Type
	case ast.IsSetAccessorDeclaration(node):
		return nil // Set accessors don't have a type annotation
	}
	return nil
}

// extractModifiersOfMember reads the modifier flags from a class element.
func extractModifiersOfMember(node *ast.Node) (ClassMemberAccessLevel, bool) {
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

	// Check for EcmaScript private identifier (#name)
	if nameNode := node.Name(); nameNode != nil && ast.IsPrivateIdentifier(nameNode) {
		accessLevel = EcmaScriptPrivate
	}

	return accessLevel, isStatic
}

// GetConstructorParameters returns the constructor parameters of the given class.
// Returns nil if there is no constructor.
func (h *TypeScriptReflectionHost) GetConstructorParameters(clazz ClassDeclaration) []CtorParameter {
	if !ast.IsClassDeclaration(clazz) && !ast.IsClassExpression(clazz) {
		return nil
	}

	var members *ast.ClassElementList
	if ast.IsClassDeclaration(clazz) {
		cd := clazz.AsClassDeclaration()
		members = cd.Members
	} else {
		ce := clazz.AsClassExpression()
		members = ce.Members
	}
	sourceFile := ast.GetSourceFileOfNode(clazz)

	isDeclaration := sourceFile != nil && sourceFile.IsDeclarationFile

	// Find the constructor with a body (implementation) for non-declaration files.
	// For declaration files, take the first constructor.
	var ctor *ast.Node
	if members != nil {
		for _, member := range members.Nodes {
			if ast.IsConstructorDeclaration(member) {
				cd := member.AsConstructorDeclaration()
				if isDeclaration || cd.Body != nil {
					ctor = member
					break
				}
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
		name := parameterName(param.Name())
		decorators := h.GetDecoratorsOfDeclaration(paramNode)

		originalTypeNode := param.Type
		typeNode := originalTypeNode

		// Unwrap simple nullable unions: Foo|null -> Foo
		if typeNode != nil && ast.IsUnionTypeNode(typeNode) {
			unionNode := typeNode.AsUnionTypeNode()
			var nonNullTypes []*ast.Node
			for _, t := range unionNode.Types.Nodes {
				if !(ast.IsLiteralTypeNode(t) && t.AsLiteralTypeNode().Literal.Kind == ast.KindNullKeyword) {
					nonNullTypes = append(nonNullTypes, t)
				}
			}
			if len(nonNullTypes) == 1 {
				typeNode = nonNullTypes[0]
			}
		}

		typeValueRef := TypeToValue(typeNode, h.checker, h.isLocalCompilation)

		result = append(result, CtorParameter{
			Name:               name,
			NameNode:           param.Name(),
			TypeValueReference: typeValueRef,
			TypeNode:           originalTypeNode,
			Decorators:         decorators,
		})
	}
	return result
}

// parameterName extracts the name string from a binding name.
func parameterName(name *ast.Node) string {
	if name == nil {
		return ""
	}
	if ast.IsIdentifier(name) {
		return name.AsIdentifier().Text
	}
	return ""
}

// GetImportOfIdentifier returns the Import info for an identifier.
// Returns nil if the identifier is not imported.
func (h *TypeScriptReflectionHost) GetImportOfIdentifier(id *ast.Node) *Import {
	directImport := h.getDirectImportOfIdentifier(id)
	if directImport != nil {
		return directImport
	}

	parent := id.Parent
	if parent == nil {
		return nil
	}

	if ast.IsQualifiedName(parent) && parent.AsQualifiedName().Right == id {
		return h.getImportOfNamespacedIdentifier(id, getQualifiedNameRoot(parent))
	}
	if ast.IsPropertyAccessExpression(parent) && parent.AsPropertyAccessExpression().Name() == id {
		return h.getImportOfNamespacedIdentifier(id, getFarLeftIdentifier(parent))
	}
	return nil
}

// getDirectImportOfIdentifier resolves an identifier to a direct import.
func (h *TypeScriptReflectionHost) getDirectImportOfIdentifier(id *ast.Node) *Import {
	if h.checker == nil {
		sf := ast.GetSourceFileOfNode(id)
		if sf == nil {
			return nil
		}
		idText := id.AsIdentifier().Text
		for _, stmt := range sf.Statements.Nodes {
			if stmt.Kind == ast.KindImportDeclaration {
				importDecl := stmt.AsImportDeclaration()
				if importDecl.ImportClause != nil && importDecl.ImportClause.AsImportClause().NamedBindings != nil {
					namedBindings := importDecl.ImportClause.AsImportClause().NamedBindings
					if namedBindings.Kind == ast.KindNamedImports {
						namedImports := namedBindings.AsNamedImports()
						for _, element := range namedImports.Elements.Nodes {
							if element.Kind == ast.KindImportSpecifier {
								importSpecifier := element.AsImportSpecifier()
								name := importSpecifier.Name().AsIdentifier().Text
								if name == idText {
									moduleSpecifier := importDecl.ModuleSpecifier
									if moduleSpecifier != nil && ast.IsStringLiteral(moduleSpecifier) {
										return &Import{
											From: moduleSpecifier.AsStringLiteral().Text,
											Name: name, // If there's a propertyName, handle it? We'll assume simple name for now
											Node: stmt,
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return nil
	}
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

	return &Import{
		From: moduleSpecifier.AsStringLiteral().Text,
		Name: getExportedName(decl, id),
		Node: importDecl,
	}
}

// getImportOfNamespacedIdentifier resolves a namespaced import like `ns.Identifier`.
func (h *TypeScriptReflectionHost) getImportOfNamespacedIdentifier(id *ast.Node, namespaceIdentifier *ast.Node) *Import {
	if namespaceIdentifier == nil {
		return nil
	}
	if h.checker == nil {
		sf := ast.GetSourceFileOfNode(id)
		if sf == nil { return nil }
		nsName := namespaceIdentifier.AsIdentifier().Text
		idName := id.AsIdentifier().Text
		for _, stmt := range sf.Statements.Nodes {
			if stmt.Kind == ast.KindImportDeclaration {
				importDecl := stmt.AsImportDeclaration()
				if importDecl.ImportClause != nil && importDecl.ImportClause.AsImportClause().NamedBindings != nil {
					namedBindings := importDecl.ImportClause.AsImportClause().NamedBindings
					if namedBindings.Kind == ast.KindNamespaceImport {
						if namedBindings.AsNamespaceImport().Name().AsIdentifier().Text == nsName {
							mod := importDecl.ModuleSpecifier
							if mod != nil && ast.IsStringLiteral(mod) {
								return &Import{From: mod.AsStringLiteral().Text, Name: idName, Node: stmt}
							}
						}
					}
				}
			}
		}
		return nil
	}
	nsSymbol := h.checker.GetSymbolAtLocation(namespaceIdentifier)
	if nsSymbol == nil || len(nsSymbol.Declarations) != 1 {
		return nil
	}
	declaration := nsSymbol.Declarations[0]
	if !ast.IsNamespaceImport(declaration) {
		return nil
	}
	// NamespaceImport → ImportClause → ImportDeclaration
	importDeclaration := declaration.Parent.Parent
	if !ast.IsImportDeclaration(importDeclaration) {
		return nil
	}
	moduleSpecifier := importDeclaration.AsImportDeclaration().ModuleSpecifier
	if moduleSpecifier == nil || !ast.IsStringLiteral(moduleSpecifier) {
		return nil
	}
	return &Import{
		From: moduleSpecifier.AsStringLiteral().Text,
		Name: id.AsIdentifier().Text,
		Node: importDeclaration,
	}
}

// GetExportsOfModule returns all exports from a source file node.
func (h *TypeScriptReflectionHost) GetExportsOfModule(module *ast.Node) map[string]Declaration {
	if !ast.IsSourceFile(module) {
		return nil
	}

	symbol := h.checker.GetSymbolAtLocation(module)
	if symbol == nil {
		return nil
	}

	result := make(map[string]Declaration)
	for name, exportSymbol := range symbol.Exports {
		decl := h.getDeclarationOfSymbol(exportSymbol, nil)
		if decl != nil {
			result[name] = *decl
		}
	}
	return result
}

// IsClass returns true if the node is a named class declaration.
func (h *TypeScriptReflectionHost) IsClass(node *ast.Node) bool {
	return IsNamedClassDeclaration(node)
}

// HasBaseClass returns true if the class extends another class.
func (h *TypeScriptReflectionHost) HasBaseClass(clazz ClassDeclaration) bool {
	return h.GetBaseClassExpression(clazz) != nil
}

// GetBaseClassExpression returns the expression in the `extends` clause, or nil.
func (h *TypeScriptReflectionHost) GetBaseClassExpression(clazz ClassDeclaration) *ast.Node {
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

// GetDeclarationOfIdentifier resolves an identifier to its declaration.
func (h *TypeScriptReflectionHost) GetDeclarationOfIdentifier(id *ast.Node) *Declaration {
	if h.checker == nil {
		sf := ast.GetSourceFileOfNode(id)
		if sf == nil { return nil }
		var idText string
		if ast.IsIdentifier(id) {
			idText = id.AsIdentifier().Text
		} else if ast.IsPrivateIdentifier(id) {
			idText = id.AsPrivateIdentifier().Text
		} else {
			return nil
		}
		var foundDecl *Declaration
		sf.ForEachChild(func(n *ast.Node) bool {
			if n.Kind == ast.KindClassDeclaration {
				c := n.AsClassDeclaration()
				if c.Name() != nil && c.Name().AsIdentifier().Text == idText {
					foundDecl = &Declaration{Node: n, ViaModule: ""}
					return true // stop
				}
			} else if n.Kind == ast.KindVariableStatement {
				// simplify: just look at VariableDeclarationList
			}
			return false
		})
		return foundDecl
	}
	symbol := h.checker.GetSymbolAtLocation(id)
	if symbol == nil {
		return nil
	}
	decl := h.getDeclarationOfSymbol(symbol, id)
	return decl
}

// GetDefinitionOfFunction returns the definition of a function-like node.
func (h *TypeScriptReflectionHost) GetDefinitionOfFunction(node *ast.Node) *FunctionDefinition {
	if !ast.IsFunctionDeclaration(node) &&
		!ast.IsMethodDeclaration(node) &&
		!ast.IsFunctionExpression(node) &&
		!ast.IsArrowFunction(node) {
		return nil
	}

	var body []*ast.Node
	var bodyNode *ast.Node

	// Get the body based on node kind.
	switch {
	case ast.IsFunctionDeclaration(node):
		if b := node.AsFunctionDeclaration().Body; b != nil {
			bodyNode = b
		}
	case ast.IsMethodDeclaration(node):
		if b := node.AsMethodDeclaration().Body; b != nil {
			bodyNode = b
		}
	case ast.IsFunctionExpression(node):
		if b := node.AsFunctionExpression().Body; b != nil {
			bodyNode = b
		}
	case ast.IsArrowFunction(node):
		if b := node.AsArrowFunction().Body; b != nil {
			bodyNode = b
		}
	}

	if bodyNode != nil {
		if ast.IsBlock(bodyNode) {
			block := bodyNode.AsBlock()
			if block.Statements != nil {
				for _, stmt := range block.Statements.Nodes {
					body = append(body, stmt)
				}
			}
		} else {
			// Arrow function with expression body
			body = []*ast.Node{bodyNode}
		}
	}

	var parameters []Parameter
	var typeParameters []*ast.Node

	// Extract parameters and type parameters.
	params := getFunctionParameters(node)
	if params != nil {
		for _, p := range params.Nodes {
			param := p.AsParameterDeclaration()
			parameters = append(parameters, Parameter{
				Name:        parameterName(param.Name()),
				Node:        p,
				Initializer: param.Initializer,
				Type:        param.Type,
			})
		}
	}

	tps := getFunctionTypeParameters(node)
	if tps != nil {
		for _, tp := range tps.Nodes {
			typeParameters = append(typeParameters, tp)
		}
	}

	return &FunctionDefinition{
		Node:           node,
		Body:           body,
		Parameters:     parameters,
		TypeParameters: typeParameters,
		SignatureCount: 0, // Signature count requires type checker; not critical for our use case
	}
}

// getFunctionParameters returns the ParameterList for function-like nodes.
func getFunctionParameters(node *ast.Node) *ast.ParameterList {
	switch {
	case ast.IsFunctionDeclaration(node):
		return node.AsFunctionDeclaration().Parameters
	case ast.IsMethodDeclaration(node):
		return node.AsMethodDeclaration().Parameters
	case ast.IsFunctionExpression(node):
		return node.AsFunctionExpression().Parameters
	case ast.IsArrowFunction(node):
		return node.AsArrowFunction().Parameters
	}
	return nil
}

// getFunctionTypeParameters returns the TypeParameterList for function-like nodes.
func getFunctionTypeParameters(node *ast.Node) *ast.TypeParameterList {
	switch {
	case ast.IsFunctionDeclaration(node):
		return node.AsFunctionDeclaration().TypeParameters
	case ast.IsMethodDeclaration(node):
		return node.AsMethodDeclaration().TypeParameters
	case ast.IsFunctionExpression(node):
		return node.AsFunctionExpression().TypeParameters
	case ast.IsArrowFunction(node):
		return node.AsArrowFunction().TypeParameters
	}
	return nil
}

// GetGenericArityOfClass returns the number of type parameters of the class, or -1.
func (h *TypeScriptReflectionHost) GetGenericArityOfClass(clazz ClassDeclaration) int {
	if !ast.IsClassDeclaration(clazz) {
		return -1
	}
	cd := clazz.AsClassDeclaration()
	if cd.TypeParameters == nil {
		return 0
	}
	return len(cd.TypeParameters.Nodes)
}

// GetVariableValue returns the initializer of a variable declaration.
func (h *TypeScriptReflectionHost) GetVariableValue(declaration *ast.Node) *ast.Node {
	if !ast.IsVariableDeclaration(declaration) {
		return nil
	}
	return declaration.AsVariableDeclaration().Initializer
}

// IsStaticallyExported checks if a declaration is statically exported from its file.
func (h *TypeScriptReflectionHost) IsStaticallyExported(decl *ast.Node) bool {
	topLevel := decl
	if ast.IsVariableDeclaration(decl) {
		parent := decl.Parent
		if parent != nil && ast.IsVariableDeclarationList(parent) {
			topLevel = parent.Parent
		}
	}

	// Check for direct export modifier.
	if ast.HasSyntacticModifier(topLevel, ast.ModifierFlagsExport) {
		return true
	}

	// Check if the top-level node is in the source file's exports (indirect export).
	if topLevel.Parent == nil || !ast.IsSourceFile(topLevel.Parent) {
		return false
	}

	sf := ast.GetSourceFileOfNode(decl)
	if sf == nil {
		return false
	}
	localExports := h.getLocalExportedDeclarationsOfSourceFile(sf)
	return localExports[decl]
}

// getLocalExportedDeclarationsOfSourceFile returns a set of locally declared
// nodes that are exported (directly or indirectly) from the given source file.
func (h *TypeScriptReflectionHost) getLocalExportedDeclarationsOfSourceFile(file *ast.SourceFile) map[*ast.Node]bool {
	if cached, ok := h.localExportsCache[file]; ok {
		return cached
	}

	exportSet := make(map[*ast.Node]bool)
	h.localExportsCache[file] = exportSet

	sfSymbol := h.checker.GetSymbolAtLocation(file.AsNode())
	if sfSymbol == nil || sfSymbol.Exports == nil {
		return exportSet
	}

	for _, exportedSymbol := range sfSymbol.Exports {
		sym := exportedSymbol
		// Unwrap alias (e.g., from `export {Foo}`)
		if sym.Flags&ast.SymbolFlagsAlias != 0 {
			sym = h.checker.GetAliasedSymbol(sym)
		}
		if sym.ValueDeclaration != nil && ast.GetSourceFileOfNode(sym.ValueDeclaration) == file {
			exportSet[sym.ValueDeclaration] = true
		}
	}
	return exportSet
}

// getDeclarationOfSymbol resolves a symbol to its Declaration.
func (h *TypeScriptReflectionHost) getDeclarationOfSymbol(symbol *ast.Symbol, originalId *ast.Node) *Declaration {
	// Handle ShorthandPropertyAssignment
	var valueDeclaration *ast.Node
	if symbol.ValueDeclaration != nil {
		valueDeclaration = symbol.ValueDeclaration
	} else if len(symbol.Declarations) > 0 {
		valueDeclaration = symbol.Declarations[0]
	}

	if valueDeclaration != nil && ast.IsShorthandPropertyAssignment(valueDeclaration) {
		spa := valueDeclaration.AsShorthandPropertyAssignment()
		shorthandSymbol := h.checker.GetSymbolAtLocation(spa.Name())
		if shorthandSymbol == nil {
			return nil
		}
		return h.getDeclarationOfSymbol(shorthandSymbol, originalId)
	}

	if valueDeclaration != nil && ast.IsExportSpecifier(valueDeclaration) {
		// Look up the symbol for the exported name.
		spec := valueDeclaration.AsExportSpecifier()
		var targetName *ast.Node
		if spec.PropertyName != nil {
			targetName = spec.PropertyName
		} else {
			targetName = spec.Name()
		}
		targetSymbol := h.checker.GetSymbolAtLocation(targetName)
		if targetSymbol == nil {
			return nil
		}
		return h.getDeclarationOfSymbol(targetSymbol, originalId)
	}

	var importInfo *Import
	if originalId != nil {
		importInfo = h.GetImportOfIdentifier(originalId)
	}

	// Follow all aliases.
	for symbol.Flags&ast.SymbolFlagsAlias != 0 {
		symbol = h.checker.GetAliasedSymbol(symbol)
	}

	if symbol.ValueDeclaration != nil {
		return &Declaration{
			Node:      symbol.ValueDeclaration,
			ViaModule: h.viaModule(symbol.ValueDeclaration, originalId, importInfo),
		}
	}
	if len(symbol.Declarations) > 0 {
		return &Declaration{
			Node:      symbol.Declarations[0],
			ViaModule: h.viaModule(symbol.Declarations[0], originalId, importInfo),
		}
	}
	return nil
}

// viaModule computes the module that a declaration was imported from.
func (h *TypeScriptReflectionHost) viaModule(declaration *ast.Node, originalId *ast.Node, importInfo *Import) string {
	if importInfo == nil && originalId != nil {
		// Declaration is from a different file (ambient import).
		if ast.GetSourceFileOfNode(declaration) != ast.GetSourceFileOfNode(originalId) {
			return AmbientImport
		}
	}
	if importInfo != nil && importInfo.From != "" && !strings.HasPrefix(importInfo.From, ".") {
		return importInfo.From
	}
	return ""
}

// --- Standalone utility functions ---

// ReflectNameOfDeclaration returns the name identifier text of a declaration, or empty string.
func ReflectNameOfDeclaration(decl *ast.Node) string {
	id := ReflectIdentifierOfDeclaration(decl)
	if id != nil {
		return id.AsIdentifier().Text
	}
	return ""
}

// ReflectIdentifierOfDeclaration returns the name identifier of a class or function declaration.
func ReflectIdentifierOfDeclaration(decl *ast.Node) *ast.Node {
	if ast.IsClassDeclaration(decl) || ast.IsFunctionDeclaration(decl) {
		name := decl.Name()
		if name != nil && ast.IsIdentifier(name) {
			return name
		}
	} else if ast.IsVariableDeclaration(decl) {
		vd := decl.AsVariableDeclaration()
		if ast.IsIdentifier(vd.Name()) {
			return vd.Name()
		}
	}
	return nil
}

// FilterToMembersWithDecorator returns class members that have a decorator with
// the given name (and optional module).
func FilterToMembersWithDecorator(members []ClassMember, name string, module string) []struct {
	Member     ClassMember
	Decorators []Decorator
} {
	var result []struct {
		Member     ClassMember
		Decorators []Decorator
	}
	for _, member := range members {
		if member.IsStatic {
			continue
		}
		if member.Decorators == nil {
			continue
		}
		var matchedDecorators []Decorator
		for _, dec := range member.Decorators {
			if dec.Import != nil {
				if dec.Import.Name == name && (module == "" || dec.Import.From == module) {
					matchedDecorators = append(matchedDecorators, dec)
				}
			} else {
				if dec.Name == name && module == "" {
					matchedDecorators = append(matchedDecorators, dec)
				}
			}
		}
		if len(matchedDecorators) > 0 {
			result = append(result, struct {
				Member     ClassMember
				Decorators []Decorator
			}{member, matchedDecorators})
		}
	}
	return result
}

// FindMember finds a class member by name and static flag.
func FindMember(members []ClassMember, name string, isStatic bool) *ClassMember {
	for i := range members {
		if members[i].IsStatic == isStatic && members[i].Name == name {
			return &members[i]
		}
	}
	return nil
}

// ReflectObjectLiteral converts an ObjectLiteralExpression to a map of property name -> expression.
func ReflectObjectLiteral(node *ast.Node) map[string]*ast.Node {
	if !ast.IsObjectLiteralExpression(node) {
		return nil
	}
	result := make(map[string]*ast.Node)
	ole := node.AsObjectLiteralExpression()
	if ole.Properties == nil {
		return result
	}
	for _, prop := range ole.Properties.Nodes {
		switch {
		case ast.IsPropertyAssignment(prop):
			pa := prop.AsPropertyAssignment()
			name := propertyNameToString(pa.Name())
			if name != "" {
				result[name] = pa.Initializer
			}
		case ast.IsShorthandPropertyAssignment(prop):
			spa := prop.AsShorthandPropertyAssignment()
			result[spa.Name().AsIdentifier().Text] = spa.Name()
		}
	}
	return result
}

// propertyNameToString converts a PropertyName node to a string.
func propertyNameToString(node *ast.Node) string {
	switch {
	case ast.IsIdentifier(node):
		return node.AsIdentifier().Text
	case ast.IsStringLiteral(node):
		return node.AsStringLiteral().Text
	case ast.IsNumericLiteral(node):
		return node.AsNumericLiteral().Text
	}
	return ""
}

// GetContainingImportDeclaration walks up the AST to find the enclosing ImportDeclaration.
func GetContainingImportDeclaration(node *ast.Node) *ast.Node {
	return getContainingImportDeclaration(node)
}

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

// getExportedName returns the name by which a declaration is exported.
func getExportedName(decl *ast.Node, originalId *ast.Node) string {
	if ast.IsImportSpecifier(decl) {
		spec := decl.AsImportSpecifier()
		if spec.PropertyName != nil {
			return spec.PropertyName.Text()
		}
		return spec.Name().Text()
	}
	if originalId != nil && ast.IsIdentifier(originalId) {
		return originalId.AsIdentifier().Text
	}
	return ""
}

// getQualifiedNameRoot walks left through a qualified name chain to find the leftmost identifier.
func getQualifiedNameRoot(qualifiedName *ast.Node) *ast.Node {
	for ast.IsQualifiedName(qualifiedName) {
		qualifiedName = qualifiedName.AsQualifiedName().Left
	}
	if ast.IsIdentifier(qualifiedName) {
		return qualifiedName
	}
	return nil
}

// getFarLeftIdentifier walks left through a PropertyAccessExpression chain to the leftmost identifier.
func getFarLeftIdentifier(propertyAccess *ast.Node) *ast.Node {
	for ast.IsPropertyAccessExpression(propertyAccess.AsPropertyAccessExpression().Expression) {
		propertyAccess = propertyAccess.AsPropertyAccessExpression().Expression
	}
	expr := propertyAccess.AsPropertyAccessExpression().Expression
	if ast.IsIdentifier(expr) {
		return expr
	}
	return nil
}

// ReflectTypeEntityToDeclaration resolves a type entity name to its declaration.
// This is used to convert type references to declarations.
func ReflectTypeEntityToDeclaration(typNode *ast.Node, chk *checker.Checker) (*ast.Node, string, error) {
	realSymbol := chk.GetSymbolAtLocation(typNode)
	if realSymbol == nil {
		return nil, "", &TypeEntityToDeclarationError{msg: "Cannot resolve type entity to symbol"}
	}

	for realSymbol.Flags&ast.SymbolFlagsAlias != 0 {
		realSymbol = chk.GetAliasedSymbol(realSymbol)
	}

	var node *ast.Node
	if realSymbol.ValueDeclaration != nil {
		node = realSymbol.ValueDeclaration
	} else if len(realSymbol.Declarations) == 1 {
		node = realSymbol.Declarations[0]
	} else {
		return nil, "", &TypeEntityToDeclarationError{msg: "Cannot resolve type entity symbol to declaration"}
	}

	if ast.IsQualifiedName(typNode) {
		qn := typNode.AsQualifiedName()
		if !ast.IsIdentifier(qn.Left) {
			return nil, "", &TypeEntityToDeclarationError{msg: "Cannot handle qualified name with non-identifier lhs"}
		}
		symbol := chk.GetSymbolAtLocation(qn.Left)
		if symbol == nil || len(symbol.Declarations) != 1 {
			return nil, "", &TypeEntityToDeclarationError{msg: "Cannot resolve qualified type entity lhs to symbol"}
		}
		decl := symbol.Declarations[0]
		if ast.IsNamespaceImport(decl) {
			// decl → ImportClause → ImportDeclaration
			importDecl := decl.Parent.Parent
			if !ast.IsImportDeclaration(importDecl) {
				return nil, "", &TypeEntityToDeclarationError{msg: "Cannot find import declaration"}
			}
			moduleSpecifier := importDecl.AsImportDeclaration().ModuleSpecifier
			if moduleSpecifier == nil || !ast.IsStringLiteral(moduleSpecifier) {
				return nil, "", &TypeEntityToDeclarationError{msg: "Module specifier is not a string"}
			}
			return node, moduleSpecifier.AsStringLiteral().Text, nil
		}
		return node, "", nil
	}

	return node, "", nil
}

// TypeEntityToDeclarationError is returned when a type entity cannot be converted to a declaration.
type TypeEntityToDeclarationError struct {
	msg string
}

func (e *TypeEntityToDeclarationError) Error() string {
	return e.msg
}
