package reflection

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// --- Decorator ---

// Decorator holds metadata extracted from a decorator applied to a declaration.
type Decorator struct {
	// Name by which the decorator was invoked.
	Name string
	// Identifier which refers to the decorator in the user's code.
	Identifier *ast.Node
	// Import by which the decorator was imported, or nil if locally declared.
	Import *Import
	// Node is the TypeScript AST node for the decorator.
	Node *ast.Node
	// Args are the arguments to the decorator call, or nil if not called.
	Args []*ast.Node
}

// isDecoratorIdentifier checks whether the expression is a simple identifier or
// a property access expression of the form `a.B` where both sides are identifiers.
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

// --- ClassMember ---

// ClassMemberKind enumerates the possible kinds of class members.
type ClassMemberKind int

const (
	Constructor ClassMemberKind = iota
	Getter
	Setter
	Property
	Method
)

// ClassMemberAccessLevel enumerates access levels for class members.
type ClassMemberAccessLevel int

const (
	PublicWritable    ClassMemberAccessLevel = iota
	PublicReadonly
	Protected
	Private
	EcmaScriptPrivate
)

// ClassMember holds static information about a class member.
type ClassMember struct {
	// Node is the AST element for the class member.
	Node *ast.Node
	// Kind is the type of class member.
	Kind ClassMemberKind
	// AccessLevel is the access level.
	AccessLevel ClassMemberAccessLevel
	// Type is the type annotation node (optional).
	Type *ast.Node
	// Name of the class member.
	Name string
	// NameNode is the identifier, private identifier or string literal for the name.
	NameNode *ast.Node
	// Value is the initializer expression (for Property members).
	Value *ast.Node
	// Implementation is the declaration that implements this member.
	Implementation *ast.Node
	// IsStatic indicates if the member is static.
	IsStatic bool
	// Decorators on this member.
	Decorators []Decorator
}

// --- TypeValueReference ---

// TypeValueReferenceKind enumerates types of TypeValueReference.
type TypeValueReferenceKind int

const (
	LOCAL      TypeValueReferenceKind = iota
	IMPORTED
	UNAVAILABLE
)

// ValueUnavailableKind enumerates reasons a type value is unavailable.
type ValueUnavailableKind int

const (
	MissingTypeKind        ValueUnavailableKind = iota
	NoValueDeclarationKind
	TypeOnlyImportKind
	UnknownReferenceKind
	NamespaceKind
	UnsupportedKind
)

// TypeValueReference is the result of converting a type to a value reference.
type TypeValueReference interface {
	GetKind() TypeValueReferenceKind
}

// LocalTypeValueRef is a type reference that refers to a locally accessible value.
type LocalTypeValueRef struct {
	// Expression is the identifier/property-access expression that refers to the value.
	Expression *ast.Node
	// DefaultImportStatement is the import declaration for default imports, or nil.
	DefaultImportStatement *ast.Node
}

func (r LocalTypeValueRef) GetKind() TypeValueReferenceKind { return LOCAL }

// ImportedTypeValueRef is a type reference that refers to an imported value.
type ImportedTypeValueRef struct {
	// ModuleName is the module from which the value is imported.
	ModuleName string
	// ImportedName is the name of the exported binding.
	ImportedName string
	// NestedPath is any additional property access path after the import.
	NestedPath []string
	// ValueDeclaration is the declaration node for the imported value.
	ValueDeclaration *ast.Node
}

func (r ImportedTypeValueRef) GetKind() TypeValueReferenceKind { return IMPORTED }

// UnavailableTypeValueRef is a type reference that cannot be used as a value.
type UnavailableTypeValueRef struct {
	// Kind of unavailability.
	Kind ValueUnavailableKind
	// Node is the primary node associated with the error.
	Node *ast.Node
	// ExtraNode is an additional node (e.g., the declaration node for NO_VALUE_DECLARATION).
	ExtraNode *ast.Node
}

func (r UnavailableTypeValueRef) GetKind() TypeValueReferenceKind { return UNAVAILABLE }

// --- CtorParameter ---

// CtorParameter holds information about a constructor parameter.
type CtorParameter struct {
	// Name of the parameter, or empty if it could not be determined.
	Name string
	// NameNode is the binding name AST node.
	NameNode *ast.Node
	// TypeValueReference is how to use the parameter's type as a value.
	TypeValueReference TypeValueReference
	// TypeNode is the original type annotation.
	TypeNode *ast.Node
	// Decorators on the parameter.
	Decorators []Decorator
}

// --- Parameter ---

// Parameter holds information about a function parameter.
type Parameter struct {
	// Name of the parameter.
	Name string
	// Node is the AST parameter node.
	Node *ast.Node
	// Initializer is the default value expression, or nil.
	Initializer *ast.Node
	// Type is the type annotation, or nil.
	Type *ast.Node
}

// --- FunctionDefinition ---

// FunctionDefinition holds information about a function-like declaration.
type FunctionDefinition struct {
	// Node is the function AST node.
	Node *ast.Node
	// Body is the list of statements in the body, or nil if there's no body.
	Body []*ast.Node
	// Parameters is the list of parameters.
	Parameters []Parameter
	// TypeParameters is the list of type parameter nodes.
	TypeParameters []*ast.Node
	// SignatureCount is the number of call signatures in the function type.
	SignatureCount int
}

// --- Import ---

// Import holds information about an import of a declaration.
type Import struct {
	// From is the module specifier from which the symbol was imported.
	From string
	// Name is the name by which the symbol was exported from the source module.
	Name string
	// Node is the ImportDeclaration node.
	Node *ast.Node
}

// --- Declaration ---

// Declaration holds a reference to a TypeScript declaration node.
type Declaration struct {
	// Node is the declaration node.
	Node *ast.Node
	// ViaModule is the module that provided this declaration, or empty for local declarations.
	// The special value AmbientImport indicates the declaration is from a different source file
	// via an ambient module.
	ViaModule string
}

// --- ClassDeclaration ---

// ClassDeclaration is an alias for an AST node that is guaranteed to be a named class.
type ClassDeclaration = *ast.Node

// DeclarationNode is an alias for any AST node that represents a declaration.
type DeclarationNode = *ast.Node

// --- ReflectionHost ---

// ReflectionHost abstracts reflection operations on a TypeScript AST.
type ReflectionHost interface {
	GetDecoratorsOfDeclaration(declaration *ast.Node) []Decorator
	GetMembersOfClass(clazz ClassDeclaration) []ClassMember
	GetConstructorParameters(clazz ClassDeclaration) []CtorParameter
	GetDefinitionOfFunction(fn *ast.Node) *FunctionDefinition
	GetImportOfIdentifier(id *ast.Node) *Import
	GetDeclarationOfIdentifier(id *ast.Node) *Declaration
	GetExportsOfModule(module *ast.Node) map[string]Declaration
	IsClass(node *ast.Node) bool
	HasBaseClass(clazz ClassDeclaration) bool
	GetBaseClassExpression(clazz ClassDeclaration) *ast.Node
	// GetGenericArityOfClass returns the number of generic type parameters, or -1 if unknown.
	GetGenericArityOfClass(clazz ClassDeclaration) int
	GetVariableValue(declaration *ast.Node) *ast.Node
	IsStaticallyExported(decl *ast.Node) bool
}
