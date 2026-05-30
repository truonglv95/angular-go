package render3

// DefinitionMap is a helper for building object literal expressions during codegen.
// Port of DefinitionMap from angular/packages/compiler/src/render3/view/pipeline.ts

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// DefinitionMapEntry holds a key-value pair for a definition map.
type DefinitionMapEntry struct {
	Key    string
	Value  output.Expression
	Quoted bool
}

// DefinitionMap builds a literal map for use in Angular def calls.
type DefinitionMap struct {
	Values []DefinitionMapEntry
}

// NewDefinitionMap creates a new empty DefinitionMap.
func NewDefinitionMap() *DefinitionMap {
	return &DefinitionMap{}
}

// Set adds or updates a key in the definition map. If value is nil, it is ignored.
func (m *DefinitionMap) Set(key string, value output.Expression) {
	if value == nil {
		return
	}
	for i := range m.Values {
		if m.Values[i].Key == key {
			m.Values[i].Value = value
			return
		}
	}
	m.Values = append(m.Values, DefinitionMapEntry{Key: key, Value: value, Quoted: false})
}

// ToLiteralMap converts the definition map to a LiteralMapExpr.
func (m *DefinitionMap) ToLiteralMap() *output.LiteralMapExpr {
	entries := make([]output.LiteralMapEntry, len(m.Values))
	for i, v := range m.Values {
		entries[i] = output.NewLiteralMapPropertyAssignment(v.Key, v.Value, v.Quoted)
	}
	return output.NewLiteralMapExpr(entries, nil, nil, nil)
}

// MapLiteral creates a LiteralMapExpr from a map of string keys to expressions.
// Port of mapLiteral from angular/packages/compiler/src/output/map_util.ts
func MapLiteral(obj map[string]output.Expression, quoted bool) output.Expression {
	entries := make([]output.LiteralMapEntry, 0, len(obj))
	for key, value := range obj {
		entries = append(entries, output.NewLiteralMapPropertyAssignment(key, value, quoted))
	}
	return output.NewLiteralMapExpr(entries, nil, nil, nil)
}

// MapLiteralOrdered creates a LiteralMapExpr from an ordered list of key-value pairs.
func MapLiteralOrdered(pairs []struct {
	Key    string
	Value  output.Expression
	Quoted bool
}) output.Expression {
	entries := make([]output.LiteralMapEntry, len(pairs))
	for i, p := range pairs {
		entries[i] = output.NewLiteralMapPropertyAssignment(p.Key, p.Value, p.Quoted)
	}
	return output.NewLiteralMapExpr(entries, nil, nil, nil)
}

// ImportExpr creates an ExternalExpr from an ExternalReference with optional type params.
func ImportExpr(ref output.ExternalReference, typeParams ...output.Type) *output.ExternalExpr {
	if len(typeParams) > 0 {
		return output.NewExternalExpr(ref, nil, typeParams, nil, nil)
	}
	return output.NewExternalExpr(ref, nil, nil, nil, nil)
}

// LiteralExpr creates a LiteralExpr for a value.
func LiteralExpr(value interface{}) *output.LiteralExpr {
	return output.NewLiteralExpr(value, nil, nil, nil)
}

// Variable creates a ReadVarExpr for a name.
func VariableExpr(name string) *output.ReadVarExpr {
	return output.NewReadVarExpr(name, nil, nil, nil)
}

// LiteralArr creates a LiteralArrayExpr.
func LiteralArr(values []output.Expression) *output.LiteralArrayExpr {
	return output.NewLiteralArrayExpr(values, nil, nil, nil)
}

// LiteralMapFromEntries creates a LiteralMapExpr from key/quoted/value triples.
func LiteralMapFromEntries(entries []struct {
	Key    string
	Quoted bool
	Value  output.Expression
}) *output.LiteralMapExpr {
	mapEntries := make([]output.LiteralMapEntry, len(entries))
	for i, e := range entries {
		mapEntries[i] = output.NewLiteralMapPropertyAssignment(e.Key, e.Value, e.Quoted)
	}
	return output.NewLiteralMapExpr(mapEntries, nil, nil, nil)
}

// ExpressionType creates an ExpressionType from an expression.
func ExpressionType(expr output.Expression, typeParams ...output.Type) *output.ExpressionType {
	if len(typeParams) > 0 {
		return output.NewExpressionTypeWithParams(expr, typeParams)
	}
	return output.NewExpressionType(expr)
}

// TypeofExpr creates a TypeofExpr.
func TypeofExpr(expr output.Expression) *output.TypeofExpr {
	return output.NewTypeofExpr(expr, nil, nil, nil)
}
