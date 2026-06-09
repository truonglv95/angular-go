package render3

import (
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	i18n_pkg "github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"

	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

func isI18nRootNode(meta i18n_pkg.I18nMeta) bool {
	if meta == nil {
		return false
	}
	_, ok := meta.(*i18n_pkg.Message)
	return ok
}

var BIND_NAME_REGEXP = regexp.MustCompile(`^(?:(bind-)|(let-)|(ref-|#)|(on-)|(bindon-)|(@))(.*)$`)

const (
	KW_BIND_IDX   = 1
	KW_LET_IDX    = 2
	KW_REF_IDX    = 3
	KW_ON_IDX     = 4
	KW_BINDON_IDX = 5
	KW_AT_IDX     = 6
	IDENT_KW_IDX  = 7
)

type Delims struct {
	Start string
	End   string
}

var BINDING_DELIMS = map[string]Delims{
	"BANANA_BOX": {Start: "[(", End: ")]"},
	"PROPERTY":   {Start: "[", End: "]"},
	"EVENT":      {Start: "(", End: ")"},
}

const TEMPLATE_ATTR_PREFIX = "*"

var UNSUPPORTED_SELECTORLESS_TAGS = map[string]bool{
	"link":         true,
	"style":        true,
	"script":       true,
	"ng-template":  true,
	"ng-container": true,
	"ng-content":   true,
}

var UNSUPPORTED_SELECTORLESS_DIRECTIVE_ATTRS = map[string]bool{
	"ngProjectAs":   true,
	"ngNonBindable": true,
}

type Render3ParseResult struct {
	Nodes              []Node
	Errors             []*parse_util.ParseError
	Styles             []string
	StyleUrls          []string
	NgContentSelectors []string
	CommentNodes       []*Comment
}

type Render3ParseOptions struct {
	CollectCommentNodes bool
}

func HtmlAstToRender3Ast(htmlNodes []ml_parser.Node, bindingParser *template_parser.BindingParser, options Render3ParseOptions) Render3ParseResult {
	transformer := NewHtmlAstToIvyAst(bindingParser, options)
	if len(htmlNodes) > 0 {
		transformer.baseSpan = htmlNodes[0].GetSourceSpan()
	}
	ivyNodes := ml_parser.VisitAll(transformer, htmlNodes, htmlNodes)

	allErrors := []*parse_util.ParseError{}
	allErrors = append(allErrors, transformer.errors...)

	mappedIvyNodes := make([]Node, 0)
	for _, n := range ivyNodes {
		if n != nil {
			mappedIvyNodes = append(mappedIvyNodes, n.(Node))
		}
	}

	result := Render3ParseResult{
		Nodes:              mappedIvyNodes,
		Errors:             allErrors,
		StyleUrls:          transformer.styleUrls,
		Styles:             transformer.styles,
		NgContentSelectors: transformer.ngContentSelectors,
	}
	if options.CollectCommentNodes {
		result.CommentNodes = transformer.commentNodes
	}
	return result
}

type HtmlAstToIvyAst struct {
	errors             []*parse_util.ParseError
	styles             []string
	styleUrls          []string
	ngContentSelectors []string
	commentNodes       []*Comment
	inI18nBlock        bool
	processedNodes     map[ml_parser.Node]bool

	bindingParser *template_parser.BindingParser
	options       Render3ParseOptions
	baseSpan      *parse_util.ParseSourceSpan
}

func NewHtmlAstToIvyAst(bindingParser *template_parser.BindingParser, options Render3ParseOptions) *HtmlAstToIvyAst {
	return &HtmlAstToIvyAst{
		errors:             []*parse_util.ParseError{},
		styles:             []string{},
		styleUrls:          []string{},
		ngContentSelectors: []string{},
		commentNodes:       []*Comment{},
		inI18nBlock:        false,
		processedNodes:     make(map[ml_parser.Node]bool),
		bindingParser:      bindingParser,
		options:            options,
	}
}

func (v *HtmlAstToIvyAst) VisitElement(element *ml_parser.Element, context any) any {
	isI18nRootElement := isI18nRootNode(element.I18n)
	if isI18nRootElement {
		if v.inI18nBlock {
			v.reportError(
				"Cannot mark an element as translatable inside of a translatable section. Please remove the nested i18n marker.",
				*element.SourceSpan,
			)
		}
		v.inI18nBlock = true
	}

	preparsedElement := template_parser.PreparseElement(element)
	if preparsedElement.Type == template_parser.PreparsedElementTypeScript {
		return nil
	} else if preparsedElement.Type == template_parser.PreparsedElementTypeStyle {
		contents := textContents(element)
		if contents != nil {
			v.styles = append(v.styles, *contents)
		}
		return nil
	} else if preparsedElement.Type == template_parser.PreparsedElementTypeStylesheet &&
		preparsedElement.HrefAttr != nil {
		v.styleUrls = append(v.styleUrls, *preparsedElement.HrefAttr)
		return nil
	}

	isTemplateElement := ml_parser.IsNgTemplate(element.Name)
	attrs := v.prepareAttributes(element.Attrs, isTemplateElement)

	directives := v.extractDirectives(element)
	var children []Node

	if preparsedElement.NonBindable {
		flatChildren := ml_parser.VisitAll(NON_BINDABLE_VISITOR, element.Children, nil)
		for _, c := range flatChildren {
			if c != nil {
				// Flatten logic here is simplified for slice
				switch c := c.(type) {
				case []Node:
					children = append(children, c...)
				case Node:
					children = append(children, c)
				}
			}
		}
	} else {
		flatChildren := ml_parser.VisitAll(v, element.Children, element.Children)
		for _, c := range flatChildren {
			if c != nil {
				children = append(children, c.(Node))
			}
		}
	}

	var parsedElement Node
	if preparsedElement.Type == template_parser.PreparsedElementTypeNgContent {
		selector := preparsedElement.SelectAttr
		var elementAttrs []*TextAttribute
		for _, attr := range element.Attrs {
			elementAttrs = append(elementAttrs, v.VisitAttribute(attr, nil).(*TextAttribute))
		}
		parsedElement = NewContent(
			selector,
			elementAttrs,
			children,
			element.IsSelfClosing,
			*element.SourceSpan,
			*element.StartSourceSpan,
			element.EndSourceSpan,
			element.I18n,
		)
		v.ngContentSelectors = append(v.ngContentSelectors, selector)
	} else if isTemplateElement {
		categorized := v.categorizePropertyAttributes(
			&element.Name,
			attrs.parsedProperties,
			attrs.i18nAttrsMeta,
		)

		parsedElement = &Template{
			TagName:         &element.Name,
			Attributes:      attrs.attributes,
			Inputs:          categorized.bound,
			Outputs:         attrs.boundEvents,
			Directives:      directives,
			TemplateAttrs:   []Node{},
			Children:        children,
			References:      attrs.references,
			Variables:       attrs.variables,
			IsSelfClosing:   element.IsSelfClosing,
			SourceSpan:      *element.SourceSpan,
			StartSourceSpan: *element.StartSourceSpan,
			EndSourceSpan:   element.EndSourceSpan,
			I18n:            element.I18n,
		}
	} else {
		categorized := v.categorizePropertyAttributes(
			&element.Name,
			attrs.parsedProperties,
			attrs.i18nAttrsMeta,
		)

		if element.Name == "ng-container" {
			for _, bound := range categorized.bound {
				if int(bound.Type) == 0 {
					v.reportError(
						"Attribute bindings are not supported on ng-container. Use property bindings instead.",
						bound.SourceSpan,
					)
				}
			}
		}

		parsedElement = &Element{
			Name:            element.Name,
			Attributes:      attrs.attributes,
			Inputs:          categorized.bound,
			Outputs:         attrs.boundEvents,
			Directives:      directives,
			Children:        children,
			References:      attrs.references,
			IsSelfClosing:   element.IsSelfClosing,
			SourceSpan:      *element.SourceSpan,
			StartSourceSpan: *element.StartSourceSpan,
			EndSourceSpan:   element.EndSourceSpan,
			IsVoid:          element.IsVoid,
			I18n:            element.I18n,
		}
	}

	if attrs.elementHasInlineTemplate {
		parsedElement = v.wrapInTemplate(
			parsedElement,
			attrs.templateParsedProperties,
			attrs.templateVariables,
			attrs.i18nAttrsMeta,
			isTemplateElement,
			isI18nRootElement,
		)
	}
	if isI18nRootElement {
		v.inI18nBlock = false
	}
	return parsedElement
}

func (v *HtmlAstToIvyAst) VisitAttribute(attribute *ml_parser.Attribute, context any) any {
	return &TextAttribute{
		Name:       attribute.Name,
		Value:      attribute.Value,
		SourceSpan: *attribute.SourceSpan,
		KeySpan:    attribute.KeySpan,
		ValueSpan:  attribute.ValueSpan,
		I18n:       attribute.I18n,
	}
}

func (v *HtmlAstToIvyAst) VisitText(text *ml_parser.Text, context any) any {
	if v.processedNodes[text] {
		return nil
	}
	return v._visitTextWithInterpolation(text.Value, *text.SourceSpan, text.Tokens, text.I18n)
}

func (v *HtmlAstToIvyAst) VisitExpansion(expansion *ml_parser.Expansion, context any) any {
	if expansion.I18n == nil {
		return nil
	}
	if !isI18nRootNode(expansion.I18n) {
		panic("Invalid type for i18n property")
	}
	message, ok := expansion.I18n.(*i18n_pkg.Message)
	if !ok {
		return nil
	}
	vars := make(map[string]*BoundText)
	placeholders := make(map[string]Node)

	const i18nIcuVarPrefix = "VAR_"
	for key, value := range message.Placeholders {
		if value == nil || value.SourceSpan == nil {
			continue
		}
		if strings.HasPrefix(key, i18nIcuVarPrefix) {
			formattedKey := strings.TrimSpace(key)
			ast := v.bindingParser.ParseInterpolationExpression(value.Text, toExpressionParserParseSourceSpan(*value.SourceSpan))
			vars[formattedKey] = &BoundText{Value: &ast, SourceSpan: *value.SourceSpan}
		} else {
			placeholders[key] = v._visitTextWithInterpolation(value.Text, *value.SourceSpan, nil, nil)
		}
	}
	return &Icu{
		Vars:         vars,
		Placeholders: placeholders,
		SourceSpan:   *expansion.SourceSpan,
		I18n:         message,
	}
}

func (v *HtmlAstToIvyAst) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	return nil
}

