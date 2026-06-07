package template_parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

const PROPERTY_PARTS_SEPARATOR = "."
const ATTRIBUTE_PREFIX = "attr"
const ANIMATE_PREFIX = "animate"
const CLASS_PREFIX = "class"
const STYLE_PREFIX = "style"
const TEMPLATE_ATTR_PREFIX = "*"
const LEGACY_ANIMATE_PROP_PREFIX = "animate-"

type BindingType int

const (
	BindingTypeProperty BindingType = iota
	BindingTypeAttribute
	BindingTypeClass
	BindingTypeStyle
	BindingTypeAnimation
	BindingTypeTwoWay
	BindingTypeLegacyAnimation
)

type ParsedPropertyType int

const (
	ParsedPropertyTypeDefault ParsedPropertyType = iota
	ParsedPropertyTypeLiteralAttr
	ParsedPropertyTypeAnimation
	ParsedPropertyTypeTwoWay
	ParsedPropertyTypeLegacyAnimation
)

type ParsedEventType int

const (
	ParsedEventTypeRegular ParsedEventType = iota
	ParsedEventTypeAnimation
	ParsedEventTypeTwoWay
	ParsedEventTypeLegacyAnimation
)

type ParseErrorLevel int

const (
	ParseErrorLevelWarning ParseErrorLevel = iota
	ParseErrorLevelError
)

type ParseError struct {
	Span    expression_parser.ParseSourceSpan
	Message string
	Level   ParseErrorLevel
}

func NewParseError(span expression_parser.ParseSourceSpan, message string, level ParseErrorLevel) ParseError {
	return ParseError{Span: span, Message: message, Level: level}
}

type BoundElementProperty struct {
	Name            string
	Type            BindingType
	SecurityContext core.SecurityContext
	Expression      expression_parser.ASTWithSource
	Unit            *string
	SourceSpan      expression_parser.ParseSourceSpan
	KeySpan         expression_parser.ParseSourceSpan
	ValueSpan       *expression_parser.ParseSourceSpan
}

func NewBoundElementProperty(
	name string,
	bindingType BindingType,
	securityContext core.SecurityContext,
	expression expression_parser.ASTWithSource,
	unit *string,
	sourceSpan expression_parser.ParseSourceSpan,
	keySpan expression_parser.ParseSourceSpan,
	valueSpan *expression_parser.ParseSourceSpan,
) BoundElementProperty {
	return BoundElementProperty{
		Name:            name,
		Type:            bindingType,
		SecurityContext: securityContext,
		Expression:      expression,
		Unit:            unit,
		SourceSpan:      sourceSpan,
		KeySpan:         keySpan,
		ValueSpan:       valueSpan,
	}
}

type ParsedProperty struct {
	Name       string
	Expression expression_parser.ASTWithSource
	Type       ParsedPropertyType
	SourceSpan expression_parser.ParseSourceSpan
	KeySpan    expression_parser.ParseSourceSpan
	ValueSpan  *expression_parser.ParseSourceSpan
}

func NewParsedProperty(
	name string,
	expression expression_parser.ASTWithSource,
	propType ParsedPropertyType,
	sourceSpan expression_parser.ParseSourceSpan,
	keySpan expression_parser.ParseSourceSpan,
	valueSpan *expression_parser.ParseSourceSpan,
) ParsedProperty {
	return ParsedProperty{
		Name:       name,
		Expression: expression,
		Type:       propType,
		SourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
	}
}

func (p *ParsedProperty) IsLegacyAnimation() bool {
	return p.Type == ParsedPropertyTypeLegacyAnimation
}

func (p *ParsedProperty) IsLiteral() bool {
	return p.Type == ParsedPropertyTypeLiteralAttr
}

func (p *ParsedProperty) IsAnimation() bool {
	return p.Type == ParsedPropertyTypeAnimation
}

type ParsedEvent struct {
	Name          string
	TargetOrPhase *string
	Type          ParsedEventType
	Handler       expression_parser.ASTWithSource
	SourceSpan    expression_parser.ParseSourceSpan
	HandlerSpan   expression_parser.ParseSourceSpan
	KeySpan       expression_parser.ParseSourceSpan
}

