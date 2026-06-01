package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type PartialDirectiveLinkerVersion1 struct {
	sourceURL string
	code      string
}

func NewPartialDirectiveLinkerVersion1(sourceURL string, code string) *PartialDirectiveLinkerVersion1 {
	return &PartialDirectiveLinkerVersion1{sourceURL: sourceURL, code: code}
}

func (l *PartialDirectiveLinkerVersion1) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	meta, err := ToR3DirectiveMeta(metaObj, l.code, l.sourceURL, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	compiled := render3.CompileDirectiveFromMetadata(meta, constantPool, render3.MakeBindingParser(false))
	return linkedDefinitionFromCompiled(compiled), nil
}

func ToR3DirectiveMeta(metaObj *AstObject, code string, sourceURL string, version string) (render3.R3DirectiveMetadata, error) {
	_ = code
	_ = sourceURL
	typeValue, err := metaObj.GetValue("type")
	if err != nil {
		return render3.R3DirectiveMetadata{}, err
	}
	typeName := typeValue.GetSymbolName()
	if typeName == "" {
		return render3.R3DirectiveMetadata{}, linkerError(typeValue.Node(), "Unsupported type, its name could not be determined.")
	}

	host, err := toHostMetadata(metaObj)
	if err != nil {
		return render3.R3DirectiveMetadata{}, err
	}
	inputs, err := toInputs(metaObj)
	if err != nil {
		return render3.R3DirectiveMetadata{}, err
	}
	outputs, err := toStringMap(metaObj, "outputs")
	if err != nil {
		return render3.R3DirectiveMetadata{}, err
	}
	queries, err := toR3QueryMetadataArray(metaObj, "queries")
	if err != nil {
		return render3.R3DirectiveMetadata{}, err
	}
	viewQueries, err := toR3QueryMetadataArray(metaObj, "viewQueries")
	if err != nil {
		return render3.R3DirectiveMetadata{}, err
	}

	var selector *string
	if metaObj.Has("selector") {
		s, err := metaObj.GetString("selector")
		if err != nil {
			return render3.R3DirectiveMetadata{}, err
		}
		selector = &s
	}

	isStandalone := getDefaultStandaloneValue(version)
	if metaObj.Has("isStandalone") {
		isStandalone, err = metaObj.GetBoolean("isStandalone")
		if err != nil {
			return render3.R3DirectiveMetadata{}, err
		}
	}

	isSignal := false
	if metaObj.Has("isSignal") {
		isSignal, err = metaObj.GetBoolean("isSignal")
		if err != nil {
			return render3.R3DirectiveMetadata{}, err
		}
	}

	exportAs, err := toStringArray(metaObj, "exportAs")
	if err != nil {
		return render3.R3DirectiveMetadata{}, err
	}

	meta := render3.R3DirectiveMetadata{
		Name: typeName,
		Type: render3.R3Reference{
			Value: output.NewWrappedNodeExpr(typeValue.Node(), nil, nil, nil),
		},
		TypeArgumentCount:      0,
		Deps:                   nil,
		Selector:               selector,
		Host:                   host,
		Inputs:                 inputs,
		Outputs:                outputs,
		ExportAs:               exportAs,
		IsStandalone:           isStandalone,
		IsSignal:               isSignal,
		LegacyOptionalChaining: false,
		UsesInheritance:        false,
		Queries:                queries,
		ViewQueries:            viewQueries,
		HostDirectives:         nil,
		Providers:              nil,
	}
	if metaObj.Has("usesInheritance") {
		meta.UsesInheritance, err = metaObj.GetBoolean("usesInheritance")
		if err != nil {
			return render3.R3DirectiveMetadata{}, err
		}
	}
	if metaObj.Has("usesOnChanges") {
		meta.Lifecycle.UsesOnChanges, err = metaObj.GetBoolean("usesOnChanges")
		if err != nil {
			return render3.R3DirectiveMetadata{}, err
		}
	}
	if metaObj.Has("providers") {
		meta.Providers, err = metaObj.GetOpaque("providers")
		if err != nil {
			return render3.R3DirectiveMetadata{}, err
		}
	}
	if metaObj.Has("hostDirectives") {
		meta.HostDirectives, err = extractHostDirectives(metaObj)
		if err != nil {
			return render3.R3DirectiveMetadata{}, err
		}
	}

	return meta, nil
}