func (v *HtmlAstToIvyAst) VisitComment(comment *ml_parser.Comment, context any) any {
	if v.options.CollectCommentNodes {
		v.commentNodes = append(v.commentNodes, &Comment{
			Value:      comment.Value,
			SourceSpan: *comment.SourceSpan,
		})
	}
	return nil
}

func (v *HtmlAstToIvyAst) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	value := v.bindingParser.ParseBinding(decl.Value, false, expression_parser.ParseSourceSpan{}, decl.ValueSpan.FullStart.Offset)

	if len(value.Errors) == 0 {
		if _, empty := value.Ast.(*expression_parser.EmptyExpr); empty {
			v.reportError("@let declaration value cannot be empty", *decl.ValueSpan)
		}
	}

	return &LetDeclaration{
		Name:       decl.Name,
		Value:      &value,
		SourceSpan: decl.SourceSpan,
		NameSpan:   *decl.NameSpan,
		ValueSpan:  *decl.ValueSpan,
	}
}

func (v *HtmlAstToIvyAst) VisitComponent(component *ml_parser.Component, context any) any {
	isI18nRootElement := isI18nRootNode(component.I18n)
	if isI18nRootElement {
		if v.inI18nBlock {
			v.reportError(
				"Cannot mark a component as translatable inside of a translatable section. Please remove the nested i18n marker.",
				*component.SourceSpan,
			)
		}
		v.inI18nBlock = true
	}

	if UNSUPPORTED_SELECTORLESS_TAGS[component.TagName] {
		v.reportError(
			"Tag name cannot be used as a component tag",
			*component.StartSourceSpan,
		)
		return nil
	}

	attrs := v.prepareAttributes(component.Attrs, false)
	v.validateSelectorlessReferences(attrs.references)

	directives := v.extractDirectives(&component.Element)
	var children []Node

	hasNonBindable := false
	for _, attr := range component.Attrs {
		if attr.Name == "ngNonBindable" {
			hasNonBindable = true
			break
		}
	}

	if hasNonBindable {
		flatChildren := ml_parser.VisitAll(NON_BINDABLE_VISITOR, component.Children, nil)
		for _, c := range flatChildren {
			if c != nil {
				switch c := c.(type) {
				case []Node:
					children = append(children, c...)
				case Node:
					children = append(children, c)
				}
			}
		}
	} else {
		flatChildren := ml_parser.VisitAll(v, component.Children, component.Children)
		for _, c := range flatChildren {
			if c != nil {
				children = append(children, c.(Node))
			}
		}
	}

	categorized := v.categorizePropertyAttributes(
		&component.TagName,
		attrs.parsedProperties,
		attrs.i18nAttrsMeta,
	)

	var node Node = &Component{
		ComponentName:   component.ComponentName,
		TagName:         &component.TagName,
		FullName:        component.FullName,
		Attributes:      attrs.attributes,
		Inputs:          categorized.bound,
		Outputs:         attrs.boundEvents,
		Directives:      directives,
		Children:        children,
		References:      attrs.references,
		IsSelfClosing:   component.IsSelfClosing,
		SourceSpan:      *component.SourceSpan,
		StartSourceSpan: *component.StartSourceSpan,
		EndSourceSpan:   component.EndSourceSpan,
		I18n:            component.I18n,
	}

	if attrs.elementHasInlineTemplate {
		node = v.wrapInTemplate(
			node,
			attrs.templateParsedProperties,
			attrs.templateVariables,
			attrs.i18nAttrsMeta,
			false,
			isI18nRootElement,
		)
	}
	if isI18nRootElement {
		v.inI18nBlock = false
	}
	return node
}