func NewParsedEvent(
	name string,
	targetOrPhase *string,
	eventType ParsedEventType,
	handler expression_parser.ASTWithSource,
	sourceSpan expression_parser.ParseSourceSpan,
	handlerSpan expression_parser.ParseSourceSpan,
	keySpan expression_parser.ParseSourceSpan,
) ParsedEvent {
	return ParsedEvent{
		Name:          name,
		TargetOrPhase: targetOrPhase,
		Type:          eventType,
		Handler:       handler,
		SourceSpan:    sourceSpan,
		HandlerSpan:   handlerSpan,
		KeySpan:       keySpan,
	}
}

type ParsedVariable struct {
	Name       string
	Value      string
	SourceSpan expression_parser.ParseSourceSpan
	KeySpan    expression_parser.ParseSourceSpan
	ValueSpan  *expression_parser.ParseSourceSpan
}

func NewParsedVariable(
	name string,
	value string,
	sourceSpan expression_parser.ParseSourceSpan,
	keySpan expression_parser.ParseSourceSpan,
	valueSpan *expression_parser.ParseSourceSpan,
) ParsedVariable {
	return ParsedVariable{
		Name:       name,
		Value:      value,
		SourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
	}
}

type HostProperties map[string]string
type HostListeners map[string]string

type ElementSchemaRegistry interface {
	ValidateProperty(name string) struct {
		Error bool
		Msg   *string
	}
	ValidateAttribute(name string) struct {
		Error bool
		Msg   *string
	}
	GetMappedPropName(name string) string
	SecurityContext(name string, propName string, isAttribute bool) core.SecurityContext
	AllKnownElementNames() []string
	HasElement(name string, schemaCtx []string) bool
}

type CssSelector struct {
	Element      *string
	NotSelectors []*CssSelector
}

func (c *CssSelector) IsElementSelector() bool {
	return c.Element != nil
}

func ParseCssSelector(selector string) []*CssSelector {
	return []*CssSelector{}
}

func splitNsName(name string, lowercase bool) (*string, string) {
	// Dummy
	return nil, name
}

func mergeNsAndName(ns string, name string) string {
	return ns + ":" + name
}

func splitAtColon(rawName string, defaultValues []*string) (*string, *string) {
	parts := strings.SplitN(rawName, ":", 2)
	if len(parts) == 2 {
		return &parts[0], &parts[1]
	}
	return defaultValues[0], defaultValues[1]
}

func splitAtPeriod(rawName string, defaultValues []*string) (*string, *string) {
	parts := strings.SplitN(rawName, ".", 2)
	if len(parts) == 2 {
		return &parts[0], &parts[1]
	}
	return defaultValues[0], defaultValues[1]
}

type BindingParser struct {
	exprParser     *expression_parser.Parser
	schemaRegistry ElementSchemaRegistry
	Errors         []ParseError
}

func NewBindingParser(exprParser *expression_parser.Parser, schemaRegistry ElementSchemaRegistry, errors []ParseError) *BindingParser {
	return &BindingParser{
		exprParser:     exprParser,
		schemaRegistry: schemaRegistry,
		Errors:         errors,
	}
}

func (b *BindingParser) CreateBoundHostProperties(properties HostProperties, sourceSpan expression_parser.ParseSourceSpan) []ParsedProperty {
	boundProps := []ParsedProperty{}
	var sortedKeys []string
	for propName := range properties {
		sortedKeys = append(sortedKeys, propName)
	}
	sort.Strings(sortedKeys)

	for _, propName := range sortedKeys {
		expression := properties[propName]
		b.ParsePropertyBinding(
			propName,
			expression,
			true,
			false,
			sourceSpan,
			sourceSpan.Start,
			nil,
			&[][]string{},
			&boundProps,
			sourceSpan,
		)
	}
	return boundProps
}

func (b *BindingParser) CreateDirectiveHostEventAsts(hostListeners HostListeners, sourceSpan expression_parser.ParseSourceSpan) []ParsedEvent {
	targetEvents := []ParsedEvent{}
	var sortedKeys []string
	for propName := range hostListeners {
		sortedKeys = append(sortedKeys, propName)
	}
	sort.Strings(sortedKeys)

	for _, propName := range sortedKeys {
		expression := hostListeners[propName]
		b.ParseEvent(
			propName,
			expression,
			false,
			sourceSpan,
			sourceSpan,
			&[][]string{},
			&targetEvents,
			sourceSpan,
		)
	}
	return targetEvents
}

