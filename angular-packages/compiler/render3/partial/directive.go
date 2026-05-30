package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

func CompileDeclareDirectiveFromMetadata(meta render3.R3DirectiveMetadata) render3.R3CompiledExpression {
	definitionMap := CreateDirectiveDefinitionMap(meta)

	expression := render3.ImportExpr(*render3.Identifiers.DeclareDirective).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
	typeExpr := output.INFERRED_TYPE // render3.CreateDirectiveType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func CreateDirectiveDefinitionMap(meta render3.R3DirectiveMetadata) *render3.DefinitionMap {
	definitionMap := render3.NewDefinitionMap()
	minVersion := getMinimumVersionForPartialOutput(meta)

	definitionMap.Set("minVersion", render3.LiteralExpr(minVersion))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))

	definitionMap.Set("type", meta.Type.Value)

	if meta.IsStandalone {
		definitionMap.Set("isStandalone", render3.LiteralExpr(meta.IsStandalone))
	}
	if meta.IsSignal {
		definitionMap.Set("isSignal", render3.LiteralExpr(true))
	}

	if meta.Selector != nil && *meta.Selector != "" {
		definitionMap.Set("selector", render3.LiteralExpr(*meta.Selector))
	}

	if needsNewInputPartialOutput(meta) {
		definitionMap.Set("inputs", createInputsPartialMetadata(meta.Inputs))
	} else {
		definitionMap.Set("inputs", legacyInputsPartialMetadata(meta.Inputs))
	}

	definitionMap.Set("outputs", render3.LiteralExpr(nil)) // Stub: render3.ConditionallyCreateDirectiveBindingLiteral(meta.Outputs)

	hostMetadata := compileHostMetadata(meta.Host)
	if hostMetadata != nil {
		definitionMap.Set("host", hostMetadata)
	}

	if meta.Providers != nil {
		definitionMap.Set("providers", meta.Providers.(output.Expression))
	}

	if len(meta.Queries) > 0 {
		var queries []output.Expression
		for _, q := range meta.Queries {
			queries = append(queries, compileQuery(q))
		}
		definitionMap.Set("queries", render3.LiteralArr(queries))
	}

	if len(meta.ViewQueries) > 0 {
		var viewQueries []output.Expression
		for _, q := range meta.ViewQueries {
			viewQueries = append(viewQueries, compileQuery(q))
		}
		definitionMap.Set("viewQueries", render3.LiteralArr(viewQueries))
	}

	if len(meta.ExportAs) > 0 {
		definitionMap.Set("exportAs", render3.LiteralExpr(meta.ExportAs))
	}

	if meta.UsesInheritance {
		definitionMap.Set("usesInheritance", render3.LiteralExpr(true))
	}

	if meta.Lifecycle.UsesOnChanges {
		definitionMap.Set("usesOnChanges", render3.LiteralExpr(true))
	}

	if meta.ControlCreate != nil {
		var passThroughExpr output.Expression = render3.LiteralExpr(nil)
		if meta.ControlCreate.PassThroughInput != nil {
			passThroughExpr = render3.LiteralExpr(*meta.ControlCreate.PassThroughInput)
		}
		definitionMap.Set("controlCreate", render3.LiteralMapFromEntries([]struct {
			Key    string
			Quoted bool
			Value  output.Expression
		}{
			{Key: "passThroughInput", Value: passThroughExpr, Quoted: false},
		}))
	}

	if len(meta.HostDirectives) > 0 {
		definitionMap.Set("hostDirectives", createHostDirectives(meta.HostDirectives))
	}

	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))

	return definitionMap
}

func getMinimumVersionForPartialOutput(meta render3.R3DirectiveMetadata) string {
	minVersion := "14.0.0"

	hasDecoratorTransformFunctions := false
	for _, input := range meta.Inputs {
		if input.TransformFunction != nil {
			hasDecoratorTransformFunctions = true
			break
		}
	}
	if hasDecoratorTransformFunctions {
		minVersion = "16.1.0"
	}

	if needsNewInputPartialOutput(meta) {
		minVersion = "17.1.0"
	}

	hasSignalQuery := false
	for _, q := range meta.Queries {
		if q.IsSignal {
			hasSignalQuery = true
			break
		}
	}
	for _, q := range meta.ViewQueries {
		if q.IsSignal {
			hasSignalQuery = true
			break
		}
	}
	if hasSignalQuery {
		minVersion = "17.2.0"
	}

	return minVersion
}

func needsNewInputPartialOutput(meta render3.R3DirectiveMetadata) bool {
	for _, input := range meta.Inputs {
		if input.IsSignal {
			return true
		}
	}
	return false
}

func compileQuery(query render3.R3QueryMetadata) *output.LiteralMapExpr {
	meta := render3.NewDefinitionMap()
	meta.Set("propertyName", render3.LiteralExpr(query.PropertyName))
	if query.First {
		meta.Set("first", render3.LiteralExpr(true))
	}

	if predicateList, ok := query.Predicate.([]string); ok {
		meta.Set("predicate", render3.LiteralExpr(predicateList))
	} else if predicateExpr, ok := query.Predicate.(render3.MaybeForwardRefExpression); ok {
		meta.Set("predicate", render3.ConvertFromMaybeForwardRefExpression(predicateExpr))
	} else {
		// Fallback
		if expr, ok := query.Predicate.(output.Expression); ok {
			meta.Set("predicate", expr)
		}
	}

	if !query.EmitDistinctChangesOnly {
		meta.Set("emitDistinctChangesOnly", render3.LiteralExpr(false))
	}

	if query.Descendants {
		meta.Set("descendants", render3.LiteralExpr(true))
	}

	if query.Read != nil {
		meta.Set("read", query.Read.(output.Expression))
	}

	if query.Static {
		meta.Set("static", render3.LiteralExpr(true))
	}

	if query.IsSignal {
		meta.Set("isSignal", render3.LiteralExpr(true))
	}

	return meta.ToLiteralMap()
}