func (v *HtmlAstToIvyAst) VisitDirective(directive *ml_parser.Directive, context any) any {
	return nil
}

func (v *HtmlAstToIvyAst) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}

func (v *HtmlAstToIvyAst) VisitBlock(block *ml_parser.Block, context any) any {
	siblings := context.([]ml_parser.Node)
	index := -1
	for i, sibling := range siblings {
		if sibling == block {
			index = i
			break
		}
	}
	if index == -1 {
		panic("Visitor invoked incorrectly. Expecting visitBlock to be invoked siblings array as its context")
	}

	if v.processedNodes[block] {
		return nil
	}

	var node Node
	var errors []*parse_util.ParseError

	switch block.Name {
	case "defer":
		node, errors = createDeferredBlock(
			block,
			v.findConnectedBlocks(index, siblings, isConnectedDeferLoopBlock),
			v,
			v.bindingParser,
		)
	case "switch":
		node, errors = createSwitchBlock(block, v, v.bindingParser)
	case "for":
		node, errors = createForLoop(
			block,
			v.findConnectedBlocks(index, siblings, isConnectedForLoopBlock),
			v,
			v.bindingParser,
		)
	case "if":
		node, errors = createIfBlock(
			block,
			v.findConnectedBlocks(index, siblings, isConnectedIfLoopBlock),
			v,
			v.bindingParser,
		)
	default:
		var errorMessage string
		if isConnectedDeferLoopBlock(block.Name) {
			errorMessage = "@" + block.Name + " block can only be used after an @defer block."
			v.processedNodes[block] = true
		} else if isConnectedForLoopBlock(block.Name) {
			errorMessage = "@" + block.Name + " block can only be used after an @for block."
			v.processedNodes[block] = true
		} else if isConnectedIfLoopBlock(block.Name) {
			errorMessage = "@" + block.Name + " block can only be used after an @if or @else if block."
			v.processedNodes[block] = true
		} else {
			errorMessage = "Unrecognized block @" + block.Name + "."
		}
		node = &UnknownBlock{Name: block.Name, SourceSpan: *block.SourceSpan, NameSpan: *block.SourceSpan}
		errors = []*parse_util.ParseError{parse_util.NewParseError(block.SourceSpan, errorMessage, nil, nil)}
	}

	v.errors = append(v.errors, errors...)
	return node
}

func (v *HtmlAstToIvyAst) findConnectedBlocks(primaryBlockIndex int, siblings []ml_parser.Node, predicate func(string) bool) []*ml_parser.Block {
	var relatedBlocks []*ml_parser.Block
	for i := primaryBlockIndex + 1; i < len(siblings); i++ {
		node := siblings[i]

		if _, ok := node.(*ml_parser.Comment); ok {
			continue
		}

		if textNode, ok := node.(*ml_parser.Text); ok && len(strings.TrimSpace(textNode.Value)) == 0 {
			v.processedNodes[node] = true
			continue
		}

		blockNode, ok := node.(*ml_parser.Block)
		if !ok || !predicate(blockNode.Name) {
			break
		}

		relatedBlocks = append(relatedBlocks, blockNode)
		v.processedNodes[node] = true
	}
	return relatedBlocks
}

func (v *HtmlAstToIvyAst) categorizePropertyAttributes(
	elementName *string,
	properties []*template_parser.ParsedProperty,
	i18nPropsMeta map[string]any,
) struct {
	bound   []*BoundAttribute
	literal []*TextAttribute
} {
	var bound []*BoundAttribute
	var literal []*TextAttribute

	for _, prop := range properties {
		i18n := i18nPropsMeta[prop.Name]
		if prop.IsLiteral() {
			keySpan := toParseSourceSpan(prop.KeySpan, v.baseSpan)
			literal = append(literal, &TextAttribute{
				Name:       prop.Name,
				Value:      prop.Expression.Source,
				SourceSpan: toParseSourceSpan(prop.SourceSpan, v.baseSpan),
				KeySpan:    &keySpan,
				ValueSpan:  toParseSourceSpanPtr(prop.ValueSpan, v.baseSpan),
				I18n:       i18n,
			})
		} else {
			isAttrOn := strings.HasPrefix(strings.ToLower(prop.Name), "attr.on")
			bep := v.bindingParser.CreateBoundElementProperty(
				elementName,
				*prop,
				!isAttrOn,
				false,
			)
			bound = append(bound, BoundAttributeFromBoundElementProperty(bep, i18n, v.baseSpan))
		}
	}

	return struct {
		bound   []*BoundAttribute
		literal []*TextAttribute
	}{bound, literal}
}

type preparedAttributes struct {
	attributes               []*TextAttribute
	boundEvents              []*BoundEvent
	references               []*Reference
	variables                []*Variable
	templateVariables        []*Variable
	elementHasInlineTemplate bool
	parsedProperties         []*template_parser.ParsedProperty
	templateParsedProperties []*template_parser.ParsedProperty
	i18nAttrsMeta            map[string]any
}