func (b *BindingParser) ParseInterpolation(value string, sourceSpan expression_parser.ParseSourceSpan, interpolatedTokens []interface{}) expression_parser.ASTWithSource {
	// Use FullStart offset (pre-trivia) as absoluteOffset, matching TypeScript's
	// `absoluteOffset = sourceSpan.fullStart.offset` in parseInterpolation().
	// FullStart defaults to Start when no trivia was consumed.
	absoluteOffset := sourceSpan.FullStart
	ast := b.exprParser.ParseInterpolation(value, sourceSpan, absoluteOffset, interpolatedTokens)
	if ast.Ast != nil {
		for _, e := range ast.Errors {
			b.Errors = append(b.Errors, ParseError{Span: e.Span, Message: e.Message, Level: ParseErrorLevelError})
		}
		return ast
	}
	return expression_parser.ASTWithSource{}
}

func (b *BindingParser) ParseInterpolationExpression(expression string, sourceSpan expression_parser.ParseSourceSpan) expression_parser.ASTWithSource {
	absoluteOffset := sourceSpan.Start
	ast := b.exprParser.ParseInterpolationExpression(expression, sourceSpan, absoluteOffset)
	if ast.Ast != nil {
		for _, e := range ast.Errors {
			b.Errors = append(b.Errors, ParseError{Span: e.Span, Message: e.Message, Level: ParseErrorLevelError})
		}
		return ast
	}
	return expression_parser.ASTWithSource{}
}

func moveParseSourceSpan(sourceSpan expression_parser.ParseSourceSpan, absoluteSpan expression_parser.AbsoluteSourceSpan) expression_parser.ParseSourceSpan {
	startDiff := absoluteSpan.Start - sourceSpan.Start
	endDiff := absoluteSpan.End - sourceSpan.End
	return expression_parser.ParseSourceSpan{
		Start: sourceSpan.Start + startDiff,
		End:   sourceSpan.End + endDiff,
	}
}

func (b *BindingParser) ParseInlineTemplateBinding(
	tplKey string,
	tplValue string,
	sourceSpan expression_parser.ParseSourceSpan,
	absoluteValueOffset int,
	targetMatchableAttrs *[][]string,
	targetProps *[]ParsedProperty,
	targetVars *[]ParsedVariable,
	isIvyAst bool,
) {
	absoluteKeyOffset := sourceSpan.Start + len(TEMPLATE_ATTR_PREFIX)
	bindings := b.parseTemplateBindings(tplKey, tplValue, sourceSpan, absoluteKeyOffset, absoluteValueOffset)

	for _, b_ := range bindings {
		switch binding := b_.(type) {
		case *expression_parser.VariableBinding:
			bindingSpan := moveParseSourceSpan(sourceSpan, binding.SourceSpan)
			key := binding.Key.Source
			keySpan := moveParseSourceSpan(sourceSpan, expression_parser.AbsoluteSourceSpan{Start: binding.Key.Span.Start, End: binding.Key.Span.End})
			value := ""
			if binding.Value != nil {
				value = binding.Value.Source
			} else {
				value = "$implicit"
			}
			*targetVars = append(*targetVars, NewParsedVariable(key, value, bindingSpan, keySpan, nil))
		case *expression_parser.ExpressionBinding:
			bindingSpan := moveParseSourceSpan(sourceSpan, binding.SourceSpan)
			key := binding.Key.Source
			keySpan := moveParseSourceSpan(sourceSpan, expression_parser.AbsoluteSourceSpan{Start: binding.Key.Span.Start, End: binding.Key.Span.End})
			if binding.Value != nil && binding.Value.Ast != nil {
				srcSpan := sourceSpan
				if isIvyAst {
					srcSpan = bindingSpan
				}
				valueSpan := moveParseSourceSpan(sourceSpan, expression_parser.AbsoluteSourceSpan{Start: binding.Value.AbsoluteOffset, End: binding.Value.AbsoluteOffset})
				b.parsePropertyAst(key, *binding.Value, false, srcSpan, keySpan, &valueSpan, targetMatchableAttrs, targetProps)
			} else {
				*targetMatchableAttrs = append(*targetMatchableAttrs, []string{key, ""})
				b.ParseLiteralAttr(key, nil, keySpan, absoluteValueOffset, nil, targetMatchableAttrs, targetProps, keySpan)
			}
		}
	}
}