func compileHostMetadata(meta render3.R3HostMetadata) *output.LiteralMapExpr {
	hostMetadata := render3.NewDefinitionMap()

	if attrMap := ToOptionalLiteralMap(meta.Attributes, func(expr render3.Expression) output.Expression { return expr.(output.Expression) }); attrMap != nil {
		hostMetadata.Set("attributes", attrMap)
	}
	if listenerMap := ToOptionalLiteralMap(meta.Listeners, func(s string) output.Expression { return render3.LiteralExpr(s) }); listenerMap != nil {
		hostMetadata.Set("listeners", listenerMap)
	}
	if propMap := ToOptionalLiteralMap(meta.Properties, func(s string) output.Expression { return render3.LiteralExpr(s) }); propMap != nil {
		hostMetadata.Set("properties", propMap)
	}

	if meta.SpecialAttributes.StyleAttr != nil && *meta.SpecialAttributes.StyleAttr != "" {
		hostMetadata.Set("styleAttribute", render3.LiteralExpr(*meta.SpecialAttributes.StyleAttr))
	}
	if meta.SpecialAttributes.ClassAttr != nil && *meta.SpecialAttributes.ClassAttr != "" {
		hostMetadata.Set("classAttribute", render3.LiteralExpr(*meta.SpecialAttributes.ClassAttr))
	}

	if len(hostMetadata.Values) > 0 {
		return hostMetadata.ToLiteralMap()
	}
	return nil
}

func createHostDirectives(hostDirectives []render3.R3HostDirectiveMetadata) *output.LiteralArrayExpr {
	var expressions []output.Expression
	for _, current := range hostDirectives {
		var keys []struct {
			Key    string
			Quoted bool
			Value  output.Expression
		}

		var dirValue output.Expression
		if current.IsForwardReference {
			dirValue = render3.GenerateForwardRef(current.Directive.Type)
		} else {
			dirValue = current.Directive.Type
		}

		keys = append(keys, struct {
			Key    string
			Quoted bool
			Value  output.Expression
		}{
			Key:    "directive",
			Value:  dirValue,
			Quoted: false,
		})

		if current.Inputs != nil {
			keys = append(keys, struct {
				Key    string
				Quoted bool
				Value  output.Expression
			}{
				Key:    "inputs",
				Value:  output.TYPED_NULL_EXPR,
				Quoted: false,
			})
		}

		if current.Outputs != nil {
			keys = append(keys, struct {
				Key    string
				Quoted bool
				Value  output.Expression
			}{
				Key:    "outputs",
				Value:  output.TYPED_NULL_EXPR,
				Quoted: false,
			})
		}

		expressions = append(expressions, render3.LiteralMapFromEntries(keys))
	}

	return render3.LiteralArr(expressions)
}

func createInputsPartialMetadata(inputs map[string]render3.R3InputMetadata) output.Expression {
	if len(inputs) == 0 {
		return render3.LiteralExpr(nil)
	}

	var entries []struct {
		Key    string
		Quoted bool
		Value  output.Expression
	}
	for declaredName, value := range inputs {
		var transformFnExpr output.Expression
		if value.TransformFunction != nil {
			transformFnExpr = value.TransformFunction.(output.Expression)
		} else {
			transformFnExpr = output.NULL_EXPR
		}

		entries = append(entries, struct {
			Key    string
			Quoted bool
			Value  output.Expression
		}{
			Key:    declaredName,
			Quoted: render3.IsUnsafeObjectKey(declaredName),
			Value: render3.LiteralMapFromEntries([]struct {
				Key    string
				Quoted bool
				Value  output.Expression
			}{
				{Key: "classPropertyName", Quoted: false, Value: render3.LiteralExpr(value.ClassPropertyName)},
				{Key: "publicName", Quoted: false, Value: render3.LiteralExpr(value.BindingPropertyName)},
				{Key: "isSignal", Quoted: false, Value: render3.LiteralExpr(value.IsSignal)},
				{Key: "isRequired", Quoted: false, Value: render3.LiteralExpr(value.Required)},
				{Key: "transformFunction", Quoted: false, Value: transformFnExpr},
			}),
		})
	}

	return render3.LiteralMapFromEntries(entries)
}

func legacyInputsPartialMetadata(inputs map[string]render3.R3InputMetadata) output.Expression {
	if len(inputs) == 0 {
		return render3.LiteralExpr(nil)
	}

	var entries []struct {
		Key    string
		Quoted bool
		Value  output.Expression
	}
	for declaredName, value := range inputs {
		publicName := value.BindingPropertyName
		differentDeclaringName := publicName != declaredName

		var result output.Expression
		if differentDeclaringName || value.TransformFunction != nil {
			arr := []output.Expression{render3.LiteralExpr(publicName), render3.LiteralExpr(declaredName)}
			if value.TransformFunction != nil {
				arr = append(arr, value.TransformFunction.(output.Expression))
			}
			result = render3.LiteralArr(arr)
		} else {
			result = render3.LiteralExpr(publicName)
		}

		entries = append(entries, struct {
			Key    string
			Quoted bool
			Value  output.Expression
		}{
			Key:    declaredName,
			Quoted: render3.IsUnsafeObjectKey(declaredName),
			Value:  result,
		})
	}

	return render3.LiteralMapFromEntries(entries)
}