func (v *HtmlAstToIvyAst) prepareAttributes(attrs []*ml_parser.Attribute, isTemplateElement bool) preparedAttributes {
	var parsedProperties []*template_parser.ParsedProperty
	var boundEvents []*BoundEvent
	var variables []*Variable
	var references []*Reference
	var attributes []*TextAttribute
	i18nAttrsMeta := make(map[string]any)
	var templateParsedProperties []*template_parser.ParsedProperty
	var templateVariables []*Variable

	elementHasInlineTemplate := false

	for _, attribute := range attrs {
		hasBinding := false
		isTemplateBinding := false

		if attribute.I18n != nil {
			i18nAttrsMeta[attribute.Name] = attribute.I18n
		}

		if strings.HasPrefix(attribute.Name, TEMPLATE_ATTR_PREFIX) {
			if elementHasInlineTemplate {
				v.reportError(
					"Can't have multiple template bindings on one element. Use only one attribute prefixed with *",
					*attribute.SourceSpan,
				)
			}
			isTemplateBinding = true
			elementHasInlineTemplate = true
			templateValue := attribute.Value
			templateKey := attribute.Name[len(TEMPLATE_ATTR_PREFIX):]

			var tempProps []template_parser.ParsedProperty
			var parsedVariables []template_parser.ParsedVariable
			absoluteValueOffset := attribute.SourceSpan.FullStart.Offset + len(attribute.Name)
			if attribute.ValueSpan != nil {
				absoluteValueOffset = attribute.ValueSpan.FullStart.Offset
			}

			srcSpan := toExpressionParserParseSourceSpan(*attribute.SourceSpan)
			var matchableAttrs [][]string
			v.bindingParser.ParseInlineTemplateBinding(
				templateKey,
				templateValue,
				srcSpan,
				absoluteValueOffset,
				&matchableAttrs,
				&tempProps,
				&parsedVariables,
				true, // isIvyAst
			)

			for i := range tempProps {
				templateParsedProperties = append(templateParsedProperties, &tempProps[i])
			}

			for _, pvar := range parsedVariables {
				pvarSrcSpan := toParseSourceSpan(pvar.SourceSpan, v.baseSpan)
				pvarKeySpan := toParseSourceSpan(pvar.KeySpan, v.baseSpan)
				var pvarValSpan *parse_util.ParseSourceSpan
				if pvar.ValueSpan != nil {
					s := toParseSourceSpan(*pvar.ValueSpan, v.baseSpan)
					pvarValSpan = &s
				}
				templateVariables = append(templateVariables, &Variable{
					Name:       pvar.Name,
					Value:      pvar.Value,
					SourceSpan: pvarSrcSpan,
					KeySpan:    pvarKeySpan,
					ValueSpan:  pvarValSpan,
				})
			}
		} else {
			hasBinding = v.parseAttribute(
				isTemplateElement,
				attribute,
				nil,
				&parsedProperties,
				&boundEvents,
				&variables,
				&references,
			)
		}

		if !hasBinding && !isTemplateBinding {
			attributes = append(attributes, v.VisitAttribute(attribute, nil).(*TextAttribute))
		}
	}

	return preparedAttributes{
		attributes:               attributes,
		boundEvents:              boundEvents,
		references:               references,
		variables:                variables,
		templateVariables:        templateVariables,
		elementHasInlineTemplate: elementHasInlineTemplate,
		parsedProperties:         parsedProperties,
		templateParsedProperties: templateParsedProperties,
		i18nAttrsMeta:            i18nAttrsMeta,
	}
}

