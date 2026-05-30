package reflection

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// Decorator holds metadata extracted from a decorator applied to a declaration.
type Decorator struct {
	// Name by which the decorator was invoked.
	Name string
	// Identifier which refers to the decorator in user's code.
	Identifier *ast.Node
	// Import info for where the decorator was imported from, or nil if locally declared.
	Import *Import
	// Node is the TypeScript AST node for the decorator itself.
	Node *ast.Node
	// Args are the arguments to the decorator call, or nil if not invoked as a function.
	Args []*ast.Node
}

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
	Node           *ast.Node
	Kind           ClassMemberKind
	AccessLevel    ClassMemberAccessLevel
	Type           *ast.Node
	Name           string
	NameNode       *ast.Node // Identifier, PrivateIdentifier, or StringLiteral
	Value          *ast.Node
	Implementation *ast.Node
	IsStatic       bool
	Decorators     []Decorator
}

// TypeValueReferenceKind enumerates types of TypeValueReference.
type TypeValueReferenceKind int

const (
	Local      TypeValueReferenceKind = iota
	Imported
	Unavailable
)

// TypeValueReference is the result of converting a type to a value reference.
type TypeValueReference interface {
	GetKind() TypeValueReferenceKind
}

// LocalTypeValueReference represents a type that resolves to a local expression.
// This covers locally declared types and default-imported types.
type LocalTypeValueReference struct {
	// Expression is the AST expression that refers to the value.
	Expression *ast.Node
	// DefaultImportStatement is the import declaration if this is a default import, or nil.
	DefaultImportStatement *ast.Node
}

func (l *LocalTypeValueReference) GetKind() TypeValueReferenceKind { return Local }

// ImportedTypeValueReference represents a type that was imported by name from a module.
type ImportedTypeValueReference struct {
	// ModuleName is the module specifier string.
	ModuleName string
	// ImportedName is the exported name from the module.
	ImportedName string
	// ValueDeclaration is the declaration node (may be nil).
	ValueDeclaration *ast.Node
}

func (i *ImportedTypeValueReference) GetKind() TypeValueReferenceKind { return Imported }

// UnavailableTypeValueReference represents a type reference that cannot be converted to a value.
type UnavailableTypeValueReference struct {
	// Reason is a short description of why the reference is unavailable.
	Reason string
}

func (u *UnavailableTypeValueReference) GetKind() TypeValueReferenceKind { return Unavailable }

// CtorParameter holds information about a constructor parameter.
type CtorParameter struct {
	Name               string
	NameNode           *ast.Node
	TypeValueReference TypeValueReference
	TypeNode           *ast.Node
	Decorators         []Decorator
}

// Parameter holds information about a function parameter.
type Parameter struct {
	Name        string
	Node        *ast.Node
	Initializer *ast.Node
	Type        *ast.Node
}

// FunctionDefinition holds information about a function-like declaration.
type FunctionDefinition struct {
	Node           *ast.Node
	Body           []*ast.Node
	Parameters     []Parameter
	TypeParameters []*ast.Node
	SignatureCount int
}

// Import holds information about an import.
type Import struct {
	Name string
	From string
	Node *ast.Node
}

// Declaration holds a reference to a declaration node.
type Declaration struct {
	// ViaModule is the module from which the declaration was imported. Empty for local declarations.
	ViaModule string
	Node      *ast.Node
}

// ClassElementList is an alias for convenience.
type ClassElementList = ast.ClassElementList

// HeritageClauseList is an alias for convenience.
type HeritageClauseList = ast.HeritageClauseList

// ReflectionHost abstracts reflection operations on a TypeScript AST.
type ReflectionHost interface {
	GetDecoratorsOfDeclaration(declaration *ast.Node) []Decorator
	GetMembersOfClass(clazz *ast.Node) []ClassMember
	GetConstructorParameters(clazz *ast.Node) []CtorParameter
	GetDefinitionOfFunction(fn *ast.Node) *FunctionDefinition
	GetImportOfIdentifier(id *ast.Node) *Import
	GetDeclarationOfIdentifier(id *ast.Node) *Declaration
	GetExportsOfModule(module *ast.Node) map[string]Declaration
	IsClass(node *ast.Node) bool
	HasBaseClass(clazz *ast.Node) bool
	GetBaseClassExpression(clazz *ast.Node) *ast.Node
	// GetGenericArityOfClass returns the number of generic type parameters, or -1 if unknown.
	GetGenericArityOfClass(clazz *ast.Node) int
	GetVariableValue(declaration *ast.Node) *ast.Node
	IsStaticallyExported(decl *ast.Node) bool
}
