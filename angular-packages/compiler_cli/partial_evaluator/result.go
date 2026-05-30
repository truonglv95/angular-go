package partial_evaluator

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// ResolvedValue represents a value resulting from static resolution.
// In Go, this is an interface{} / any since it can be primitive, map, array, EnumValue, etc.
type ResolvedValue any

// ResolvedValueArray is an array of ResolvedValues.
type ResolvedValueArray []ResolvedValue

// ResolvedValueMap is a map of strings to ResolvedValues.
type ResolvedValueMap map[string]ResolvedValue

// EnumValue represents a value member of an enumeration.
type EnumValue struct {
	// EnumRef is a reference to the enumeration itself.
	EnumRef  *ast.Node
	Name     string
	Resolved ResolvedValue
}

// ResolvedModule represents a collection of publicly exported declarations from a module.
type ResolvedModule struct {
	Exports  map[string]reflection.Declaration
	Evaluate func(decl reflection.Declaration) ResolvedValue
}

func (rm *ResolvedModule) GetExport(name string) ResolvedValue {
	decl, exists := rm.Exports[name]
	if !exists {
		return nil
	}
	return rm.Evaluate(decl)
}

func (rm *ResolvedModule) GetExports() ResolvedValueMap {
	m := make(ResolvedValueMap)
	for name, decl := range rm.Exports {
		m[name] = rm.Evaluate(decl)
	}
	return m
}

// KnownFn represents an implementation of a known function that can be statically evaluated.
type KnownFn interface {
	Evaluate(node *ast.CallExpression, args ResolvedValueArray) ResolvedValue
}

// DynamicValue represents a value that could not be statically evaluated.
// The Reason can be another DynamicValue (chained) or a plain string or any ResolvedValue.
type DynamicValue struct {
	Node   *ast.Node
	Reason any // string, *DynamicValue, or ResolvedValue

	// Flags for why this is dynamic
	fromUnsupportedSyntax     bool
	fromInvalidExpressionType bool
	fromDynamicInput          bool
	fromExternalReference     bool
	fromComplexFunctionCall   bool
	fromDynamicString         bool
	fromDynamicType           bool
}

func (d *DynamicValue) IsDynamic() bool { return true }

// IsFromUnsupportedSyntax returns true if the value is from unsupported syntax.
func (d *DynamicValue) IsFromUnsupportedSyntax() bool { return d.fromUnsupportedSyntax }

// IsFromInvalidExpressionType returns true if the value is from an invalid expression type.
func (d *DynamicValue) IsFromInvalidExpressionType() bool { return d.fromInvalidExpressionType }

// IsFromDynamicInput returns true if the value derives from a dynamic input.
func (d *DynamicValue) IsFromDynamicInput() bool { return d.fromDynamicInput }

// IsFromExternalReference returns true if the value is from an external reference.
func (d *DynamicValue) IsFromExternalReference() bool { return d.fromExternalReference }

// IsFromComplexFunctionCall returns true if value is from a complex function (multiple statements).
func (d *DynamicValue) IsFromComplexFunctionCall() bool { return d.fromComplexFunctionCall }

// IsFromDynamicString returns true if the value is from a dynamic string interpolation.
func (d *DynamicValue) IsFromDynamicString() bool { return d.fromDynamicString }

// IsFromDynamicType returns true if the value is from a type that cannot be resolved.
func (d *DynamicValue) IsFromDynamicType() bool { return d.fromDynamicType }

// SyntheticValue represents a value that is synthesized during evaluation.
type SyntheticValue struct {
	Value any
}