func (v *HtmlAstToIvyAst) parseAttribute(
	isTemplateElement bool,
	attribute *ml_parser.Attribute,
	matchableAttributes [][]string,
	parsedProperties *[]*template_parser.ParsedProperty,
	boundEvents *[]*BoundEvent,
	variables *[]*Variable,
	references *[]*Reference,
) bool {
	name := attribute.Name
	value := attribute.Value
	srcSpan := *attribute.SourceSpan
	absoluteOffset := srcSpan.FullStart.Offset
	if attribute.ValueSpan != nil {
		absoluteOffset = attribute.ValueSpan.FullStart.Offset
	}

	createKeySpan := func(srcSpan parse_util.ParseSourceSpan, prefix string, identifier string) parse_util.ParseSourceSpan {
		keySpanStart := srcSpan.Start.MoveBy(len(prefix))
		keySpanEnd := keySpanStart.MoveBy(len(identifier))
		return *parse_util.NewParseSourceSpan(keySpanStart, keySpanEnd, keySpanStart, &identifier)
	}

	bindParts := BIND_NAME_REGEXP.FindStringSubmatch(name)

	if bindParts != nil {
		if bindParts[KW_BIND_IDX] != "" {
			identifier := bindParts[IDENT_KW_IDX]
			keySpan := createKeySpan(srcSpan, bindParts[KW_BIND_IDX], identifier)
			var tempProps []template_parser.ParsedProperty
			valueSpan := toExpressionParserParseSourceSpanPtr(attribute.ValueSpan)
			v.bindingParser.ParsePropertyBinding(
				identifier,
				value,
				false,
				false,
				toExpressionParserParseSourceSpan(srcSpan),
				absoluteOffset,
				valueSpan,
				&matchableAttributes,
				&tempProps,
				toExpressionParserParseSourceSpan(keySpan),
			)
			for i := range tempProps {
				*parsedProperties = append(*parsedProperties, &tempProps[i])
			}
		} else if bindParts[KW_LET_IDX] != "" {
			if isTemplateElement {
				identifier := bindParts[IDENT_KW_IDX]
				keySpan := createKeySpan(srcSpan, bindParts[KW_LET_IDX], identifier)
				v.parseVariable(identifier, value, srcSpan, keySpan, attribute.ValueSpan, variables)
			} else {
				v.reportError(`"let-" is only supported on ng-template elements.`, srcSpan)
			}
		} else if bindParts[KW_REF_IDX] != "" {
			identifier := bindParts[IDENT_KW_IDX]
			keySpan := createKeySpan(srcSpan, bindParts[KW_REF_IDX], identifier)
			v.parseReference(identifier, value, srcSpan, keySpan, attribute.ValueSpan, references)
		} else if bindParts[KW_ON_IDX] != "" {
			identifier := bindParts[IDENT_KW_IDX]
			keySpan := createKeySpan(srcSpan, bindParts[KW_ON_IDX], identifier)
			var parsedEvents []template_parser.ParsedEvent
			valueSpan := srcSpan
			if attribute.ValueSpan != nil {
				valueSpan = *attribute.ValueSpan
			}
			v.bindingParser.ParseEvent(
				identifier,
				value,
				false,
				toExpressionParserParseSourceSpan(srcSpan),
				toExpressionParserParseSourceSpan(valueSpan),
				&matchableAttributes,
				&parsedEvents,
				toExpressionParserParseSourceSpan(keySpan),
			)
			var events []*template_parser.ParsedEvent
			for i := range parsedEvents {
				events = append(events, &parsedEvents[i])
			}
			v.addEvents(events, boundEvents)
		} else if bindParts[KW_BINDON_IDX] != "" {
			identifier := bindParts[IDENT_KW_IDX]
			keySpan := createKeySpan(srcSpan, bindParts[KW_BINDON_IDX], identifier)
			var tempProps []template_parser.ParsedProperty
			valueSpan := toExpressionParserParseSourceSpanPtr(attribute.ValueSpan)
			v.bindingParser.ParsePropertyBinding(
				identifier,
				value,
				false,
				true,
				toExpressionParserParseSourceSpan(srcSpan),
				absoluteOffset,
				valueSpan,
				&matchableAttributes,
				&tempProps,
				toExpressionParserParseSourceSpan(keySpan),
			)
			for i := range tempProps {
				*parsedProperties = append(*parsedProperties, &tempProps[i])
			}
			v.parseAssignmentEvent(
				identifier,
				value,
				srcSpan,
				attribute.ValueSpan,
				matchableAttributes,
				boundEvents,
				keySpan,
				absoluteOffset,
			)
		} else if bindParts[KW_AT_IDX] != "" {
			keySpan := createKeySpan(srcSpan, "", name)
			var tempProps []template_parser.ParsedProperty
			valueSpan := toExpressionParserParseSourceSpanPtr(attribute.ValueSpan)
			v.bindingParser.ParseLiteralAttr(
				name,
				&value,
				toExpressionParserParseSourceSpan(srcSpan),
				absoluteOffset,
				valueSpan,
				&matchableAttributes,
				&tempProps,
				toExpressionParserParseSourceSpan(keySpan),
			)
			for i := range tempProps {
				*parsedProperties = append(*parsedProperties, &tempProps[i])
			}
		}
		return true
	}

	var delims *Delims
	if strings.HasPrefix(name, BINDING_DELIMS["BANANA_BOX"].Start) {
		d := BINDING_DELIMS["BANANA_BOX"]
		delims = &d
	} else if strings.HasPrefix(name, BINDING_DELIMS["PROPERTY"].Start) {
		d := BINDING_DELIMS["PROPERTY"]
		delims = &d
	} else if strings.HasPrefix(name, BINDING_DELIMS["EVENT"].Start) {
		d := BINDING_DELIMS["EVENT"]
		delims = &d
	}
	if delims != nil &&
		strings.HasSuffix(name, delims.End) &&
		len(name) > len(delims.Start)+len(delims.End) {
		identifier := name[len(delims.Start) : len(name)-len(delims.End)]
		keySpan := createKeySpan(srcSpan, delims.Start, identifier)
		if delims.Start == BINDING_DELIMS["BANANA_BOX"].Start {
			var tempProps []template_parser.ParsedProperty
			valueSpan := toExpressionParserParseSourceSpanPtr(attribute.ValueSpan)
			v.bindingParser.ParsePropertyBinding(
				identifier,
				value,
				false,
				true,
				toExpressionParserParseSourceSpan(srcSpan),
				absoluteOffset,
				valueSpan,
				&matchableAttributes,
				&tempProps,
				toExpressionParserParseSourceSpan(keySpan),
			)
			for i := range tempProps {
				*parsedProperties = append(*parsedProperties, &tempProps[i])
			}
			v.parseAssignmentEvent(
				identifier,
				value,
				srcSpan,
				attribute.ValueSpan,
				matchableAttributes,
				boundEvents,
				keySpan,
				absoluteOffset,
			)
		} else if delims.Start == BINDING_DELIMS["PROPERTY"].Start {
			var tempProps []template_parser.ParsedProperty
			valueSpan := toExpressionParserParseSourceSpanPtr(attribute.ValueSpan)
			v.bindingParser.ParsePropertyBinding(
				identifier,
				value,
				false,
				false,
				toExpressionParserParseSourceSpan(srcSpan),
				absoluteOffset,
				valueSpan,
				&matchableAttributes,
				&tempProps,
				toExpressionParserParseSourceSpan(keySpan),
			)
			for i := range tempProps {
				*parsedProperties = append(*parsedProperties, &tempProps[i])
			}
		} else {
			var parsedEvents []template_parser.ParsedEvent
			valueSpan := srcSpan
			if attribute.ValueSpan != nil {
				valueSpan = *attribute.ValueSpan
			}
			v.bindingParser.ParseEvent(
				identifier,
				value,
				false,
				toExpressionParserParseSourceSpan(srcSpan),
				toExpressionParserParseSourceSpan(valueSpan),
				&matchableAttributes,
				&parsedEvents,
				toExpressionParserParseSourceSpan(keySpan),
			)
			var events []*template_parser.ParsedEvent
			for i := range parsedEvents {
				events = append(events, &parsedEvents[i])
			}
			v.addEvents(events, boundEvents)
		}

		return true
	}

	keySpan := createKeySpan(srcSpan, "", name)
	var tempProps []template_parser.ParsedProperty
	var valueTokens []interface{}
	if len(attribute.ValueTokens) > 0 {
		valueTokens = make([]interface{}, len(attribute.ValueTokens))
		for i, tok := range attribute.ValueTokens {
			valueTokens[i] = tok
		}
	}
	hasBinding := v.bindingParser.ParsePropertyInterpolation(
		name,
		value,
		toExpressionParserParseSourceSpan(srcSpan),
		toExpressionParserParseSourceSpanPtr(attribute.ValueSpan),
		&matchableAttributes,
		&tempProps,
		toExpressionParserParseSourceSpan(keySpan),
		valueTokens,
	)
	for i := range tempProps {
		*parsedProperties = append(*parsedProperties, &tempProps[i])
	}
	return hasBinding
}