func (b *BindingParser) parseTemplateBindings(
	tplKey string,
	tplValue string,
	sourceSpan expression_parser.ParseSourceSpan,
	absoluteKeyOffset int,
	absoluteValueOffset int,
) []expression_parser.TemplateBinding {
	bindingsResult := b.exprParser.ParseTemplateBindings(tplKey, tplValue, sourceSpan, absoluteKeyOffset, absoluteValueOffset)
	for _, e := range bindingsResult.Errors {
		b.Errors = append(b.Errors, ParseError{Span: e.Span, Message: e.Message, Level: ParseErrorLevelError})
	}
	for _, warning := range bindingsResult.Warnings {
		b.reportError(warning, sourceSpan, ParseErrorLevelWarning)
	}
	return bindingsResult.TemplateBindings
}

func isLegacyAnimationLabel(name string) bool {
	return len(name) > 0 && name[0] == '@'
}

func (b *BindingParser) ParseLiteralAttr(
	name string,
	value *string,
	sourceSpan expression_parser.ParseSourceSpan,
	absoluteOffset int,
	valueSpan *expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetProps *[]ParsedProperty,
	keySpan expression_parser.ParseSourceSpan,
) {
	if isLegacyAnimationLabel(name) {
		name = name[1:]
		if true {
			keySpan = moveParseSourceSpan(keySpan, expression_parser.AbsoluteSourceSpan{Start: keySpan.Start + 1, End: keySpan.End})
		}
		if value != nil && *value != "" {
			b.reportError(`Assigning animation triggers via @prop="exp" attributes with an expression is invalid. Use property bindings (e.g. [@prop]="exp") or use an attribute without a value (e.g. @prop) instead.`, sourceSpan, ParseErrorLevelError)
		}
		b.parseLegacyAnimation(name, value, sourceSpan, absoluteOffset, keySpan, valueSpan, targetMatchableAttrs, targetProps)
	} else {
		val := ""
		if value != nil {
			val = *value
		}
		*targetProps = append(*targetProps, NewParsedProperty(
			name,
			b.exprParser.WrapLiteralPrimitive(&val, "", absoluteOffset),
			ParsedPropertyTypeLiteralAttr,
			sourceSpan,
			keySpan,
			valueSpan,
		))
	}
}

func (b *BindingParser) ParsePropertyBinding(
	name string,
	expression string,
	isHost bool,
	isPartOfAssignmentBinding bool,
	sourceSpan expression_parser.ParseSourceSpan,
	absoluteOffset int,
	valueSpan *expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetProps *[]ParsedProperty,
	keySpan expression_parser.ParseSourceSpan,
) {
	if len(name) == 0 {
		b.reportError(`Property name is missing in binding`, sourceSpan, ParseErrorLevelError)
	}

	isLegacyAnimationProp := false
	if strings.HasPrefix(name, LEGACY_ANIMATE_PROP_PREFIX) {
		isLegacyAnimationProp = true
		name = name[len(LEGACY_ANIMATE_PROP_PREFIX):]
		keySpan = moveParseSourceSpan(keySpan, expression_parser.AbsoluteSourceSpan{Start: keySpan.Start + len(LEGACY_ANIMATE_PROP_PREFIX), End: keySpan.End})
	} else if isLegacyAnimationLabel(name) {
		isLegacyAnimationProp = true
		name = name[1:]
		keySpan = moveParseSourceSpan(keySpan, expression_parser.AbsoluteSourceSpan{Start: keySpan.Start + 1, End: keySpan.End})
	}

	spanToUse := sourceSpan
	if valueSpan != nil {
		spanToUse = *valueSpan
	}

	if isLegacyAnimationProp {
		b.parseLegacyAnimation(name, &expression, sourceSpan, absoluteOffset, keySpan, valueSpan, targetMatchableAttrs, targetProps)
	} else if strings.HasPrefix(name, ANIMATE_PREFIX+PROPERTY_PARTS_SEPARATOR) {
		b.parseAnimation(name, b.ParseBinding(expression, isHost, spanToUse, absoluteOffset), sourceSpan, keySpan, valueSpan, targetMatchableAttrs, targetProps)
	} else {
		b.parsePropertyAst(name, b.ParseBinding(expression, isHost, spanToUse, absoluteOffset), isPartOfAssignmentBinding, sourceSpan, keySpan, valueSpan, targetMatchableAttrs, targetProps)
	}
}