func toR3QueryMetadataArray(metaObj *AstObject, field string) ([]render3.R3QueryMetadata, error) {
	if !metaObj.Has(field) {
		return nil, nil
	}

	values, err := metaObj.GetArray(field)
	if err != nil {
		return nil, err
	}

	queries := make([]render3.R3QueryMetadata, 0, len(values))
	for _, value := range values {
		queryObj, err := value.GetObject()
		if err != nil {
			return nil, err
		}

		propertyName, err := queryObj.GetString("propertyName")
		if err != nil {
			return nil, err
		}

		predicateValue, err := queryObj.GetValue("predicate")
		if err != nil {
			return nil, err
		}
		predicate, err := toQueryPredicate(predicateValue)
		if err != nil {
			return nil, err
		}

		query := render3.R3QueryMetadata{
			PropertyName:            propertyName,
			Predicate:               predicate,
			EmitDistinctChangesOnly: true,
		}

		if queryObj.Has("first") {
			query.First, err = queryObj.GetBoolean("first")
			if err != nil {
				return nil, err
			}
		}
		if queryObj.Has("descendants") {
			query.Descendants, err = queryObj.GetBoolean("descendants")
			if err != nil {
				return nil, err
			}
		}
		if queryObj.Has("emitDistinctChangesOnly") {
			query.EmitDistinctChangesOnly, err = queryObj.GetBoolean("emitDistinctChangesOnly")
			if err != nil {
				return nil, err
			}
		}
		if queryObj.Has("read") {
			query.Read, err = queryObj.GetOpaque("read")
			if err != nil {
				return nil, err
			}
		}
		if queryObj.Has("static") {
			query.Static, err = queryObj.GetBoolean("static")
			if err != nil {
				return nil, err
			}
		}
		if queryObj.Has("isSignal") {
			query.IsSignal, err = queryObj.GetBoolean("isSignal")
			if err != nil {
				return nil, err
			}
		}

		queries = append(queries, query)
	}

	return queries, nil
}

func toQueryPredicate(value *AstValue) (interface{}, error) {
	if value.IsArray() {
		values, err := value.GetArray()
		if err != nil {
			return nil, err
		}
		predicates := make([]string, 0, len(values))
		for _, value := range values {
			predicate, err := value.GetString()
			if err != nil {
				return nil, err
			}
			predicates = append(predicates, predicate)
		}
		return predicates, nil
	}

	return render3.CreateMayBeForwardRefExpression(value.GetOpaque(), render3.ForwardRefHandlingNone), nil
}

func toHostMetadata(metaObj *AstObject) (render3.R3HostMetadata, error) {
	host := render3.R3HostMetadata{
		Attributes: map[string]render3.Expression{},
		Listeners:  map[string]string{},
		Properties: map[string]string{},
	}
	if !metaObj.Has("host") {
		return host, nil
	}
	hostObj, err := metaObj.GetObject("host")
	if err != nil {
		return host, err
	}
	if hostObj.Has("listeners") {
		host.Listeners, err = toStringMapFromObject(hostObj, "listeners")
		if err != nil {
			return host, err
		}
	}
	if hostObj.Has("properties") {
		host.Properties, err = toStringMapFromObject(hostObj, "properties")
		if err != nil {
			return host, err
		}
	}
	if hostObj.Has("attributes") {
		attrObj, err := hostObj.GetObject("attributes")
		if err != nil {
			return host, err
		}
		for key, node := range attrObj.obj {
			host.Attributes[key] = output.NewWrappedNodeExpr(node, nil, nil, nil)
		}
	}
	if hostObj.Has("styleAttribute") {
		style, err := hostObj.GetString("styleAttribute")
		if err != nil {
			return host, err
		}
		host.SpecialAttributes.StyleAttr = &style
	}
	if hostObj.Has("classAttribute") {
		class, err := hostObj.GetString("classAttribute")
		if err != nil {
			return host, err
		}
		host.SpecialAttributes.ClassAttr = &class
	}
	return host, nil
}