func (v *HtmlAstToIvyAst) extractDirectives(node *ml_parser.Element) []*Directive {
	elementName := node.Name
	var directives []*Directive
	seenDirectives := make(map[string]bool)

	for _, directive := range node.GetDirectives() {
		invalid := false
		for _, attr := range directive.Attrs { // Assume directive.Attrs exists
			if strings.HasPrefix(attr.Name, TEMPLATE_ATTR_PREFIX) {
				invalid = true
				v.reportError(
					"Shorthand template syntax \""+attr.Name+"\" is not supported inside a directive context",
					*attr.SourceSpan,
				)
			} else if UNSUPPORTED_SELECTORLESS_DIRECTIVE_ATTRS[attr.Name] {
				invalid = true
				v.reportError(
					"Attribute \""+attr.Name+"\" is not supported in a directive context",
					*attr.SourceSpan,
				)
			}
		}

		if !invalid && seenDirectives[directive.Name] {
			invalid = true
			v.reportError(
				"Cannot apply directive \""+directive.Name+"\" multiple times on the same element",
				parse_util.ParseSourceSpan{},
			)
		}

		if invalid {
			continue
		}

		attrs := v.prepareAttributes(directive.Attrs, false)
		v.validateSelectorlessReferences(attrs.references)

		categorized := v.categorizePropertyAttributes(
			&elementName,
			attrs.parsedProperties,
			attrs.i18nAttrsMeta,
		)

		for _, input := range categorized.bound {
			if int(input.Type) != 0 && int(input.Type) != 2 {
				invalid = true
				v.reportError("Binding is not supported in a directive context", input.SourceSpan)
			}
		}

		if invalid {
			continue
		}

		seenDirectives[directive.Name] = true
		directives = append(directives, &Directive{
			Name:            directive.Name,
			Attributes:      attrs.attributes,
			Inputs:          categorized.bound,
			Outputs:         attrs.boundEvents,
			References:      attrs.references,
			SourceSpan:      parse_util.ParseSourceSpan{},
			StartSourceSpan: *directive.StartSourceSpan,
			EndSourceSpan:   directive.EndSourceSpan,
		})
	}

	return directives
}