func (b *BindingParser) ParsePropertyInterpolation(
	name string,
	value string,
	sourceSpan expression_parser.ParseSourceSpan,
	valueSpan *expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetProps *[]ParsedProperty,
	keySpan expression_parser.ParseSourceSpan,
	interpolatedTokens []interface{},
) bool {
	spanToUse := sourceSpan
	if valueSpan != nil {
		spanToUse = *valueSpan
	}
	expr := b.ParseInterpolation(value, spanToUse, interpolatedTokens)
	if expr.Ast != nil {
		b.parsePropertyAst(name, expr, false, sourceSpan, keySpan, valueSpan, targetMatchableAttrs, targetProps)
		return true
	}
	return false
}

func (b *BindingParser) parsePropertyAst(
	name string,
	ast expression_parser.ASTWithSource,
	isPartOfAssignmentBinding bool,
	sourceSpan expression_parser.ParseSourceSpan,
	keySpan expression_parser.ParseSourceSpan,
	valueSpan *expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetProps *[]ParsedProperty,
) {
	*targetMatchableAttrs = append(*targetMatchableAttrs, []string{name, ast.Source})
	propType := ParsedPropertyTypeDefault
	if isPartOfAssignmentBinding {
		propType = ParsedPropertyTypeTwoWay
	}
	*targetProps = append(*targetProps, NewParsedProperty(name, ast, propType, sourceSpan, keySpan, valueSpan))
}

func (b *BindingParser) parseAnimation(
	name string,
	ast expression_parser.ASTWithSource,
	sourceSpan expression_parser.ParseSourceSpan,
	keySpan expression_parser.ParseSourceSpan,
	valueSpan *expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetProps *[]ParsedProperty,
) {
	*targetMatchableAttrs = append(*targetMatchableAttrs, []string{name, ast.Source})
	*targetProps = append(*targetProps, NewParsedProperty(name, ast, ParsedPropertyTypeAnimation, sourceSpan, keySpan, valueSpan))
}

func (b *BindingParser) parseLegacyAnimation(
	name string,
	expression *string,
	sourceSpan expression_parser.ParseSourceSpan,
	absoluteOffset int,
	keySpan expression_parser.ParseSourceSpan,
	valueSpan *expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetProps *[]ParsedProperty,
) {
	if len(name) == 0 {
		b.reportError("Animation trigger is missing", sourceSpan, ParseErrorLevelError)
	}
	exprToUse := "undefined"
	if expression != nil {
		exprToUse = *expression
	}
	spanToUse := sourceSpan
	if valueSpan != nil {
		spanToUse = *valueSpan
	}
	ast := b.ParseBinding(exprToUse, false, spanToUse, absoluteOffset)
	*targetMatchableAttrs = append(*targetMatchableAttrs, []string{name, ast.Source})
	*targetProps = append(*targetProps, NewParsedProperty(name, ast, ParsedPropertyTypeLegacyAnimation, sourceSpan, keySpan, valueSpan))
}

func (b *BindingParser) ParseBinding(value string, isHostBinding bool, sourceSpan expression_parser.ParseSourceSpan, absoluteOffset int) expression_parser.ASTWithSource {
	var ast expression_parser.ASTWithSource
	if isHostBinding {
		ast = b.exprParser.ParseSimpleBinding(value, sourceSpan, absoluteOffset)
	} else {
		ast = b.exprParser.ParseBinding(value, sourceSpan, absoluteOffset)
	}
	if ast.Ast != nil {
		for _, e := range ast.Errors {
			b.Errors = append(b.Errors, ParseError{Span: e.Span, Message: e.Message, Level: ParseErrorLevelError})
		}
		return ast
	}
	val := "ERROR"
	return b.exprParser.WrapLiteralPrimitive(&val, sourceSpan, absoluteOffset)
}

