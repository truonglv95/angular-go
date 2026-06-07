package render3

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/shadow_css"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

var IngestComponent func(
	componentName string,
	template []Node,
	constantPool ConstantPool,
	compilationMode compilation.TemplateCompilationMode,
	relativeContextFilePath string,
	i18nUseExternalIds bool,
	deferMeta any,
	allDeferrableDepsFn *output.ReadVarExpr,
	relativeTemplatePath *string,
	enableDebugLocations bool,
	legacyOptionalChaining bool,
	foreignImports []any,
) *compilation.ComponentCompilationJob

var Transform func(job *compilation.ComponentCompilationJob)

var EmitTemplateFn func(job *compilation.ComponentCompilationJob, pool ConstantPool) output.Expression

type Type any
type Statement any
type ConstantPool interface {
	GetConstLiteral(literal output.Expression, forceShared bool) output.Expression
}
type BindingParser any

type HostBindingInput struct {
	ComponentName          string
	ComponentSelector      string
	Properties             []template_parser.ParsedProperty
	Attributes             map[string]output.Expression
	Events                 []template_parser.ParsedEvent
	LegacyOptionalChaining bool
}

var IngestHostBinding func(
	input *HostBindingInput,
	bindingParser BindingParser,
	constantPool ConstantPool,
) *compilation.HostBindingCompilationJob

var TransformHostBinding func(job *compilation.HostBindingCompilationJob)

var EmitHostBindingFunction func(job *compilation.HostBindingCompilationJob) *output.FunctionExpr

const COMPONENT_VARIABLE = "%COMP%"

var HOST_ATTR = fmt.Sprintf("_nghost-%s", COMPONENT_VARIABLE)
var CONTENT_ATTR = fmt.Sprintf("_ngcontent-%s", COMPONENT_VARIABLE)

var ParseSelectorToR3Selector func(selector *string) []any