func (v *HtmlAstToIvyAst) filterAnimationAttributes(attributes []*TextAttribute) []*TextAttribute {
	var filtered []*TextAttribute
	for _, a := range attributes {
		if !strings.HasPrefix(a.Name, "animate.") {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func (v *HtmlAstToIvyAst) filterAnimationInputs(attributes []*BoundAttribute) []*BoundAttribute {
	var filtered []*BoundAttribute
	for _, a := range attributes {
		if int(a.Type) != 1 {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func (v *HtmlAstToIvyAst) wrapInTemplate(
	node Node,
	templateProperties []*template_parser.ParsedProperty,
	templateVariables []*Variable,
	i18nAttrsMeta map[string]any,
	isTemplateElement bool,
	isI18nRootElement bool,
) Node {
	ngTemplate := "ng-template"
	attrs := v.categorizePropertyAttributes(&ngTemplate, templateProperties, i18nAttrsMeta)
	var templateAttrs []Node
	for _, attr := range attrs.literal {
		templateAttrs = append(templateAttrs, attr)
	}
	for _, attr := range attrs.bound {
		templateAttrs = append(templateAttrs, attr)
	}

	hoistedAttrs := struct {
		attributes []*TextAttribute
		inputs     []*BoundAttribute
		outputs    []*BoundEvent
	}{}

	if el, ok := node.(*Element); ok {
		hoistedAttrs.attributes = append(hoistedAttrs.attributes, v.filterAnimationAttributes(el.Attributes)...)
		hoistedAttrs.inputs = append(hoistedAttrs.inputs, v.filterAnimationInputs(el.Inputs)...)
		hoistedAttrs.outputs = append(hoistedAttrs.outputs, el.Outputs...)
	} else if comp, ok := node.(*Component); ok {
		hoistedAttrs.attributes = append(hoistedAttrs.attributes, v.filterAnimationAttributes(comp.Attributes)...)
		hoistedAttrs.inputs = append(hoistedAttrs.inputs, v.filterAnimationInputs(comp.Inputs)...)
		hoistedAttrs.outputs = append(hoistedAttrs.outputs, comp.Outputs...)
	}

	var i18nMeta any
	if !(isTemplateElement && isI18nRootElement) {
		if el, ok := node.(*Element); ok {
			i18nMeta = el.I18n
		} else if comp, ok := node.(*Component); ok {
			i18nMeta = comp.I18n
		} else if tpl, ok := node.(*Template); ok {
			i18nMeta = tpl.I18n
		} else if c, ok := node.(*Content); ok {
			i18nMeta = c.I18n
		}
	}

	var name *string
	if comp, ok := node.(*Component); ok {
		name = comp.TagName
	} else if _, ok := node.(*Template); ok {
		name = nil
	} else if el, ok := node.(*Element); ok {
		name = &el.Name
	} else if c, ok := node.(*Content); ok {
		name = &c.Name
	}

	var sourceSpan, startSourceSpan parse_util.ParseSourceSpan
	var endSourceSpan *parse_util.ParseSourceSpan
	if el, ok := node.(*Element); ok {
		sourceSpan = el.SourceSpan
		startSourceSpan = el.StartSourceSpan
		endSourceSpan = el.EndSourceSpan
	} else if comp, ok := node.(*Component); ok {
		sourceSpan = comp.SourceSpan
		startSourceSpan = comp.StartSourceSpan
		endSourceSpan = comp.EndSourceSpan
	} else if tpl, ok := node.(*Template); ok {
		sourceSpan = tpl.SourceSpan
		startSourceSpan = tpl.StartSourceSpan
		endSourceSpan = tpl.EndSourceSpan
	} else if c, ok := node.(*Content); ok {
		sourceSpan = c.SourceSpan
		startSourceSpan = c.StartSourceSpan
		endSourceSpan = c.EndSourceSpan
	}

	return &Template{
		TagName:         name,
		Attributes:      hoistedAttrs.attributes,
		Inputs:          hoistedAttrs.inputs,
		Outputs:         hoistedAttrs.outputs,
		Directives:      []*Directive{},
		TemplateAttrs:   templateAttrs,
		Children:        []Node{node},
		References:      []*Reference{},
		Variables:       templateVariables,
		IsSelfClosing:   false,
		SourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		I18n:            i18nMeta,
	}
}

func (v *HtmlAstToIvyAst) _visitTextWithInterpolation(
	value string,
	sourceSpan parse_util.ParseSourceSpan,
	interpolatedTokens []ml_parser.Token,
	i18nMeta any,
) Node {
	valueNoNgsp := ml_parser.ReplaceNgsp(value)
	var tokens []interface{}
	if interpolatedTokens != nil {
		tokens = make([]interface{}, len(interpolatedTokens))
		for i, tok := range interpolatedTokens {
			tokens[i] = tok
		}
	}
	expr := v.bindingParser.ParseInterpolation(value, toExpressionParserParseSourceSpan(sourceSpan), tokens)
	if expr.Ast != nil {
		return &BoundText{Value: &expr, SourceSpan: sourceSpan, I18n: i18nMeta}
	}
	return &Text{Value: valueNoNgsp, SourceSpan: sourceSpan}
}

func (v *HtmlAstToIvyAst) parseVariable(
	identifier string,
	value string,
	sourceSpan parse_util.ParseSourceSpan,
	keySpan parse_util.ParseSourceSpan,
	valueSpan *parse_util.ParseSourceSpan,
	variables *[]*Variable,
) {
	if strings.Contains(identifier, "-") {
		v.reportError(`"-" is not allowed in variable names`, sourceSpan)
	} else if len(identifier) == 0 {
		v.reportError(`Variable does not have a name`, sourceSpan)
	}

	*variables = append(*variables, &Variable{
		Name:       identifier,
		Value:      value,
		SourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
	})
}

func (v *HtmlAstToIvyAst) parseReference(
	identifier string,
	value string,
	sourceSpan parse_util.ParseSourceSpan,
	keySpan parse_util.ParseSourceSpan,
	valueSpan *parse_util.ParseSourceSpan,
	references *[]*Reference,
) {
	if strings.Contains(identifier, "-") {
		v.reportError(`"-" is not allowed in reference names`, sourceSpan)
	} else if len(identifier) == 0 {
		v.reportError(`Reference does not have a name`, sourceSpan)
	} else {
		for _, ref := range *references {
			if ref.Name == identifier {
				v.reportError(`Reference "#`+identifier+`" is defined more than once`, sourceSpan)
				break
			}
		}
	}

	*references = append(*references, &Reference{
		Name:       identifier,
		Value:      value,
		SourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
	})
}

func (v *HtmlAstToIvyAst) parseAssignmentEvent(
	name string,
	expression string,
	sourceSpan parse_util.ParseSourceSpan,
	valueSpan *parse_util.ParseSourceSpan,
	targetMatchableAttrs [][]string,
	boundEvents *[]*BoundEvent,
	keySpan parse_util.ParseSourceSpan,
	absoluteOffset int,
) {
	var parsedEvents []template_parser.ParsedEvent
	actualValueSpan := sourceSpan
	if valueSpan != nil {
		actualValueSpan = *valueSpan
	}
	v.bindingParser.ParseEvent(
		name+"Change",
		expression,
		true,
		toExpressionParserParseSourceSpan(sourceSpan),
		toExpressionParserParseSourceSpan(actualValueSpan),
		&targetMatchableAttrs,
		&parsedEvents,
		toExpressionParserParseSourceSpan(keySpan),
	)
	var events []*template_parser.ParsedEvent
	for i := range parsedEvents {
		events = append(events, &parsedEvents[i])
	}
	v.addEvents(events, boundEvents)
}

func (v *HtmlAstToIvyAst) validateSelectorlessReferences(references []*Reference) {
	if len(references) == 0 {
		return
	}

	seenNames := make(map[string]bool)

	for _, ref := range references {
		if len(ref.Value) > 0 {
			span := ref.SourceSpan
			if ref.ValueSpan != nil {
				span = *ref.ValueSpan
			}
			v.reportError("Cannot specify a value for a local reference in this context", span)
		} else if seenNames[ref.Name] {
			v.reportError("Duplicate reference names are not allowed", ref.SourceSpan)
		} else {
			seenNames[ref.Name] = true
		}
	}
}

func (v *HtmlAstToIvyAst) reportError(message string, sourceSpan parse_util.ParseSourceSpan) {
	v.errors = append(v.errors, parse_util.NewParseError(&sourceSpan, message, nil, nil))
}

type NonBindableVisitor struct{}

func (v *NonBindableVisitor) VisitElement(ast *ml_parser.Element, context any) any {
	preparsedElement := template_parser.PreparseElement(ast)
	if preparsedElement.Type == template_parser.PreparsedElementTypeScript ||
		preparsedElement.Type == template_parser.PreparsedElementTypeStyle ||
		preparsedElement.Type == template_parser.PreparsedElementTypeStylesheet {
		return nil
	}

	var children []Node
	for _, child := range ml_parser.VisitAll(v, ast.Children, nil) {
		if child != nil {
			children = append(children, child.(Node))
		}
	}

	var attrs []*TextAttribute
	for _, attr := range ast.Attrs {
		attrs = append(attrs, v.VisitAttribute(attr, nil).(*TextAttribute))
	}

	return &Element{
		Name:            ast.Name,
		Attributes:      attrs,
		Inputs:          []*BoundAttribute{},
		Outputs:         []*BoundEvent{},
		Directives:      []*Directive{},
		Children:        children,
		References:      []*Reference{},
		IsSelfClosing:   ast.IsSelfClosing,
		SourceSpan:      *ast.SourceSpan,
		StartSourceSpan: *ast.StartSourceSpan,
		EndSourceSpan:   ast.EndSourceSpan,
		IsVoid:          ast.IsVoid,
	}
}

func (v *NonBindableVisitor) VisitComment(comment *ml_parser.Comment, context any) any {
	return nil
}

func (v *NonBindableVisitor) VisitAttribute(attribute *ml_parser.Attribute, context any) any {
	return &TextAttribute{
		Name:       attribute.Name,
		Value:      attribute.Value,
		SourceSpan: *attribute.SourceSpan,
		KeySpan:    attribute.KeySpan,
		ValueSpan:  attribute.ValueSpan,
		I18n:       attribute.I18n,
	}
}

func (v *NonBindableVisitor) VisitText(text *ml_parser.Text, context any) any {
	return &Text{Value: text.Value, SourceSpan: *text.SourceSpan}
}

func (v *NonBindableVisitor) VisitExpansion(expansion *ml_parser.Expansion, context any) any {
	return nil
}

func (v *NonBindableVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	return nil
}

func (v *NonBindableVisitor) VisitBlock(block *ml_parser.Block, context any) any {
	nodes := []Node{
		&Text{Value: "", SourceSpan: *block.StartSourceSpan},
	}
	for _, child := range ml_parser.VisitAll(v, block.Children, nil) {
		if child != nil {
			nodes = append(nodes, child.(Node))
		}
	}
	if block.EndSourceSpan != nil {
		nodes = append(nodes, &Text{Value: "", SourceSpan: *block.EndSourceSpan})
	}
	return nodes
}

func (v *NonBindableVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}

func (v *NonBindableVisitor) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	return &Text{Value: "@let " + decl.Name + " = " + decl.Value + ";", SourceSpan: decl.SourceSpan}
}

func (v *NonBindableVisitor) VisitComponent(ast *ml_parser.Component, context any) any {
	var children []Node
	for _, child := range ml_parser.VisitAll(v, ast.Children, nil) {
		if child != nil {
			children = append(children, child.(Node))
		}
	}

	var attrs []*TextAttribute
	for _, attr := range ast.Attrs {
		attrs = append(attrs, v.VisitAttribute(attr, nil).(*TextAttribute))
	}

	return &Element{
		Name:            ast.FullName,
		Attributes:      attrs,
		Inputs:          []*BoundAttribute{},
		Outputs:         []*BoundEvent{},
		Directives:      []*Directive{},
		Children:        children,
		References:      []*Reference{},
		IsSelfClosing:   ast.IsSelfClosing,
		SourceSpan:      *ast.SourceSpan,
		StartSourceSpan: *ast.StartSourceSpan,
		EndSourceSpan:   ast.EndSourceSpan,
		IsVoid:          false,
	}
}

func (v *NonBindableVisitor) VisitDirective(directive *ml_parser.Directive, context any) any {
	return nil
}

var NON_BINDABLE_VISITOR = &NonBindableVisitor{}

func (v *HtmlAstToIvyAst) addEvents(events []*template_parser.ParsedEvent, boundEvents *[]*BoundEvent) {
	for _, e := range events {
		*boundEvents = append(*boundEvents, BoundEventFromParsedEvent(e, v.baseSpan))
	}
}

func textContents(node *ml_parser.Element) *string {
	if len(node.Children) != 1 {
		return nil
	}
	if textNode, ok := node.Children[0].(*ml_parser.Text); ok {
		return &textNode.Value
	}
	return nil
}

// These are mocked control flow blocks functions based on TS imports.
// In practice, they should map to actual implementation in other files.
func createDeferredBlock(block *ml_parser.Block, siblings []*ml_parser.Block, visitor *HtmlAstToIvyAst, parser *template_parser.BindingParser) (Node, []*parse_util.ParseError) {
	node, errs := CreateDeferredBlock(block, siblings, visitor, parser)
	var parseErrors []*parse_util.ParseError
	for i := range errs {
		parseErrors = append(parseErrors, &errs[i])
	}
	return node, parseErrors
}

func createSwitchBlock(block *ml_parser.Block, visitor *HtmlAstToIvyAst, parser *template_parser.BindingParser) (Node, []*parse_util.ParseError) {
	node, errs := CreateSwitchBlock(block, visitor, parser)
	var parseErrors []*parse_util.ParseError
	for i := range errs {
		parseErrors = append(parseErrors, &errs[i])
	}
	return node, parseErrors
}

func createForLoop(block *ml_parser.Block, siblings []*ml_parser.Block, visitor *HtmlAstToIvyAst, parser *template_parser.BindingParser) (Node, []*parse_util.ParseError) {
	node, errs := CreateForLoop(block, siblings, visitor, parser)
	var parseErrors []*parse_util.ParseError
	for i := range errs {
		parseErrors = append(parseErrors, &errs[i])
	}
	return node, parseErrors
}

func createIfBlock(block *ml_parser.Block, siblings []*ml_parser.Block, visitor *HtmlAstToIvyAst, parser *template_parser.BindingParser) (Node, []*parse_util.ParseError) {
	node, errs := CreateIfBlock(block, siblings, visitor, parser)
	var parseErrors []*parse_util.ParseError
	for i := range errs {
		parseErrors = append(parseErrors, &errs[i])
	}
	return node, parseErrors
}

func isConnectedDeferLoopBlock(name string) bool { return IsConnectedDeferLoopBlock(name) }
func isConnectedForLoopBlock(name string) bool   { return IsConnectedForLoopBlock(name) }
func isConnectedIfLoopBlock(name string) bool    { return IsConnectedIfLoopBlock(name) }

func toExpressionParserParseSourceSpan(span parse_util.ParseSourceSpan) expression_parser.ParseSourceSpan {
	fullStart := span.Start.Offset
	if span.FullStart != nil {
		fullStart = span.FullStart.Offset
	}
	return expression_parser.ParseSourceSpan{
		Start:     span.Start.Offset,
		End:       span.End.Offset,
		FullStart: fullStart,
	}
}

func toExpressionParserParseSourceSpanPtr(span *parse_util.ParseSourceSpan) *expression_parser.ParseSourceSpan {
	if span == nil {
		return nil
	}
	fullStart := span.Start.Offset
	if span.FullStart != nil {
		fullStart = span.FullStart.Offset
	}
	return &expression_parser.ParseSourceSpan{
		Start:     span.Start.Offset,
		End:       span.End.Offset,
		FullStart: fullStart,
	}
}

// toParseUtilParseSourceSpan converts an expression_parser.ParseSourceSpan (integers only)
// to a parse_util.ParseSourceSpan, using the provided reference span's source file for location info.
func toParseUtilParseSourceSpan(exprSpan expression_parser.ParseSourceSpan) parse_util.ParseSourceSpan {
	startLoc := &parse_util.ParseLocation{Offset: exprSpan.Start}
	endLoc := &parse_util.ParseLocation{Offset: exprSpan.End}
	return parse_util.ParseSourceSpan{
		Start:     startLoc,
		End:       endLoc,
		FullStart: startLoc,
	}
}