func (b *BindingParser) CreateBoundElementProperty(
	elementSelector *string,
	boundProp ParsedProperty,
	skipValidation bool,
	mapPropertyName bool,
) BoundElementProperty {
	if boundProp.IsLegacyAnimation() {
		return NewBoundElementProperty(
			boundProp.Name,
			BindingTypeLegacyAnimation,
			core.SecurityContextNone,
			boundProp.Expression,
			nil,
			boundProp.SourceSpan,
			boundProp.KeySpan,
			boundProp.ValueSpan,
		)
	}

	var unit *string = nil
	var bindingType BindingType
	var boundPropertyName *string = nil
	parts := strings.Split(boundProp.Name, PROPERTY_PARTS_SEPARATOR)
	var securityContexts []core.SecurityContext

	if len(parts) > 1 {
		if parts[0] == ATTRIBUTE_PREFIX {
			nameJoin := strings.Join(parts[1:], PROPERTY_PARTS_SEPARATOR)
			boundPropertyName = &nameJoin
			if !skipValidation {
				b.validatePropertyOrAttributeName(*boundPropertyName, boundProp.SourceSpan, true)
			}
			securityContexts = CalcPossibleSecurityContexts(b.schemaRegistry, elementSelector, *boundPropertyName, true)

			nsSeparatorIdx := strings.Index(*boundPropertyName, ":")
			if nsSeparatorIdx > -1 {
				ns := (*boundPropertyName)[:nsSeparatorIdx]
				name := (*boundPropertyName)[nsSeparatorIdx+1:]
				*boundPropertyName = mergeNsAndName(ns, name)
			}
			bindingType = BindingTypeAttribute
		} else if parts[0] == CLASS_PREFIX {
			boundPropertyName = &parts[1]
			bindingType = BindingTypeClass
			securityContexts = []core.SecurityContext{core.SecurityContextNone}
		} else if parts[0] == STYLE_PREFIX {
			if len(parts) > 2 {
				unit = &parts[2]
			}
			boundPropertyName = &parts[1]
			bindingType = BindingTypeStyle
			securityContexts = []core.SecurityContext{core.SecurityContextStyle}
		} else if parts[0] == ANIMATE_PREFIX {
			boundPropertyName = &boundProp.Name
			bindingType = BindingTypeAnimation
			securityContexts = []core.SecurityContext{core.SecurityContextNone}
		}
	}

	if boundPropertyName == nil {
		mappedPropName := b.schemaRegistry.GetMappedPropName(boundProp.Name)
		nameToUse := boundProp.Name
		if mapPropertyName {
			nameToUse = mappedPropName
		}
		boundPropertyName = &nameToUse
		securityContexts = CalcPossibleSecurityContexts(b.schemaRegistry, elementSelector, mappedPropName, false)
		if boundProp.Type == ParsedPropertyTypeTwoWay {
			bindingType = BindingTypeTwoWay
		} else {
			bindingType = BindingTypeProperty
		}
		if !skipValidation {
			b.validatePropertyOrAttributeName(mappedPropName, boundProp.SourceSpan, false)
		}
	}

	secCtx := core.SecurityContextNone
	if len(securityContexts) > 0 {
		secCtx = securityContexts[0]
	}

	return NewBoundElementProperty(
		*boundPropertyName,
		bindingType,
		secCtx,
		boundProp.Expression,
		unit,
		boundProp.SourceSpan,
		boundProp.KeySpan,
		boundProp.ValueSpan,
	)
}

func (b *BindingParser) ParseEvent(
	name string,
	expression string,
	isAssignmentEvent bool,
	sourceSpan expression_parser.ParseSourceSpan,
	handlerSpan expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetEvents *[]ParsedEvent,
	keySpan expression_parser.ParseSourceSpan,
) {
	if len(name) == 0 {
		b.reportError("Event name is missing in binding", sourceSpan, ParseErrorLevelError)
	}

	if isLegacyAnimationLabel(name) {
		name = name[1:]
		keySpan = moveParseSourceSpan(keySpan, expression_parser.AbsoluteSourceSpan{Start: keySpan.Start + 1, End: keySpan.End})
		b.parseLegacyAnimationEvent(name, expression, sourceSpan, handlerSpan, targetEvents, keySpan)
	} else {
		b.parseRegularEvent(name, expression, isAssignmentEvent, sourceSpan, handlerSpan, targetMatchableAttrs, targetEvents, keySpan)
	}
}

func (b *BindingParser) CalcPossibleSecurityContexts(selector string, propName string, isAttribute bool) []core.SecurityContext {
	prop := b.schemaRegistry.GetMappedPropName(propName)
	sel := &selector
	return CalcPossibleSecurityContexts(b.schemaRegistry, sel, prop, isAttribute)
}

func (b *BindingParser) parseEventListenerName(rawName string) (string, *string) {
	target, eventName := splitAtColon(rawName, []*string{nil, &rawName})
	return *eventName, target
}

func (b *BindingParser) parseLegacyAnimationEventName(rawName string) (string, *string) {
	matches0, matches1 := splitAtPeriod(rawName, []*string{&rawName, nil})
	var phase *string
	if matches1 != nil {
		ph := strings.ToLower(*matches1)
		phase = &ph
	}
	return *matches0, phase
}