func extractHostDirectives(metaObj *AstObject) ([]render3.R3HostDirectiveMetadata, error) {
	arr, err := metaObj.GetArray("hostDirectives")
	if err != nil {
		return nil, err
	}
	var res []render3.R3HostDirectiveMetadata
	for _, node := range arr {
		obj, err := node.GetObject()
		if err != nil {
			continue
		}
		hd := render3.R3HostDirectiveMetadata{}
		if obj.Has("directive") {
			expr, err := obj.GetOpaque("directive")
			if err == nil {
				hd.Directive = render3.R3Reference{
					Type: expr,
				}
			}
		}
		if obj.Has("isForwardReference") {
			hd.IsForwardReference, _ = obj.GetBoolean("isForwardReference")
		}

		if obj.Has("inputs") {
			inputsArr, _ := obj.GetArray("inputs")
			hd.Inputs = make(map[string]string)
			for i := 0; i < len(inputsArr); i += 2 {
				publicName, _ := inputsArr[i].GetString()
				alias, _ := inputsArr[i+1].GetString()
				hd.Inputs[publicName] = alias
			}
		}
		if obj.Has("outputs") {
			outputsArr, _ := obj.GetArray("outputs")
			hd.Outputs = make(map[string]string)
			for i := 0; i < len(outputsArr); i += 2 {
				publicName, _ := outputsArr[i].GetString()
				alias, _ := outputsArr[i+1].GetString()
				hd.Outputs[publicName] = alias
			}
		}
		res = append(res, hd)
	}
	return res, nil
}

func toInputs(metaObj *AstObject) (map[string]render3.R3InputMetadata, error) {
	inputs := map[string]render3.R3InputMetadata{}
	if !metaObj.Has("inputs") {
		return inputs, nil
	}
	inputObj, err := metaObj.GetObject("inputs")
	if err != nil {
		return nil, err
	}
	for key, node := range inputObj.obj {
		value := &AstValue{expression: node, host: inputObj.host}
		if value.IsString() {
			publicName, err := value.GetString()
			if err != nil {
				return nil, err
			}
			inputs[key] = render3.R3InputMetadata{
				ClassPropertyName:   key,
				BindingPropertyName: publicName,
			}
			continue
		}
		if value.IsArray() {
			parts, err := value.GetArray()
			if err != nil {
				return nil, err
			}
			if len(parts) < 2 {
				return nil, linkerError(node, "Unsupported input array metadata for %q.", key)
			}
			publicName, err := parts[0].GetString()
			if err != nil {
				return nil, err
			}
			className, err := parts[1].GetString()
			if err != nil {
				return nil, err
			}
			inputs[key] = render3.R3InputMetadata{
				ClassPropertyName:   className,
				BindingPropertyName: publicName,
			}
			continue
		}
		if value.IsObject() {
			obj, err := value.GetObject()
			if err != nil {
				return nil, err
			}
			className, err := obj.GetString("classPropertyName")
			if err != nil {
				return nil, err
			}
			publicName, err := obj.GetString("publicName")
			if err != nil {
				return nil, err
			}
			required := false
			if obj.Has("isRequired") {
				required, err = obj.GetBoolean("isRequired")
				if err != nil {
					return nil, err
				}
			}
			isSignal := false
			if obj.Has("isSignal") {
				isSignal, err = obj.GetBoolean("isSignal")
				if err != nil {
					return nil, err
				}
			}
			var transform output.Expression
			if obj.Has("transformFunction") {
				transformValue, err := obj.GetValue("transformFunction")
				if err != nil {
					return nil, err
				}
				if !transformValue.IsNull() {
					transform = transformValue.GetOpaque()
				}
			}
			inputs[key] = render3.R3InputMetadata{
				ClassPropertyName:   className,
				BindingPropertyName: publicName,
				Required:            required,
				IsSignal:            isSignal,
				TransformFunction:   transform,
			}
		}
	}
	return inputs, nil
}