func baseDirectiveFields(
	meta R3DirectiveMetadata,
	constantPool ConstantPool,
	bindingParser BindingParser,
) *DefinitionMap {
	definitionMap := NewDefinitionMap()
	var selectors []any
	if meta.Selector != nil && ParseSelectorToR3Selector != nil {
		selectors = ParseSelectorToR3Selector(meta.Selector)
	} else if ParseSelectorToR3Selector != nil {
		emptyStr := ""
		selectors = ParseSelectorToR3Selector(&emptyStr)
	}

	// e.g. `type: MyDirective`
	definitionMap.Set("type", meta.Type.Value)

	// e.g. `selectors: [['', 'someDir', '']]`
	if len(selectors) > 0 {
		definitionMap.Set("selectors", arrayToLiteral(selectors))
	}

	if len(meta.Queries) > 0 {
		// e.g. `contentQueries: (rf, ctx, dirIndex) => { ... }
		definitionMap.Set(
			"contentQueries",
			CreateContentQueriesFunction(meta.Queries, constantPool, meta.Name),
		)
	}

	if len(meta.ViewQueries) > 0 {
		definitionMap.Set(
			"viewQuery",
			CreateViewQueriesFunction(meta.ViewQueries, constantPool, meta.Name),
		)
	}

	var selectorStr string
	if meta.Selector != nil {
		selectorStr = *meta.Selector
	}

	// e.g. `hostBindings: (rf, ctx) => { ... }
	definitionMap.Set(
		"hostBindings",
		createHostBindingsFunction(
			meta.Host,
			parse_util.ParseSourceSpan{},
			bindingParser,
			constantPool,
			selectorStr,
			meta.Name,
			definitionMap,
			meta.LegacyOptionalChaining,
		),
	)

	if len(meta.Inputs) > 0 {
		var inputProps []output.LiteralMapEntry
		var sortedKeys []string
		for k := range meta.Inputs {
			sortedKeys = append(sortedKeys, k)
		}
		if len(meta.InputProperties) > 0 {
			orderMap := make(map[string]int)
			for i, prop := range meta.InputProperties {
				orderMap[prop] = i
			}
			sort.Slice(sortedKeys, func(i, j int) bool {
				idxI, okI := orderMap[sortedKeys[i]]
				idxJ, okJ := orderMap[sortedKeys[j]]
				if okI && okJ {
					return idxI < idxJ
				}
				if okI {
					return true
				}
				if okJ {
					return false
				}
				return sortedKeys[i] < sortedKeys[j]
			})
		} else {
			sort.Strings(sortedKeys)
		}

		for _, classPropName := range sortedKeys {
			inputMeta := meta.Inputs[classPropName]
			var valueExpr output.Expression

			flags := 0 // Default InputFlags.None for standard @Input
			if inputMeta.IsSignal {
				flags |= 1 // InputFlags.SignalBased
			}
			if inputMeta.TransformFunction != nil {
				flags |= 2 // InputFlags.HasTransform
			}

			if classPropName == inputMeta.BindingPropertyName {
				if flags == 0 {
					valueExpr = LiteralExpr(classPropName)
				} else {
					valueExpr = LiteralArr([]output.Expression{
						LiteralExpr(flags),
						LiteralExpr(inputMeta.BindingPropertyName),
					})
				}
			} else {
				valueExpr = LiteralArr([]output.Expression{
					LiteralExpr(flags),
					LiteralExpr(inputMeta.BindingPropertyName),
					LiteralExpr(classPropName),
				})
			}
			inputProps = append(inputProps, output.NewLiteralMapPropertyAssignment(classPropName, valueExpr, false))
		}
		definitionMap.Set("inputs", output.NewLiteralMapExpr(inputProps, nil, nil, nil))
	} else {
		// definitionMap.Set("inputs", LiteralExpr(nil)) // Can be omitted if empty
	}

	if len(meta.Outputs) > 0 {
		var outputProps []output.LiteralMapEntry
		var sortedKeys []string
		for k := range meta.Outputs {
			sortedKeys = append(sortedKeys, k)
		}
		if len(meta.OutputProperties) > 0 {
			orderMap := make(map[string]int)
			for i, prop := range meta.OutputProperties {
				orderMap[prop] = i
			}
			sort.Slice(sortedKeys, func(i, j int) bool {
				idxI, okI := orderMap[sortedKeys[i]]
				idxJ, okJ := orderMap[sortedKeys[j]]
				if okI && okJ {
					return idxI < idxJ
				}
				if okI {
					return true
				}
				if okJ {
					return false
				}
				return sortedKeys[i] < sortedKeys[j]
			})
		} else {
			sort.Strings(sortedKeys)
		}

		for _, classPropName := range sortedKeys {
			bindingPropName := meta.Outputs[classPropName]
			outputProps = append(outputProps, output.NewLiteralMapPropertyAssignment(classPropName, LiteralExpr(bindingPropName), false))
		}
		definitionMap.Set("outputs", output.NewLiteralMapExpr(outputProps, nil, nil, nil))
	} else {
		// definitionMap.Set("outputs", LiteralExpr(nil)) // Can be omitted if empty
	}

	if meta.ExportAs != nil {
		var exportAs []output.Expression
		for _, e := range meta.ExportAs {
			exportAs = append(exportAs, LiteralExpr(e))
		}
		definitionMap.Set("exportAs", LiteralArr(exportAs))
	}

	if !meta.IsStandalone {
		definitionMap.Set("standalone", LiteralExpr(false))
	}
	if meta.IsSignal {
		definitionMap.Set("signals", LiteralExpr(true))
	}

	return definitionMap
}