func (b *BindingParser) parseLegacyAnimationEvent(
	name string,
	expression string,
	sourceSpan expression_parser.ParseSourceSpan,
	handlerSpan expression_parser.ParseSourceSpan,
	targetEvents *[]ParsedEvent,
	keySpan expression_parser.ParseSourceSpan,
) {
	eventName, phase := b.parseLegacyAnimationEventName(name)
	ast := b.parseAction(expression, handlerSpan)
	*targetEvents = append(*targetEvents, NewParsedEvent(eventName, phase, ParsedEventTypeLegacyAnimation, ast, sourceSpan, handlerSpan, keySpan))

	if len(eventName) == 0 {
		b.reportError("Animation event name is missing in binding", sourceSpan, ParseErrorLevelError)
	}
	if phase != nil {
		if *phase != "start" && *phase != "done" {
			b.reportError(fmt.Sprintf(`The provided animation output phase value "%s" for "@%s" is not supported (use start or done)`, *phase, eventName), sourceSpan, ParseErrorLevelError)
		}
	} else {
		b.reportError(fmt.Sprintf(`The animation trigger output event (@%s) is missing its phase value name (start or done are currently supported)`, eventName), sourceSpan, ParseErrorLevelError)
	}
}

func (b *BindingParser) parseRegularEvent(
	name string,
	expression string,
	isAssignmentEvent bool,
	sourceSpan expression_parser.ParseSourceSpan,
	handlerSpan expression_parser.ParseSourceSpan,
	targetMatchableAttrs *[][]string,
	targetEvents *[]ParsedEvent,
	keySpan expression_parser.ParseSourceSpan,
) {
	eventName, target := b.parseEventListenerName(name)
	prevErrorCount := len(b.Errors)
	ast := b.parseAction(expression, handlerSpan)
	isValid := len(b.Errors) == prevErrorCount
	*targetMatchableAttrs = append(*targetMatchableAttrs, []string{name, ast.Source})

	if isAssignmentEvent && isValid && !b.isAllowedAssignmentEvent(ast.Ast) {
		b.reportError("Unsupported expression in a two-way binding", sourceSpan, ParseErrorLevelError)
	}

	eventType := ParsedEventTypeRegular
	if isAssignmentEvent {
		eventType = ParsedEventTypeTwoWay
	}
	if strings.HasPrefix(name, ANIMATE_PREFIX+PROPERTY_PARTS_SEPARATOR) {
		eventType = ParsedEventTypeAnimation
	}

	*targetEvents = append(*targetEvents, NewParsedEvent(eventName, target, eventType, ast, sourceSpan, handlerSpan, keySpan))
}

func (b *BindingParser) parseAction(value string, sourceSpan expression_parser.ParseSourceSpan) expression_parser.ASTWithSource {
	absoluteOffset := sourceSpan.Start
	ast := b.exprParser.ParseAction(value, sourceSpan, absoluteOffset)
	if ast.Ast != nil {
		for _, e := range ast.Errors {
			b.Errors = append(b.Errors, ParseError{Span: e.Span, Message: e.Message, Level: ParseErrorLevelError})
		}
		if _, empty := ast.Ast.(*expression_parser.EmptyExpr); empty {
			b.reportError("Empty expressions are not allowed", sourceSpan, ParseErrorLevelError)
			val := "ERROR"
			return b.exprParser.WrapLiteralPrimitive(&val, sourceSpan, absoluteOffset)
		}
		return ast
	}
	val := "ERROR"
	return b.exprParser.WrapLiteralPrimitive(&val, sourceSpan, absoluteOffset)
}

func (b *BindingParser) reportError(message string, sourceSpan expression_parser.ParseSourceSpan, level ParseErrorLevel) {
	b.Errors = append(b.Errors, NewParseError(sourceSpan, message, level))
}

func (b *BindingParser) validatePropertyOrAttributeName(propName string, sourceSpan expression_parser.ParseSourceSpan, isAttr bool) {
	var report struct {
		Error bool
		Msg   *string
	}
	if isAttr {
		report = b.schemaRegistry.ValidateAttribute(propName)
	} else {
		report = b.schemaRegistry.ValidateProperty(propName)
	}
	if report.Error {
		msg := ""
		if report.Msg != nil {
			msg = *report.Msg
		}
		b.reportError(msg, sourceSpan, ParseErrorLevelError)
	}
}

