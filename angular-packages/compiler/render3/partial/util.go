package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

// ToOptionalLiteralArray creates an array literal expression from the given array,
// mapping all values to an expression using the provided mapping function.
// If the array is empty or null, then nil is returned.
func ToOptionalLiteralArray[T any](values []T, mapper func(T) output.Expression) *output.LiteralArrayExpr {
	if len(values) == 0 {
		return nil
	}
	var entries []output.Expression
	for _, value := range values {
		entries = append(entries, mapper(value))
	}
	return render3.LiteralArr(entries)
}

// ToOptionalLiteralMap creates an object literal expression from the given map,
// mapping all values to an expression using the provided mapping function.
// If the map has no keys, then nil is returned.
func ToOptionalLiteralMap[T any](obj map[string]T, mapper func(T) output.Expression) *output.LiteralMapExpr {
	if len(obj) == 0 {
		return nil
	}
	var entries []struct {
		Key    string
		Quoted bool
		Value  output.Expression
	}
	for key, value := range obj {
		entries = append(entries, struct {
			Key    string
			Quoted bool
			Value  output.Expression
		}{
			Key:    key,
			Value:  mapper(value),
			Quoted: true,
		})
	}
	return render3.LiteralMapFromEntries(entries)
}

func CompileDependencies(deps interface{}) output.Expression {
	if depsStr, ok := deps.(string); ok && depsStr == "invalid" {
		return render3.LiteralExpr("invalid")
	} else if deps == nil {
		return render3.LiteralExpr(nil)
	} else if depsList, ok := deps.([]render3.R3DependencyMetadata); ok {
		var entries []output.Expression
		for _, dep := range depsList {
			entries = append(entries, CompileDependency(dep))
		}
		return render3.LiteralArr(entries)
	}
	return render3.LiteralExpr(nil)
}

func CompileDependency(dep render3.R3DependencyMetadata) *output.LiteralMapExpr {
	depMeta := render3.NewDefinitionMap()
	depMeta.Set("token", dep.Token)
	if dep.AttributeNameType != nil {
		depMeta.Set("attribute", render3.LiteralExpr(true))
	}
	if dep.Host {
		depMeta.Set("host", render3.LiteralExpr(true))
	}
	if dep.Optional {
		depMeta.Set("optional", render3.LiteralExpr(true))
	}
	if dep.Self {
		depMeta.Set("self", render3.LiteralExpr(true))
	}
	if dep.SkipSelf {
		depMeta.Set("skipSelf", render3.LiteralExpr(true))
	}
	return depMeta.ToLiteralMap()
}