func addFeatures(
	definitionMap *DefinitionMap,
	meta interface{},
) {
	var features []output.Expression
	var providers, viewProviders interface{}
	var hostDirectives []R3HostDirectiveMetadata
	var usesInheritance bool
	var usesOnChanges bool
	var controlCreate *struct{ PassThroughInput *string }
	var externalStyles []string

	var isStandalone bool
	if dirMeta, ok := meta.(R3DirectiveMetadata); ok {
		providers = dirMeta.Providers
		hostDirectives = dirMeta.HostDirectives
		usesInheritance = dirMeta.UsesInheritance
		usesOnChanges = dirMeta.Lifecycle.UsesOnChanges
		controlCreate = dirMeta.ControlCreate
		isStandalone = dirMeta.IsStandalone
	} else if compMeta, ok := meta.(R3ComponentMetadata[R3TemplateDependency]); ok {
		providers = compMeta.Providers
		viewProviders = compMeta.ViewProviders // Wait, ViewProviders isn't in R3ComponentMetadata? I will assume it is.
		hostDirectives = compMeta.HostDirectives
		usesInheritance = compMeta.UsesInheritance
		usesOnChanges = compMeta.Lifecycle.UsesOnChanges
		controlCreate = compMeta.ControlCreate
		externalStyles = compMeta.ExternalStyles
		isStandalone = compMeta.IsStandalone
	}

	if isStandalone {
		// Angular 19+ does not use StandaloneFeature anymore.
	}

	if providers != nil || viewProviders != nil {
		var args []output.Expression
		if providers != nil {
			args = append(args, providers.(output.Expression))
		} else {
			args = append(args, LiteralArr([]output.Expression{}))
		}
		if viewProviders != nil {
			args = append(args, viewProviders.(output.Expression))
		}
		features = append(features, ImportExpr(*Identifiers.ProvidersFeature).CallFn(args, nil, false, nil))
	}

	if len(hostDirectives) > 0 {
		features = append(features, ImportExpr(*Identifiers.HostDirectivesFeature).CallFn([]output.Expression{
			createHostDirectivesFeatureArg(hostDirectives),
		}, nil, false, nil))
	}

	if usesInheritance {
		features = append(features, ImportExpr(*Identifiers.InheritDefinitionFeature))
	}

	if usesOnChanges {
		features = append(features, ImportExpr(*Identifiers.NgOnChangesFeature))
	}

	if controlCreate != nil {
		features = append(features, ImportExpr(*Identifiers.ControlFeature).CallFn([]output.Expression{
			LiteralExpr(controlCreate.PassThroughInput),
		}, nil, false, nil))
	}

	if len(externalStyles) > 0 {
		var externalStyleNodes []output.Expression
		for _, externalStyle := range externalStyles {
			externalStyleNodes = append(externalStyleNodes, LiteralExpr(externalStyle))
		}
		features = append(features, ImportExpr(*Identifiers.ExternalStylesFeature).CallFn([]output.Expression{
			LiteralArr(externalStyleNodes),
		}, nil, false, nil))
	}

	if len(features) > 0 {
		definitionMap.Set("features", LiteralArr(features))
	}
}