func (b *BindingParser) isAllowedAssignmentEvent(ast expression_parser.AST) bool {
	if a, ok := ast.(*expression_parser.ASTWithSource); ok {
		return b.isAllowedAssignmentEvent(a.Ast)
	}

	if a, ok := ast.(*expression_parser.NonNullAssert); ok {
		return b.isAllowedAssignmentEvent(a.Expression)
	}

	if a, ok := ast.(*expression_parser.Call); ok {
		if len(a.Args) == 1 {
			if recv, ok := a.Receiver.(*expression_parser.PropertyRead); ok && recv.Name == "$any" {
				if _, ok := recv.Receiver.(*expression_parser.ImplicitReceiver); ok {
					return b.isAllowedAssignmentEvent(a.Args[0])
				}
			}
		}
	}

	switch ast.(type) {
	case *expression_parser.PropertyRead, *expression_parser.KeyedRead:
		if !hasRecursiveSafeReceiver(ast) {
			return true
		}
	}

	return false
}

func hasRecursiveSafeReceiver(ast expression_parser.AST) bool {
	switch a := ast.(type) {
	case *expression_parser.SafePropertyRead, *expression_parser.SafeKeyedRead:
		return true
	case *expression_parser.ParenthesizedExpression:
		return hasRecursiveSafeReceiver(a.Expression)
	case *expression_parser.PropertyRead:
		return hasRecursiveSafeReceiver(a.Receiver)
	case *expression_parser.KeyedRead:
		return hasRecursiveSafeReceiver(a.Receiver)
	case *expression_parser.Call:
		return hasRecursiveSafeReceiver(a.Receiver)
	}
	return false
}

func CalcPossibleSecurityContexts(
	registry ElementSchemaRegistry,
	selector *string,
	propName string,
	isAttribute bool,
) []core.SecurityContext {
	var ctxs []core.SecurityContext

	namespaceKey, baseSelector := splitNsName("", false)
	if selector != nil {
		namespaceKey, baseSelector = splitNsName(*selector, false)
	}

	nameToContext := func(elName string) core.SecurityContext {
		nsStr, name := splitNsName(elName, false)
		ns := nsStr
		if ns == nil {
			ns = namespaceKey
		}
		fullName := name
		if ns != nil && *ns != "" {
			fullName = ":" + *ns + ":" + name
		}
		return registry.SecurityContext(fullName, propName, isAttribute)
	}

	allKnownElements := registry.AllKnownElementNames()
	if baseSelector == "" {
		for _, name := range allKnownElements {
			ctxs = append(ctxs, nameToContext(name))
		}
	} else {
		for _, sel := range ParseCssSelector(baseSelector) {
			var elementNames []string
			if sel.Element != nil {
				elementNames = []string{*sel.Element}
			} else {
				elementNames = allKnownElements
			}

			if sel.Element != nil && !registry.HasElement(*sel.Element, []string{}) {
				svgElement := ":" + core.SVG_NAMESPACE + ":" + *sel.Element
				mathElement := ":" + core.MATH_ML_NAMESPACE + ":" + *sel.Element
				if registry.HasElement(svgElement, []string{}) {
					elementNames = []string{svgElement}
				} else if registry.HasElement(mathElement, []string{}) {
					elementNames = []string{mathElement}
				}
			}

			notElementNames := make(map[string]bool)
			for _, notSel := range sel.NotSelectors {
				if notSel.IsElementSelector() && notSel.Element != nil {
					notElementNames[strings.ToLower(*notSel.Element)] = true
				}
			}

			for _, elName := range elementNames {
				elNameLowerCase := strings.ToLower(elName)
				_, namePart := splitNsName(elNameLowerCase, false)
				if !notElementNames[elNameLowerCase] && !notElementNames[namePart] {
					ctxs = append(ctxs, nameToContext(elName))
				}
			}
		}
	}

	if len(ctxs) == 0 {
		return []core.SecurityContext{core.SecurityContextNone}
	}

	// Deduplicate and sort
	ctxMap := make(map[core.SecurityContext]bool)
	for _, ctx := range ctxs {
		ctxMap[ctx] = true
	}
	var uniqueCtxs []core.SecurityContext
	for ctx := range ctxMap {
		uniqueCtxs = append(uniqueCtxs, ctx)
	}
	sort.Slice(uniqueCtxs, func(i, j int) bool {
		return uniqueCtxs[i] < uniqueCtxs[j]
	})
	return uniqueCtxs
}