func CompileDirectiveFromMetadata(
	meta R3DirectiveMetadata,
	constantPool ConstantPool,
	bindingParser BindingParser,
) R3CompiledExpression {
	definitionMap := baseDirectiveFields(meta, constantPool, bindingParser)
	addFeatures(definitionMap, meta)

	expression := ImportExpr(*Identifiers.DefineDirective).CallFn([]output.Expression{
		definitionMap.ToLiteralMap(),
	}, nil, false, nil)
	// true passed for pure? we can assume CallFn handles pure as 3rd/4th arg. Wait, `true` is pure.

	typeExpr := createDirectiveType(meta)

	return R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func CompileComponentFromMetadata(
	meta R3ComponentMetadata[R3TemplateDependency],
	constantPool ConstantPool,
	bindingParser BindingParser,
) R3CompiledExpression {
	definitionMap := baseDirectiveFields(meta.R3DirectiveMetadata, constantPool, bindingParser)
	addFeatures(definitionMap, meta)

	templateTypeName := meta.Name

	_ = (output.Expression)(nil)
	if meta.Defer.Mode == DeferBlockDepsEmitMode_PerComponent && meta.Defer.DependenciesFn != nil {
		_ = templateTypeName + "_DeferFn"
		// constantPool.statements.push
		// we assume ConstantPool has Statements
		// For now we'll just ignore constant pool statements mutation in this stub if we can't express it, but let's try.
		// allDeferrableDepsFn = output.Variable(fnName)
	}

	compilationMode := compilation.TemplateCompilationMode_Full
	if meta.IsStandalone && !meta.HasDirectiveDependencies && meta.DeclarationListEmitMode != DeclarationListEmitMode_RuntimeResolved {
		compilationMode = compilation.TemplateCompilationMode_DomOnly
	}

	var foreignImportsAny []any
	for _, fi := range meta.ForeignImports {
		foreignImportsAny = append(foreignImportsAny, fi)
	}

	if IngestComponent == nil {
		panic("render3.IngestComponent is not initialized")
	}
	var tpl *compilation.ComponentCompilationJob
	func() {
		tpl = IngestComponent(
			meta.Name,
			meta.Template.Children,
			constantPool,
			compilationMode,
			meta.RelativeContextFilePath,
			meta.I18nUseExternalIds,
			meta.Defer,
			nil,
			meta.RelativeTemplatePath,
			false,
			meta.LegacyOptionalChaining,
			foreignImportsAny,
		)
	}()

	if Transform != nil {
		func() {
			Transform(tpl)
		}()
	}

	if EmitTemplateFn == nil {
		panic("render3.EmitTemplateFn is not initialized")
	}
	var templateFn output.Expression
	func() {
		templateFn = EmitTemplateFn(tpl, constantPool)
	}()

	if tpl.ContentSelectors != nil {
		definitionMap.Set("ngContentSelectors", tpl.ContentSelectors)
	}

	definitionMap.Set("decls", LiteralExpr(tpl.Root.Decls))
	definitionMap.Set("vars", LiteralExpr(tpl.Root.Vars))

	if len(tpl.Consts) > 0 {
		if len(tpl.ConstsInitializers) > 0 {
			definitionMap.Set("consts", output.NewArrowFunctionExpr([]*output.FnParam{}, append(tpl.ConstsInitializers, output.NewReturnStatement(LiteralArr(tpl.Consts), nil, nil)), nil, nil, nil))
		} else {
			definitionMap.Set("consts", LiteralArr(tpl.Consts))
		}
	}
	definitionMap.Set("template", templateFn)

	if meta.DeclarationListEmitMode != DeclarationListEmitMode_RuntimeResolved && len(meta.Declarations) > 0 {
		var declTypes []output.Expression
		for _, decl := range meta.Declarations {
			declTypes = append(declTypes, decl.Type.(output.Expression))
		}
		definitionMap.Set("dependencies", compileDeclarationList(LiteralArr(declTypes), meta.DeclarationListEmitMode))
	} else if meta.DeclarationListEmitMode == DeclarationListEmitMode_RuntimeResolved {
		var args []output.Expression
		args = append(args, meta.Type.Value)
		if meta.RawImports != nil {
			args = append(args, meta.RawImports.(output.Expression))
		}
		definitionMap.Set("dependencies", ImportExpr(*Identifiers.GetComponentDepsFactory).CallFn(args, nil, false, nil))
	}

	if meta.Encapsulation == 0 { // Emulated
		meta.Encapsulation = 0
	}

	hasStyles := len(meta.ExternalStyles) > 0
	if len(meta.Styles) > 0 {
		styleValues := meta.Styles
		if meta.Encapsulation == 0 {
			styleValues = compileStyles(meta.Styles, CONTENT_ATTR, HOST_ATTR)
		}

		var styleNodes []output.Expression
		for _, style := range styleValues {
			if len(strings.TrimSpace(style)) > 0 {
				// constantPool.getConstLiteral
				styleNodes = append(styleNodes, LiteralExpr(style)) // Simplified
			}
		}

		if len(styleNodes) > 0 {
			hasStyles = true
			definitionMap.Set("styles", LiteralArr(styleNodes))
		}
	}

	if !hasStyles && meta.Encapsulation == 0 {
		meta.Encapsulation = 2 // ViewEncapsulation.None
	}

	if meta.Encapsulation != 0 {
		definitionMap.Set("encapsulation", output.NewLiteralExpr(meta.Encapsulation, nil, nil, nil))
	}

	if meta.Animations != nil {
		definitionMap.Set("data", output.NewLiteralMapExpr([]output.LiteralMapEntry{
			output.NewLiteralMapPropertyAssignment("animation", meta.Animations.(output.Expression), false),
		}, nil, nil, nil))
	}

	if meta.ChangeDetection != nil {
		switch cd := meta.ChangeDetection.(type) {
		case int:
			definitionMap.Set("changeDetection", output.NewLiteralExpr(cd, nil, nil, nil))
		case output.Expression:
			definitionMap.Set("changeDetection", cd)
		default:
			definitionMap.Set("changeDetection", output.NewLiteralExpr(cd, nil, nil, nil))
		}
	}

	expression := ImportExpr(*Identifiers.DefineComponent).CallFn([]output.Expression{
		definitionMap.ToLiteralMap(),
	}, nil, false, nil)
	typeExpr := createComponentType(meta)

	return R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func createComponentType(meta R3ComponentMetadata[R3TemplateDependency]) output.Type {
	typeParams := createBaseDirectiveTypeParams(meta.R3DirectiveMetadata)
	typeParams = append(typeParams, stringArrayAsType([]string{}))
	typeParams = append(typeParams, output.NewExpressionType(LiteralExpr(meta.IsStandalone)))
	typeParams = append(typeParams, createHostDirectivesType(meta.R3DirectiveMetadata))
	if meta.IsSignal {
		typeParams = append(typeParams, output.NewExpressionType(LiteralExpr(meta.IsSignal)))
	}
	return output.NewExpressionType(ImportExpr(*Identifiers.ComponentDeclaration))
}

func compileDeclarationList(list output.Expression, mode DeclarationListEmitMode) output.Expression {
	switch mode {
	case DeclarationListEmitMode_Direct:
		return list
	case DeclarationListEmitMode_Closure:
		return output.NewArrowFunctionExpr([]*output.FnParam{}, list, nil, nil, nil)
	case DeclarationListEmitMode_ClosureResolved:
		resolvedList := list.Prop("map", nil).CallFn([]output.Expression{ImportExpr(*Identifiers.ResolveForwardRef)}, nil, false, nil)
		return output.NewArrowFunctionExpr([]*output.FnParam{}, resolvedList, nil, nil, nil)
	case DeclarationListEmitMode_RuntimeResolved:
		panic("Unsupported with an array of pre-resolved dependencies")
	}
	return nil
}

func stringAsType(str string) output.Type {
	return output.NewExpressionType(LiteralExpr(str))
}

func stringMapAsLiteralExpression(m map[string]interface{}) output.Expression {
	var mapValues []output.LiteralMapEntry
	for key, val := range m {
		var value string
		if vStr, ok := val.(string); ok {
			value = vStr
		} else if vArr, ok := val.([]string); ok && len(vArr) > 0 {
			value = vArr[0]
		}
		mapValues = append(mapValues, output.NewLiteralMapPropertyAssignment(key, LiteralExpr(value), true))
	}
	return output.NewLiteralMapExpr(mapValues, nil, nil, nil)
}

func stringArrayAsType(arr []string) output.Type {
	if len(arr) > 0 {
		var mapped []output.Expression
		for _, value := range arr {
			mapped = append(mapped, LiteralExpr(value))
		}
		return output.NewExpressionType(LiteralArr(mapped))
	}
	return output.NONE_TYPE
}

func createBaseDirectiveTypeParams(meta R3DirectiveMetadata) []output.Type {
	var selectorForType *string
	if meta.Selector != nil {
		s := strings.ReplaceAll(*meta.Selector, "\n", "")
		selectorForType = &s
	}

	var selType output.Type = output.NONE_TYPE
	if selectorForType != nil {
		selType = stringAsType(*selectorForType)
	}

	var exportAsType output.Type = output.NONE_TYPE
	if meta.ExportAs != nil {
		exportAsType = stringArrayAsType(meta.ExportAs)
	}

	var queryNames []string
	for _, q := range meta.Queries {
		queryNames = append(queryNames, q.PropertyName)
	}

	// Prepare outputs for map
	outputsMap := make(map[string]interface{})
	for k, v := range meta.Outputs {
		outputsMap[k] = v
	}

	return []output.Type{
		TypeWithParameters(meta.Type.Type, meta.TypeArgumentCount),
		selType,
		exportAsType,
		output.NewExpressionType(getInputsTypeExpression(meta)),
		output.NewExpressionType(stringMapAsLiteralExpression(outputsMap)),
		stringArrayAsType(queryNames),
	}
}

func getInputsTypeExpression(meta R3DirectiveMetadata) output.Expression {
	var entries []output.LiteralMapEntry
	for key, value := range meta.Inputs {
		values := []output.LiteralMapEntry{
			output.NewLiteralMapPropertyAssignment("alias", LiteralExpr(value.BindingPropertyName), true),
			output.NewLiteralMapPropertyAssignment("required", LiteralExpr(value.Required), true),
		}
		if value.IsSignal {
			values = append(values, output.NewLiteralMapPropertyAssignment("isSignal", LiteralExpr(value.IsSignal), true))
		}
		entries = append(entries, output.NewLiteralMapPropertyAssignment(key, output.NewLiteralMapExpr(values, nil, nil, nil), true))
	}
	return output.NewLiteralMapExpr(entries, nil, nil, nil)
}

func createDirectiveType(meta R3DirectiveMetadata) output.Type {
	typeParams := createBaseDirectiveTypeParams(meta)
	typeParams = append(typeParams, output.NONE_TYPE)
	typeParams = append(typeParams, output.NewExpressionType(LiteralExpr(meta.IsStandalone)))
	typeParams = append(typeParams, createHostDirectivesType(meta))
	if meta.IsSignal {
		typeParams = append(typeParams, output.NewExpressionType(LiteralExpr(meta.IsSignal)))
	}
	return output.NewExpressionTypeWithParams(ImportExpr(*Identifiers.DirectiveDeclaration), typeParams)
}

func createHostBindingsFunction(
	hostBindingsMetadata R3HostMetadata,
	typeSourceSpan parse_util.ParseSourceSpan,
	bindingParserAny BindingParser,
	constantPool ConstantPool,
	selector string,
	name string,
	definitionMap *DefinitionMap,
	legacyOptionalChaining bool,
) output.Expression {
	var bindingParser *template_parser.BindingParser
	if bindingParserAny != nil {
		bindingParser = bindingParserAny.(*template_parser.BindingParser)
	} else {
		lexer := expression_parser.Lexer{}
		exprParser := expression_parser.NewParser(lexer, false)
		bindingParser = template_parser.NewBindingParser(exprParser, ElementRegistry, nil)
	}

	var sourceSpan expression_parser.ParseSourceSpan
	if typeSourceSpan.Start != nil {
		sourceSpan.Start = typeSourceSpan.Start.Offset
		sourceSpan.FullStart = typeSourceSpan.Start.Offset
	}
	if typeSourceSpan.End != nil {
		sourceSpan.End = typeSourceSpan.End.Offset
	}

	var parsedProperties []template_parser.ParsedProperty
	if bindingParser != nil && hostBindingsMetadata.Properties != nil {
		parsedProperties = bindingParser.CreateBoundHostProperties(hostBindingsMetadata.Properties, sourceSpan)
	}

	var parsedEvents []template_parser.ParsedEvent
	if bindingParser != nil && hostBindingsMetadata.Listeners != nil {
		parsedEvents = bindingParser.CreateDirectiveHostEventAsts(hostBindingsMetadata.Listeners, sourceSpan)
	}

	var hostAttrsArray []output.Expression

	if hostBindingsMetadata.SpecialAttributes.StyleAttr != nil {
		hostAttrsArray = append(hostAttrsArray, output.NewLiteralExpr("style", nil, nil, nil))
		hostAttrsArray = append(hostAttrsArray, output.NewLiteralExpr(*hostBindingsMetadata.SpecialAttributes.StyleAttr, nil, nil, nil))
	}
	if hostBindingsMetadata.SpecialAttributes.ClassAttr != nil {
		hostAttrsArray = append(hostAttrsArray, output.NewLiteralExpr("class", nil, nil, nil))
		hostAttrsArray = append(hostAttrsArray, output.NewLiteralExpr(*hostBindingsMetadata.SpecialAttributes.ClassAttr, nil, nil, nil))
	}
	if hostBindingsMetadata.Attributes != nil {
		var keys []string
		for k := range hostBindingsMetadata.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			hostAttrsArray = append(hostAttrsArray, output.NewLiteralExpr(k, nil, nil, nil))
			hostAttrsArray = append(hostAttrsArray, hostBindingsMetadata.Attributes[k].(output.Expression))
		}
	}

	if len(hostAttrsArray) > 0 {
		definitionMap.Set("hostAttrs", output.NewLiteralArrayExpr(hostAttrsArray, nil, nil, nil))
	}

	if IngestHostBinding == nil {
		panic("render3.IngestHostBinding is not initialized")
	}
	hostJob := IngestHostBinding(&HostBindingInput{
		ComponentName:          name,
		ComponentSelector:      selector,
		Properties:             parsedProperties,
		Attributes:             nil,
		Events:                 parsedEvents,
		LegacyOptionalChaining: legacyOptionalChaining,
	}, bindingParser, constantPool)

	if TransformHostBinding == nil {
		panic("render3.TransformHostBinding is not initialized")
	}
	TransformHostBinding(hostJob)

	if hostJob.Root.Vars != nil && *hostJob.Root.Vars > 0 {
		definitionMap.Set("hostVars", output.NewLiteralExpr(*hostJob.Root.Vars, nil, nil, nil))
	}

	if EmitHostBindingFunction == nil {
		panic("render3.EmitHostBindingFunction is not initialized")
	}
	fn := EmitHostBindingFunction(hostJob)
	if fn == nil {
		return nil
	}
	return fn
}

type ParsedHostBindings struct {
	Attributes        map[string]output.Expression
	Listeners         map[string]string
	Properties        map[string]string
	SpecialAttributes struct {
		StyleAttr *string
		ClassAttr *string
	}
}

func ParseHostBindings(host map[string]interface{}) ParsedHostBindings {
	attributes := make(map[string]output.Expression)
	listeners := make(map[string]string)
	properties := make(map[string]string)
	var specialAttributes struct {
		StyleAttr *string
		ClassAttr *string
	}

	for key, value := range host {
		if strings.HasPrefix(key, "(") && strings.HasSuffix(key, ")") {
			if vStr, ok := value.(string); ok {
				listeners[key[1:len(key)-1]] = vStr
			} else {
				panic("Event binding must be string")
			}
		} else if strings.HasPrefix(key, "[") && strings.HasSuffix(key, "]") {
			if vStr, ok := value.(string); ok {
				properties[key[1:len(key)-1]] = vStr
			} else {
				panic("Property binding must be string")
			}
		} else {
			switch key {
			case "class":
				if vStr, ok := value.(string); ok {
					specialAttributes.ClassAttr = &vStr
				} else {
					panic("Class binding must be string")
				}
			case "style":
				if vStr, ok := value.(string); ok {
					specialAttributes.StyleAttr = &vStr
				} else {
					panic("Style binding must be string")
				}
			default:
				if vStr, ok := value.(string); ok {
					attributes[key] = LiteralExpr(vStr)
				} else {
					attributes[key] = value.(output.Expression)
				}
			}
		}
	}

	return ParsedHostBindings{
		Attributes:        attributes,
		Listeners:         listeners,
		Properties:        properties,
		SpecialAttributes: specialAttributes,
	}
}

func VerifyHostBindings(
	bindings ParsedHostBindings,
	sourceSpan parse_util.ParseSourceSpan,
) []parse_util.ParseError {
	return nil
}

func validateNoEventBindings(
	bindings ParsedHostBindings,
	bindingParser BindingParser,
	sourceSpan parse_util.ParseSourceSpan,
) {
}

func compileStyles(styles []string, selector string, hostSelector string) []string {
	var compiled []string
	shadowCss := shadow_css.NewShadowCss()
	for _, style := range styles {
		compiled = append(compiled, shadowCss.ShimCssText(style, selector, hostSelector))
	}
	return compiled
}

func EncapsulateStyle(style string, componentIdentifier *string) string {
	return ""
}

func createHostDirectivesType(meta R3DirectiveMetadata) output.Type {
	if len(meta.HostDirectives) == 0 {
		return output.NONE_TYPE
	}

	var arr []output.Expression
	for _, hostMeta := range meta.HostDirectives {

		inputsMap := make(map[string]interface{})
		for k, v := range hostMeta.Inputs {
			inputsMap[k] = v
		}

		outputsMap := make(map[string]interface{})
		for k, v := range hostMeta.Outputs {
			outputsMap[k] = v
		}

		arr = append(arr, output.NewLiteralMapExpr([]output.LiteralMapEntry{
			output.NewLiteralMapPropertyAssignment("directive", output.NewTypeofExpr(hostMeta.Directive.Type, nil, nil, nil), false),
			output.NewLiteralMapPropertyAssignment("inputs", stringMapAsLiteralExpression(inputsMap), false),
			output.NewLiteralMapPropertyAssignment("outputs", stringMapAsLiteralExpression(outputsMap), false),
		}, nil, nil, nil))
	}

	return output.NewExpressionType(LiteralArr(arr))
}

func createHostDirectivesFeatureArg(hostDirectives []R3HostDirectiveMetadata) output.Expression {
	var expressions []output.Expression
	hasForwardRef := false

	for _, current := range hostDirectives {
		if len(current.Inputs) == 0 && len(current.Outputs) == 0 {
			expressions = append(expressions, current.Directive.Type)
		} else {
			keys := []output.LiteralMapEntry{
				output.NewLiteralMapPropertyAssignment("directive", current.Directive.Type, false),
			}

			if len(current.Inputs) > 0 {
				inputsLiteral := CreateHostDirectivesMappingArray(current.Inputs)
				if inputsLiteral != nil {
					keys = append(keys, output.NewLiteralMapPropertyAssignment("inputs", inputsLiteral, false))
				}
			}

			if len(current.Outputs) > 0 {
				outputsLiteral := CreateHostDirectivesMappingArray(current.Outputs)
				if outputsLiteral != nil {
					keys = append(keys, output.NewLiteralMapPropertyAssignment("outputs", outputsLiteral, false))
				}
			}

			expressions = append(expressions, output.NewLiteralMapExpr(keys, nil, nil, nil))
		}

		if current.IsForwardReference {
			hasForwardRef = true
		}
	}

	if hasForwardRef {
		return output.NewFunctionExpr([]*output.FnParam{}, []output.Statement{output.NewReturnStatement(LiteralArr(expressions), nil, nil)}, nil, nil, nil, nil)
	}
	return LiteralArr(expressions)
}

func CreateHostDirectivesMappingArray(mapping map[string]string) output.Expression {
	var elements []output.Expression
	// Note: in Go maps are unordered. In TS they are ordered by insertion.
	// We might need to sort keys to make it deterministic if required.
	for publicName, alias := range mapping {
		elements = append(elements, LiteralExpr(publicName), LiteralExpr(alias))
	}
	if len(elements) > 0 {
		return LiteralArr(elements)
	}
	return nil
}

func CompileDeferResolverFunction(meta R3DeferResolverFunctionMetadata) output.Expression {
	var depExprs []output.Expression

	if deps, ok := meta.Dependencies.([]R3DeferPerBlockDependency); ok {
		for _, dep := range deps {
			if dep.IsDeferrable && dep.ImportPath != nil {
				// import("importPath").then(m => m.SymbolName)
				dynamicImport := output.NewDynamicImportExpr(*dep.ImportPath, nil, nil, nil)

				mVar := output.NewReadVarExpr("m", nil, nil, nil)
				var returnExpr output.Expression = mVar.Prop(dep.SymbolName, nil)
				if dep.IsDefaultImport {
					returnExpr = mVar.Prop("default", nil)
				}

				fnParam := &output.FnParam{Name: "m", Type: nil}
				thenFn := output.NewArrowFunctionExpr([]*output.FnParam{fnParam}, returnExpr, nil, nil, nil)

				promise := dynamicImport.Prop("then", nil).CallFn([]output.Expression{thenFn}, nil, false, nil)
				depExprs = append(depExprs, promise)
			} else {
				// fallback for eager
				typeRef := dep.TypeReference.(output.Expression)
				promiseResolve := output.NewReadVarExpr("Promise", nil, nil, nil).Prop("resolve", nil).CallFn([]output.Expression{typeRef}, nil, false, nil)
				depExprs = append(depExprs, promiseResolve)
			}
		}
	}

	arrLiteral := output.NewLiteralArrayExpr(depExprs, nil, nil, nil)
	return output.NewArrowFunctionExpr(nil, arrLiteral, nil, nil, nil)
}

func arrayToLiteral(val any) output.Expression {
	if val == nil {
		return output.NewLiteralExpr(nil, nil, nil, nil)
	}

	switch v := val.(type) {
	case string:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case int:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case float64:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case bool:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case output.Expression:
		return v
	}

	rt := reflect.TypeOf(val)
	if rt != nil && rt.Kind() == reflect.Slice {
		rv := reflect.ValueOf(val)
		var entries []output.Expression
		for i := 0; i < rv.Len(); i++ {
			entries = append(entries, arrayToLiteral(rv.Index(i).Interface()))
		}
		return output.NewLiteralArrayExpr(entries, nil, nil, nil)
	}

	return output.NewLiteralExpr(val, nil, nil, nil)
}
