package ingest

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	i18n "github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	r3i18n "github.com/microsoft/typescript-go/angular-packages/compiler/render3/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/schema"
	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

var domSchema = schema.NewDomElementSchemaRegistry()

const ngTemplateTagName = "ng-template"
const animatePrefix = "animate."

func init() {
	render3.IngestComponent = IngestComponent
	render3.IngestHostBinding = IngestHostBinding
}

func IsI18nRootNode(meta i18n.I18nMeta) bool {
	if meta == nil {
		return false
	}
	_, ok := meta.(*i18n.Message)
	return ok
}

func IsSingleI18nIcu(meta i18n.I18nMeta) bool {
	if meta == nil {
		return false
	}
	msg, ok := meta.(*i18n.Message)
	if !ok {
		return false
	}
	if len(msg.Nodes) != 1 {
		return false
	}
	_, okIcu := msg.Nodes[0].(*i18n.Icu)
	_, okIcuPh := msg.Nodes[0].(*i18n.IcuPlaceholder)
	return okIcu || okIcuPh
}

func IngestComponent(
	componentName string,
	template []render3.Node,
	constantPool render3.ConstantPool,
	compilationMode compilation.TemplateCompilationMode,
	relativeContextFilePath string,
	i18nUseExternalIds bool,
	deferMeta any,
	allDeferrableDepsFn *output.ReadVarExpr,
	relativeTemplatePath *string,
	enableDebugLocations bool,
	legacyOptionalChaining bool,
	foreignImports []any,
) *compilation.ComponentCompilationJob {
	job := compilation.NewComponentCompilationJob(
		componentName,
		constantPool,
		compilationMode,
		relativeContextFilePath,
		i18nUseExternalIds,
		deferMeta,
		allDeferrableDepsFn,
		relativeTemplatePath,
		enableDebugLocations,
		legacyOptionalChaining,
		foreignImports,
	)
	ingestNodes(job.Root, template)
	return job
}

func IngestHostBinding(
	input *render3.HostBindingInput,
	bindingParserAny render3.BindingParser,
	constantPool render3.ConstantPool,
) *compilation.HostBindingCompilationJob {
	bindingParser := bindingParserAny.(*template_parser.BindingParser)
	job := compilation.NewHostBindingCompilationJob(
		input.ComponentName,
		constantPool,
		compilation.TemplateCompilationMode_DomOnly,
		input.LegacyOptionalChaining,
	)
	for _, property := range input.Properties {
		bindingKind := ir.BindingKindProperty
		name := property.Name
		if strings.HasPrefix(name, "attr.") {
			name = name[len("attr."):]
			bindingKind = ir.BindingKindAttribute
		} else if strings.HasPrefix(name, "class.") {
			name = name[len("class."):]
			bindingKind = ir.BindingKindClassName
		} else if strings.HasPrefix(name, "style.") {
			name = name[len("style."):]
			bindingKind = ir.BindingKindStyleProperty
		}
		if property.IsLegacyAnimation() {
			bindingKind = ir.BindingKindLegacyAnimation
		}
		if property.IsAnimation() {
			bindingKind = ir.BindingKindAnimation
		}
		contexts := bindingParser.CalcPossibleSecurityContexts(
			input.ComponentSelector,
			name,
			bindingKind == ir.BindingKindAttribute,
		)
		var securityContexts []core.SecurityContext
		for _, ctx := range contexts {
			if ctx != core.SecurityContextNone {
				securityContexts = append(securityContexts, ctx)
			}
		}
		ingestDomProperty(job, &property, name, bindingKind, securityContexts)
	}
	for name, expr := range input.Attributes {
		contexts := bindingParser.CalcPossibleSecurityContexts(
			input.ComponentSelector,
			name,
			true,
		)
		var securityContexts []core.SecurityContext
		for _, ctx := range contexts {
			if ctx != core.SecurityContextNone {
				securityContexts = append(securityContexts, ctx)
			}
		}
		ingestHostAttribute(job, name, expr, securityContexts)
	}
	for _, event := range input.Events {
		ingestHostEvent(job, &event)
	}
	return job
}

func ingestDomProperty(
	job *compilation.HostBindingCompilationJob,
	property *template_parser.ParsedProperty,
	name string,
	bindingKind ir.BindingKind,
	securityContexts []core.SecurityContext,
) {
	var expression any
	ast := astOf(&property.Expression)
	if interp, ok := ast.(*expression_parser.Interpolation); ok {
		exprs := make([]output.Expression, len(interp.Expressions))
		for i, expr := range interp.Expressions {
			exprs[i] = convertAst(expr, job, convertParseSourceSpan(property.SourceSpan))
		}
		expression = ir.NewInterpolation(toStringSlice(interp.Strings), exprs, nil)
	} else {
		expression = convertAst(ast, job, convertParseSourceSpan(property.SourceSpan))
	}

	job.Root.Update.Push(
		ir.CreateBindingOp(
			job.Root.Xref,
			bindingKind,
			name,
			expression.(output.Expression),
			nil,
			securityContexts,
			false,
			false,
			nil,
			nil,
			convertParseSourceSpan(property.SourceSpan),
		),
	)
}

func ingestHostAttribute(
	job *compilation.HostBindingCompilationJob,
	name string,
	value output.Expression,
	securityContexts []core.SecurityContext,
) {
	var srcSpan *parse_util.ParseSourceSpan
	if valExpr, ok := value.(interface{ GetSourceSpan() interface{} }); ok {
		if spanVal, ok := valExpr.GetSourceSpan().(*parse_util.ParseSourceSpan); ok {
			srcSpan = spanVal
		} else if spanVal2, ok := valExpr.GetSourceSpan().(parse_util.ParseSourceSpan); ok {
			srcSpan = &spanVal2
		}
	}
	attrBinding := ir.CreateBindingOp(
		job.Root.Xref,
		ir.BindingKindAttribute,
		name,
		value,
		nil,
		securityContexts,
		true,
		false,
		nil,
		nil,
		srcSpan,
	)
	job.Root.Update.Push(attrBinding)
}

func ingestHostEvent(job *compilation.HostBindingCompilationJob, event *template_parser.ParsedEvent) {
	var eventBinding ir.Op
	if event.Type == template_parser.ParsedEventTypeAnimation {
		eventBinding = ir.CreateAnimationListenerOp(
			job.Root.Xref,
			ir.NewSlotHandle(),
			event.Name,
			nil,
			makeListenerHandlerOps(job.Root, &event.Handler, convertParseSourceSpan(event.HandlerSpan)),
			func() ir.AnimationKind {
				if strings.HasSuffix(event.Name, "enter") {
					return ir.AnimationKindENTER
				}
				return ir.AnimationKindLEAVE
			}(),
			event.TargetOrPhase,
			true,
			convertParseSourceSpan(event.SourceSpan),
		)
	} else {
		var phase *string
		var target *string
		if event.Type != template_parser.ParsedEventTypeLegacyAnimation {
			target = event.TargetOrPhase
		} else {
			phase = event.TargetOrPhase
		}

		eventBinding = ir.CreateListenerOp(
			job.Root.Xref,
			ir.NewSlotHandle(),
			event.Name,
			nil,
			makeListenerHandlerOps(job.Root, &event.Handler, convertParseSourceSpan(event.HandlerSpan)),
			phase,
			target,
			true,
			convertParseSourceSpan(event.SourceSpan),
		)
	}
	job.Root.Create.Push(eventBinding)
}

func ingestNodes(unit *compilation.ViewCompilationUnit, template []render3.Node) {
	for _, node := range template {
		switch n := node.(type) {
		case *render3.Element:
			ingestElement(unit, n)
		case *render3.Template:
			ingestTemplate(unit, n)
		case *render3.Content:
			ingestContent(unit, n)
		case *render3.Text:
			ingestText(unit, n, nil)
		case *render3.BoundText:
			ingestBoundText(unit, n, nil)
		case *render3.IfBlock:
			ingestIfBlock(unit, n)
		case *render3.SwitchBlock:
			ingestSwitchBlock(unit, n)
		case *render3.DeferredBlock:
			ingestDeferBlock(unit, n)
		case *render3.Icu:
			ingestIcu(unit, n)
		case *render3.ForLoopBlock:
			ingestForBlock(unit, n)
		case *render3.LetDeclaration:
			ingestLetDeclaration(unit, n)
		case *render3.Component:
			// TODO(crisbeto): account for selectorless nodes.
		default:
			panic(fmt.Sprintf("Unsupported template node: %T", node))
		}
	}
}

func ingestElement(unit *compilation.ViewCompilationUnit, element *render3.Element) {
	if element.I18n != nil {
		if _, ok := element.I18n.(*i18n.Message); !ok {
			if _, ok2 := element.I18n.(*i18n.TagPlaceholder); !ok2 {
				panic(fmt.Sprintf("Unhandled i18n metadata type for element: %T", element.I18n))
			}
		}
	}

	id := unit.Job.AllocateXrefId()

	foreignComp := unit.Job.GetForeignComponent(element)
	if foreignComp != nil {
		// Stub/implemented in compilation_job.go
		return
	}

	ns, elementName, _ := tags.SplitNsName(element.Name, false)

	startOp := ir.CreateElementStartOp(
		elementName,
		id,
		namespaceForKey(ns),
		element.I18n,
		&element.StartSourceSpan,
		&element.SourceSpan,
	)
	if _, ok := element.I18n.(*i18n.TagPlaceholder); !ok {
		startOp.I18nPlaceholder = nil
	}
	unit.Create.Push(startOp)

	ingestElementBindings(unit, startOp, element)
	ingestReferences(startOp, element.References)

	var i18nBlockId *ir.XrefId
	if msg, ok := element.I18n.(*i18n.Message); ok && msg != nil {
		bid := unit.Job.AllocateXrefId()
		i18nBlockId = &bid
		unit.Create.Push(ir.CreateI18nStartOp(*i18nBlockId, element.I18n, nil, &element.StartSourceSpan))
	}

	ingestNodes(unit, element.Children)

	var endSourceSpan *parse_util.ParseSourceSpan = element.EndSourceSpan
	if endSourceSpan == nil {
		endSourceSpan = &element.StartSourceSpan
	}

	endOp := ir.CreateElementEndOp(id, endSourceSpan)
	unit.Create.Push(endOp)

	if i18nBlockId != nil {
		ir.OpListInsertBefore(unit.Create, ir.CreateI18nEndOp(*i18nBlockId, endSourceSpan), endOp)
	}
}

func ingestTemplate(unit *compilation.ViewCompilationUnit, tmpl *render3.Template) {
	if tmpl.I18n != nil {
		if _, ok := tmpl.I18n.(*i18n.Message); !ok {
			if _, ok2 := tmpl.I18n.(*i18n.TagPlaceholder); !ok2 {
				panic(fmt.Sprintf("Unhandled i18n metadata type for template: %T", tmpl.I18n))
			}
		}
	}

	childView := unit.Job.AllocateView(unit.Xref)

	var tagNameWithoutNamespace *string
	var namespacePrefix *string
	if tmpl.TagName != nil {
		ns, name, _ := tags.SplitNsName(*tmpl.TagName, false)
		tagNameWithoutNamespace = &name
		namespacePrefix = ns
	}

	var i18nPlaceholder any
	if _, ok := tmpl.I18n.(*i18n.TagPlaceholder); ok {
		i18nPlaceholder = tmpl.I18n
	}

	namespace := namespaceForKey(namespacePrefix)
	var functionNameSuffix string
	if tagNameWithoutNamespace != nil {
		functionNameSuffix = prefixWithNamespace(*tagNameWithoutNamespace, namespace)
	}

	templateKind := ir.TemplateKindStructural
	if isPlainTemplate(tmpl) {
		templateKind = ir.TemplateKindNgTemplate
	}

	templateOp := ir.CreateTemplateOp(
		childView.Xref,
		templateKind,
		tagNameWithoutNamespace,
		functionNameSuffix,
		namespace,
		i18nPlaceholder,
		&tmpl.StartSourceSpan,
		&tmpl.SourceSpan,
	)
	unit.Create.Push(templateOp)

	ingestTemplateBindings(unit, templateOp, tmpl, templateKind)
	ingestReferences(templateOp, tmpl.References)
	ingestNodes(childView, tmpl.Children)

	for _, variable := range tmpl.Variables {
		val := variable.Value
		if val == "" {
			val = "$implicit"
		}
		childView.SetContextVariable(variable.Name, val)
	}

	if templateKind == ir.TemplateKindNgTemplate {
		if _, ok := tmpl.I18n.(*i18n.Message); ok {
			id := unit.Job.AllocateXrefId()
			childView.Create.Prepend([]ir.Op{ir.CreateI18nStartOp(id, tmpl.I18n, nil, &tmpl.StartSourceSpan)})
			var endSourceSpan *parse_util.ParseSourceSpan = tmpl.EndSourceSpan
			if endSourceSpan == nil {
				endSourceSpan = &tmpl.StartSourceSpan
			}
			childView.Create.Push(ir.CreateI18nEndOp(id, endSourceSpan))
		}
	}
}

func ingestContent(unit *compilation.ViewCompilationUnit, content *render3.Content) {
	if content.I18n != nil {
		if _, ok := content.I18n.(*i18n.TagPlaceholder); !ok {
			panic(fmt.Sprintf("Unhandled i18n metadata type for content: %T", content.I18n))
		}
	}

	var fallbackView *compilation.ViewCompilationUnit
	hasRealContent := false
	for _, child := range content.Children {
		if _, isComment := child.(*render3.Comment); isComment {
			continue
		}
		if text, isText := child.(*render3.Text); isText {
			if len(strings.TrimSpace(text.Value)) > 0 {
				hasRealContent = true
				break
			}
		} else {
			hasRealContent = true
			break
		}
	}

	if hasRealContent {
		fallbackView = unit.Job.AllocateView(unit.Xref)
		ingestNodes(fallbackView, content.Children)
	}

	id := unit.Job.AllocateXrefId()
	var fallbackViewXref *ir.XrefId
	if fallbackView != nil {
		fallbackViewXref = &fallbackView.Xref
	}

	var selector *string
	if content.Selector != "" {
		selector = &content.Selector
	}

	op := ir.CreateProjectionOp(
		id,
		selector,
		content.I18n,
		fallbackViewXref,
		&content.SourceSpan,
	)

	for _, attr := range content.Attributes {
		securityContext := domSchema.SecurityContext(content.Name, attr.Name, true)
		unit.Update.Push(
			ir.CreateBindingOp(
				op.Xref,
				ir.BindingKindAttribute,
				attr.Name,
				output.NewLiteralExpr(attr.Value, nil, nil, nil),
				nil,
				securityContext,
				true,
				false,
				nil,
				asMessage(attr.I18n),
				&attr.SourceSpan,
			),
		)
	}
	unit.Create.Push(op)
}

func ingestText(unit *compilation.ViewCompilationUnit, text *render3.Text, icuPlaceholder *string) {
	unit.Create.Push(ir.CreateTextOp(unit.Job.AllocateXrefId(), text.Value, icuPlaceholder, &text.SourceSpan))
}

func ingestBoundText(unit *compilation.ViewCompilationUnit, text *render3.BoundText, icuPlaceholder *string) {
	value := text.Value
	if astWithSrc, ok := value.(*expression_parser.ASTWithSource); ok {
		value = astWithSrc.Ast
	}
	interp, ok := value.(*expression_parser.Interpolation)
	if !ok {
		panic(fmt.Sprintf("AssertionError: expected Interpolation for BoundText node, got %T", value))
	}
	if text.I18n != nil {
		if _, ok := text.I18n.(*i18n.Container); !ok {
			panic(fmt.Sprintf("Unhandled i18n metadata type for text interpolation: %T", text.I18n))
		}
	}

	var i18nPlaceholders []string
	if container, ok := text.I18n.(*i18n.Container); ok && container != nil {
		for _, child := range container.Children {
			if placeholder, ok := child.(*i18n.Placeholder); ok {
				i18nPlaceholders = append(i18nPlaceholders, placeholder.Name)
			}
		}
	}

	if len(i18nPlaceholders) > 0 && len(i18nPlaceholders) != len(interp.Expressions) {
		panic(fmt.Sprintf("Unexpected number of i18n placeholders (%d) for BoundText with %d expressions", len(i18nPlaceholders), len(interp.Expressions)))
	}

	textXref := unit.Job.AllocateXrefId()
	unit.Create.Push(ir.CreateTextOp(textXref, "", icuPlaceholder, &text.SourceSpan))

	convertedExprs := make([]output.Expression, len(interp.Expressions))
	for i, expr := range interp.Expressions {
		convertedExprs[i] = convertAst(expr, unit.Job, nil)
	}

	unit.Update.Push(ir.CreateInterpolateTextOp(
		textXref,
		ir.NewInterpolation(toStringSlice(interp.Strings), convertedExprs, i18nPlaceholders),
		&text.SourceSpan,
	))
}

func ingestIfBlock(unit *compilation.ViewCompilationUnit, ifBlock *render3.IfBlock) {
	var firstXref *ir.XrefId
	var conditions []*ir.ConditionalCaseExpr
	for i, ifCase := range ifBlock.Branches {
		cView := unit.Job.AllocateView(unit.Xref)
		tagName := ingestControlFlowInsertionPoint(unit, cView.Xref, ifCase)

		if ifCase.ExpressionAlias != nil {
			cView.SetContextVariable(ifCase.ExpressionAlias.Name, ir.CTX_REF)
		}

		var ifCaseI18nMeta any
		if ifCase.I18n != nil {
			if _, ok := ifCase.I18n.(*i18n.BlockPlaceholder); !ok {
				panic(fmt.Sprintf("Unhandled i18n metadata type for if block: %T", ifCase.I18n))
			}
			ifCaseI18nMeta = ifCase.I18n
		}

		var conditionalCreateOp ir.Op
		suffix := "Conditional"
		if i == 0 {
			conditionalCreateOp = ir.CreateConditionalCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagName,
				suffix,
				ir.NamespaceHTML,
				ifCaseI18nMeta,
				&ifCase.StartSourceSpan,
				&ifCase.SourceSpan,
			)
		} else {
			conditionalCreateOp = ir.CreateConditionalBranchCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagName,
				suffix,
				ir.NamespaceHTML,
				ifCaseI18nMeta,
				&ifCase.StartSourceSpan,
				&ifCase.SourceSpan,
			)
		}
		unit.Create.Push(conditionalCreateOp)

		if firstXref == nil {
			bid := cView.Xref
			firstXref = &bid
		}

		var caseExpr output.Expression
		if ifCase.Expression != nil {
			caseExpr = convertAst(ifCase.Expression, unit.Job, nil)
		}

		var cOpXref ir.XrefId
		var cOpHandle *ir.SlotHandle
		if cc, ok := conditionalCreateOp.(*ir.ConditionalCreateOp); ok {
			cOpXref = cc.Xref
			cOpHandle = cc.Handle()
		} else if cbc, ok := conditionalCreateOp.(*ir.ConditionalBranchCreateOp); ok {
			cOpXref = cbc.Xref
			cOpHandle = cbc.Handle()
		}

		var alias any
		if ifCase.ExpressionAlias != nil {
			alias = ifCase.ExpressionAlias
		}

		conditionalCaseExpr := ir.NewConditionalCaseExpr(
			caseExpr,
			cOpXref,
			cOpHandle,
			alias,
		)
		conditions = append(conditions, conditionalCaseExpr)
		ingestNodes(cView, ifCase.Children)
	}

	unit.Update.Push(ir.CreateConditionalOp(*firstXref, nil, conditions, &ifBlock.SourceSpan))
}

func ingestSwitchBlock(unit *compilation.ViewCompilationUnit, switchBlock *render3.SwitchBlock) {
	if len(switchBlock.Groups) == 0 {
		return
	}

	var firstXref *ir.XrefId
	var conditions []*ir.ConditionalCaseExpr
	for i, group := range switchBlock.Groups {
		cView := unit.Job.AllocateView(unit.Xref)
		tagName := ingestControlFlowInsertionPoint(unit, cView.Xref, group)

		var switchCaseI18nMeta any
		if group.I18n != nil {
			if _, ok := group.I18n.(*i18n.BlockPlaceholder); !ok {
				panic(fmt.Sprintf("Unhandled i18n metadata type for switch block: %T", group.I18n))
			}
			switchCaseI18nMeta = group.I18n
		}

		var conditionalCreateOp ir.Op
		suffix := "Case"
		if i == 0 {
			conditionalCreateOp = ir.CreateConditionalCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagName,
				suffix,
				ir.NamespaceHTML,
				switchCaseI18nMeta,
				&group.StartSourceSpan,
				&group.SourceSpan,
			)
		} else {
			conditionalCreateOp = ir.CreateConditionalBranchCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagName,
				suffix,
				ir.NamespaceHTML,
				switchCaseI18nMeta,
				&group.StartSourceSpan,
				&group.SourceSpan,
			)
		}
		unit.Create.Push(conditionalCreateOp)

		if firstXref == nil {
			bid := cView.Xref
			firstXref = &bid
		}

		var cOpXref ir.XrefId
		var cOpHandle *ir.SlotHandle
		if cc, ok := conditionalCreateOp.(*ir.ConditionalCreateOp); ok {
			cOpXref = cc.Xref
			cOpHandle = cc.Handle()
		} else if cbc, ok := conditionalCreateOp.(*ir.ConditionalBranchCreateOp); ok {
			cOpXref = cbc.Xref
			cOpHandle = cbc.Handle()
		}

		for _, switchCase := range group.Cases {
			var caseExpr output.Expression
			if switchCase.Expression != nil {
				caseExpr = convertAst(switchCase.Expression, unit.Job, &switchBlock.StartSourceSpan)
			}
			conditionalCaseExpr := ir.NewConditionalCaseExpr(
				caseExpr,
				cOpXref,
				cOpHandle,
				nil,
			)
			conditions = append(conditions, conditionalCaseExpr)
		}
		ingestNodes(cView, group.Children)
	}

	unit.Update.Push(
		ir.CreateConditionalOp(
			*firstXref,
			convertAst(switchBlock.Expression, unit.Job, nil),
			conditions,
			&switchBlock.SourceSpan,
		),
	)
}

func ingestDeferView(
	unit *compilation.ViewCompilationUnit,
	suffix string,
	i18nMeta i18n.I18nMeta,
	children []render3.Node,
	sourceSpan *parse_util.ParseSourceSpan,
) *ir.TemplateOp {
	if i18nMeta != nil {
		if _, ok := i18nMeta.(*i18n.BlockPlaceholder); !ok {
			panic("Unhandled i18n metadata type for defer block")
		}
	}
	if children == nil {
		return nil
	}
	secondaryView := unit.Job.AllocateView(unit.Xref)
	ingestNodes(secondaryView, children)
	templateOp := ir.CreateTemplateOp(
		secondaryView.Xref,
		ir.TemplateKindBlock,
		nil,
		"Defer"+suffix,
		ir.NamespaceHTML,
		i18nMeta,
		sourceSpan,
		sourceSpan,
	)
	unit.Create.Push(templateOp)
	return templateOp
}

func ingestDeferBlock(unit *compilation.ViewCompilationUnit, deferBlock *render3.DeferredBlock) {
	var ownResolverFn output.Expression

	var dm render3.R3ComponentDeferMetadata
	if unit.Job.DeferMeta != nil {
		if m, ok := unit.Job.DeferMeta.(render3.R3ComponentDeferMetadata); ok {
			dm = m
		} else if mptr, ok := unit.Job.DeferMeta.(*render3.R3ComponentDeferMetadata); ok && mptr != nil {
			dm = *mptr
		}
	}

	if dm.Mode == render3.DeferBlockDepsEmitMode_PerBlock {
		if dm.Blocks != nil {
			var found bool
			for k, v := range dm.Blocks {
				// Comparing AST nodes
				if k == deferBlock {
					if expr, ok := v.(output.Expression); ok {
						ownResolverFn = expr
					}
					found = true
					break
				}
			}
			if !found {
				// panic("AssertionError: unable to find a dependency function for this deferred block")
			}
		} else {
			// panic("AssertionError: unable to find a dependency function for this deferred block")
		}
	}

	var mainNodes []render3.Node = deferBlock.Children
	main := ingestDeferView(unit, "", deferBlock.I18n, mainNodes, &deferBlock.SourceSpan)

	var loadingNodes []render3.Node
	var loadingSourceSpan *parse_util.ParseSourceSpan
	var loadingI18n i18n.I18nMeta
	if deferBlock.Loading != nil {
		loadingNodes = deferBlock.Loading.Children
		loadingSourceSpan = &deferBlock.Loading.SourceSpan
		loadingI18n = deferBlock.Loading.I18n
	}
	loading := ingestDeferView(unit, "Loading", loadingI18n, loadingNodes, loadingSourceSpan)

	var placeholderNodes []render3.Node
	var placeholderSourceSpan *parse_util.ParseSourceSpan
	var placeholderI18n i18n.I18nMeta
	if deferBlock.Placeholder != nil {
		placeholderNodes = deferBlock.Placeholder.Children
		placeholderSourceSpan = &deferBlock.Placeholder.SourceSpan
		placeholderI18n = deferBlock.Placeholder.I18n
	}
	placeholder := ingestDeferView(unit, "Placeholder", placeholderI18n, placeholderNodes, placeholderSourceSpan)

	var errorNodes []render3.Node
	var errorSourceSpan *parse_util.ParseSourceSpan
	var errorI18n i18n.I18nMeta
	if deferBlock.Error != nil {
		errorNodes = deferBlock.Error.Children
		errorSourceSpan = &deferBlock.Error.SourceSpan
		errorI18n = deferBlock.Error.I18n
	}
	errorView := ingestDeferView(unit, "Error", errorI18n, errorNodes, errorSourceSpan)

	deferXref := unit.Job.AllocateXrefId()
	deferOp := ir.CreateDeferOp(
		deferXref,
		main.Xref,
		main.Handle(),
		ownResolverFn,
		unit.Job.AllDeferrableDepsFn,
		&deferBlock.SourceSpan,
	)

	if placeholder != nil {
		deferOp.PlaceholderView = placeholder.Xref
		deferOp.PlaceholderSlot = placeholder.Handle()
	}
	if loading != nil {
		deferOp.LoadingSlot = loading.Handle()
	}
	if errorView != nil {
		deferOp.ErrorSlot = errorView.Handle()
	}

	if deferBlock.Placeholder != nil {
		deferOp.PlaceholderMinimumTime = float64PtrToIntPtr(deferBlock.Placeholder.MinimumTime)
	}
	if deferBlock.Loading != nil {
		deferOp.LoadingMinimumTime = float64PtrToIntPtr(deferBlock.Loading.MinimumTime)
		deferOp.LoadingAfterTime = float64PtrToIntPtr(deferBlock.Loading.AfterTime)
	}

	deferOp.Flags = calcDeferBlockFlags(deferBlock)
	unit.Create.Push(deferOp)

	var deferOnOps []ir.Op
	var deferWhenOps []ir.Op

	ingestDeferTriggers(
		ir.DeferOpModifierKindHYDRATE,
		&deferBlock.HydrateTriggers,
		&deferOnOps,
		&deferWhenOps,
		unit,
		deferXref,
	)

	ingestDeferTriggers(
		ir.DeferOpModifierKindNONE,
		&deferBlock.Triggers,
		&deferOnOps,
		&deferWhenOps,
		unit,
		deferXref,
	)

	ingestDeferTriggers(
		ir.DeferOpModifierKindPREFETCH,
		&deferBlock.PrefetchTriggers,
		&deferOnOps,
		&deferWhenOps,
		unit,
		deferXref,
	)

	hasConcreteTrigger := false
	for _, op := range deferOnOps {
		if dop, ok := op.(*ir.DeferOnOp); ok && dop.Modifier == ir.DeferOpModifierKindNONE {
			hasConcreteTrigger = true
			break
		}
	}
	for _, op := range deferWhenOps {
		if dwop, ok := op.(*ir.DeferWhenOp); ok && dwop.Modifier == ir.DeferOpModifierKindNONE {
			hasConcreteTrigger = true
			break
		}
	}

	if !hasConcreteTrigger {
		deferOnOps = append(deferOnOps, ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferIdleTrigger{
				DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindIdle},
			},
			ir.DeferOpModifierKindNONE,
			nil,
		))
	}

	for _, op := range deferOnOps {
		unit.Create.Push(op)
	}
	for _, op := range deferWhenOps {
		unit.Update.Push(op)
	}
}

func calcDeferBlockFlags(deferBlockDetails *render3.DeferredBlock) any {
	// Checks if there are hydrate triggers
	// In Go, deferBlockDetails.HydrateTriggers is render3.DeferredBlockTriggers.
	// Let's see if we have any fields in HydrateTriggers set.
	ht := deferBlockDetails.HydrateTriggers
	if ht.Idle != nil || ht.Immediate != nil || ht.Timer != nil || ht.Hover != nil || ht.Interaction != nil || ht.Viewport != nil || ht.Never != nil || ht.When != nil {
		return ir.TDeferDetailsFlagsHasHydrateTriggers
	}
	return nil
}

func ingestDeferTriggers(
	modifier any,
	triggers *render3.DeferredBlockTriggers,
	onOps *[]ir.Op,
	whenOps *[]ir.Op,
	unit *compilation.ViewCompilationUnit,
	deferXref ir.XrefId,
) {
	if triggers.Idle != nil {
		deferOnOp := ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferIdleTrigger{
				DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindIdle},
				Timeout:          triggers.Idle.Timeout,
			},
			modifier,
			getTriggerSpan(triggers.Idle),
		)
		*onOps = append(*onOps, deferOnOp)
	}
	if triggers.Immediate != nil {
		deferOnOp := ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferImmediateTrigger{
				DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindImmediate},
			},
			modifier,
			getTriggerSpan(triggers.Immediate),
		)
		*onOps = append(*onOps, deferOnOp)
	}
	if triggers.Timer != nil {
		deferOnOp := ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferTimerTrigger{
				DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindTimer},
				Delay:            triggers.Timer.Delay,
			},
			modifier,
			getTriggerSpan(triggers.Timer),
		)
		*onOps = append(*onOps, deferOnOp)
	}
	if triggers.Hover != nil {
		deferOnOp := ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferHoverTrigger{
				DeferTriggerWithTargetBase: ir.DeferTriggerWithTargetBase{
					DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindHover},
					TargetName:       triggers.Hover.Reference,
				},
			},
			modifier,
			getTriggerSpan(triggers.Hover),
		)
		*onOps = append(*onOps, deferOnOp)
	}
	if triggers.Interaction != nil {
		deferOnOp := ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferInteractionTrigger{
				DeferTriggerWithTargetBase: ir.DeferTriggerWithTargetBase{
					DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindInteraction},
					TargetName:       triggers.Interaction.Reference,
				},
			},
			modifier,
			getTriggerSpan(triggers.Interaction),
		)
		*onOps = append(*onOps, deferOnOp)
	}
	if triggers.Viewport != nil {
		deferOnOp := ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferViewportTrigger{
				DeferTriggerWithTargetBase: ir.DeferTriggerWithTargetBase{
					DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindViewport},
					TargetName:       triggers.Viewport.Reference,
				},
				Options: nil,
			},
			modifier,
			getTriggerSpan(triggers.Viewport),
		)
		*onOps = append(*onOps, deferOnOp)
	}
	if triggers.Never != nil {
		deferOnOp := ir.CreateDeferOnOp(
			deferXref,
			&ir.DeferNeverTrigger{
				DeferTriggerBase: ir.DeferTriggerBase{Kind: ir.DeferTriggerKindNever},
			},
			modifier,
			getTriggerSpan(triggers.Never),
		)
		*onOps = append(*onOps, deferOnOp)
	}
	if triggers.When != nil {
		if _, ok := triggers.When.Value.(*expression_parser.Interpolation); ok {
			panic("Unexpected interpolation in defer block when trigger")
		}
		deferOnOp := ir.CreateDeferWhenOp(
			deferXref,
			convertAst(triggers.When.Value, unit.Job, getTriggerSpan(triggers.When)),
			modifier,
			getTriggerSpan(triggers.When),
		)
		*whenOps = append(*whenOps, deferOnOp)
	}
}

func ingestIcu(unit *compilation.ViewCompilationUnit, icu *render3.Icu) {
	if IsSingleI18nIcu(icu.I18n) {
		msg := icu.I18n.(*i18n.Message)
		var icuPhName string
		if icuPh := r3i18n.IcuFromI18nMessage(msg); icuPh != nil {
			icuPhName = icuPh.Name
		}
		xref := unit.Job.AllocateXrefId()
		unit.Create.Push(ir.CreateIcuStartOp(xref, icu.I18n, icuPhName, &icu.SourceSpan))
		// Collect vars and placeholders
		for ph, textNode := range icu.Vars {
			ingestBoundText(unit, textNode, &ph)
		}
		for ph, textNode := range icu.Placeholders {
			if bt, ok := textNode.(*render3.BoundText); ok {
				ingestBoundText(unit, bt, &ph)
			} else if t, ok := textNode.(*render3.Text); ok {
				ingestText(unit, t, &ph)
			}
		}
		unit.Create.Push(ir.CreateIcuEndOp(xref))
	} else {
		panic(fmt.Sprintf("Unhandled i18n metadata type for ICU: %T", icu.I18n))
	}
}

func ingestForBlock(unit *compilation.ViewCompilationUnit, forBlock *render3.ForLoopBlock) {
	repeaterView := unit.Job.AllocateView(unit.Xref)

	indexName := fmt.Sprintf("ɵ$index_%d", repeaterView.Xref)
	countName := fmt.Sprintf("ɵ$count_%d", repeaterView.Xref)
	indexVarNames := make(map[string]bool)

	repeaterView.SetContextVariable(forBlock.Item.Name, forBlock.Item.Value)

	for _, variable := range forBlock.ContextVariables {
		if variable.Value == "$index" {
			indexVarNames[variable.Name] = true
		}
		if variable.Name == "$index" {
			repeaterView.SetContextVariable("$index", variable.Value)
			repeaterView.SetContextVariable(indexName, variable.Value)
		} else if variable.Name == "$count" {
			repeaterView.SetContextVariable("$count", variable.Value)
			repeaterView.SetContextVariable(countName, variable.Value)
		} else {
			repeaterView.Aliases = append(repeaterView.Aliases, &ir.AliasVariable{
				Kind:       ir.SemanticVariableKindAlias,
				Name:       nil,
				Identifier: variable.Name,
				Expression: getComputedForLoopVariableExpression(variable, indexName, countName),
			})
		}
	}

	var track output.Expression
	if forBlock.TrackBy == nil {
		track = output.NewReadVarExpr("$index", nil, nil, nil)
	} else {
		track = convertAst(
			forBlock.TrackBy,
			unit.Job,
			convertSourceSpan(forBlock.TrackBy.Span(), &forBlock.SourceSpan),
		)
	}

	ingestNodes(repeaterView, forBlock.Children)

	var emptyView *compilation.ViewCompilationUnit
	var emptyTagName *string
	if forBlock.Empty != nil {
		emptyView = unit.Job.AllocateView(unit.Xref)
		ingestNodes(emptyView, forBlock.Empty.Children)
		emptyTagName = ingestControlFlowInsertionPoint(unit, emptyView.Xref, forBlock.Empty)
	}

	var indexVarNamesSet = make(map[string]bool)
	for k := range indexVarNames {
		indexVarNamesSet[k] = true
	}

	var i18nPlaceholder any
	if forBlock.I18n != nil {
		if _, ok := forBlock.I18n.(*i18n.BlockPlaceholder); !ok {
			panic("AssertionError: Unhandled i18n metadata type or @for")
		}
		i18nPlaceholder = forBlock.I18n
	}
	var emptyI18nPlaceholder any
	if forBlock.Empty != nil && forBlock.Empty.I18n != nil {
		if _, ok := forBlock.Empty.I18n.(*i18n.BlockPlaceholder); !ok {
			panic("AssertionError: Unhandled i18n metadata type or @empty")
		}
		emptyI18nPlaceholder = forBlock.Empty.I18n
	}

	tagName := ingestControlFlowInsertionPoint(unit, repeaterView.Xref, forBlock)
	var emptyViewXref *ir.XrefId
	if emptyView != nil {
		emptyViewXref = &emptyView.Xref
	}

	repeaterCreate := ir.CreateRepeaterCreateOp(
		repeaterView.Xref,
		emptyViewXref,
		tagName,
		track,
		struct {
			Index    map[string]bool
			Implicit string
		}{
			Index:    indexVarNamesSet,
			Implicit: forBlock.Item.Name,
		},
		emptyTagName,
		i18nPlaceholder,
		emptyI18nPlaceholder,
		&forBlock.StartSourceSpan,
		&forBlock.SourceSpan,
	)
	unit.Create.Push(repeaterCreate)

	expression := convertAst(
		&forBlock.Expression,
		unit.Job,
		convertSourceSpan(forBlock.Expression.Span(), &forBlock.SourceSpan),
	)
	repeater := ir.CreateRepeaterOp(
		repeaterCreate.Xref,
		repeaterCreate.Handle(),
		expression,
		&forBlock.SourceSpan,
	)
	unit.Update.Push(repeater)
}

func getComputedForLoopVariableExpression(
	variable *render3.Variable,
	indexName string,
	countName string,
) output.Expression {
	switch variable.Value {
	case "$index":
		return ir.NewLexicalReadExpr(indexName)
	case "$count":
		return ir.NewLexicalReadExpr(countName)
	case "$first":
		return ir.NewLexicalReadExpr(indexName).Identical(output.NewLiteralExpr(0, nil, nil, nil), nil)
	case "$last":
		return ir.NewLexicalReadExpr(indexName).Identical(
			ir.NewLexicalReadExpr(countName).Minus(output.NewLiteralExpr(1, nil, nil, nil), nil),
			nil,
		)
	case "$even":
		return ir.NewLexicalReadExpr(indexName).Modulo(output.NewLiteralExpr(2, nil, nil, nil), nil).Identical(output.NewLiteralExpr(0, nil, nil, nil), nil)
	case "$odd":
		return ir.NewLexicalReadExpr(indexName).Modulo(output.NewLiteralExpr(2, nil, nil, nil), nil).NotIdentical(output.NewLiteralExpr(0, nil, nil, nil), nil)
	default:
		panic(fmt.Sprintf("AssertionError: unknown @for loop variable %s", variable.Value))
	}
}

func ingestLetDeclaration(unit *compilation.ViewCompilationUnit, node *render3.LetDeclaration) {
	target := unit.Job.AllocateXrefId()

	unit.Create.Push(ir.CreateDeclareLetOp(target, node.Name, &node.SourceSpan))
	unit.Update.Push(
		ir.CreateStoreLetOp(
			target,
			node.Name,
			convertAst(node.Value, unit.Job, &node.ValueSpan),
			&node.SourceSpan,
		),
	)
}

func convertAst(
	ast expression_parser.AST,
	job compilation.CompilationJob,
	baseSourceSpan *parse_util.ParseSourceSpan,
) output.Expression {
	if astWithSrc, ok := ast.(*expression_parser.ASTWithSource); ok {
		return convertAst(astWithSrc.Ast, job, baseSourceSpan)
	}

	switch a := ast.(type) {
	case *expression_parser.PropertyWrite:
		var receiverExpr output.Expression
		if _, ok := a.Receiver.(*expression_parser.ImplicitReceiver); ok {
			receiverExpr = ir.NewContextExpr(job.GetRoot().GetXref())
		} else {
			receiverExpr = convertAst(a.Receiver, job, baseSourceSpan)
		}
		return output.NewBinaryOperatorExpr(
			output.BinaryOperatorAssign,
			output.NewReadPropExpr(receiverExpr, a.Name, nil, nil, nil, false),
			convertAst(a.Value, job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
	case *expression_parser.KeyedWrite:
		return output.NewBinaryOperatorExpr(
			output.BinaryOperatorAssign,
			output.NewReadKeyExpr(
				convertAst(a.Receiver, job, baseSourceSpan),
				convertAst(a.Key, job, baseSourceSpan),
				nil, nil, nil, false,
			),
			convertAst(a.Value, job, baseSourceSpan),
			nil, nil, nil,
		)
	case *expression_parser.PropertyRead:
		if _, ok := a.Receiver.(*expression_parser.ImplicitReceiver); ok {
			return ir.NewLexicalReadExpr(a.Name)
		}
		return output.NewReadPropExpr(
			convertAst(a.Receiver, job, baseSourceSpan),
			a.Name,
			nil,
			nil,
			nil,
			false,
		)
	case *expression_parser.SafeCall:
		args := make([]output.Expression, len(a.Args))
		for i, arg := range a.Args {
			args[i] = convertAst(arg, job, baseSourceSpan)
		}
		return output.NewInvokeFunctionExpr(
			convertAst(a.Receiver, job, baseSourceSpan),
			args,
			nil,
			nil,
			false,
			nil,
			true,
		)
	case *expression_parser.Call:
		if _, ok := a.Receiver.(*expression_parser.ImplicitReceiver); ok {
			panic("Unexpected ImplicitReceiver")
		}
		args := make([]output.Expression, len(a.Args))
		for i, arg := range a.Args {
			args[i] = convertAst(arg, job, baseSourceSpan)
		}
		return output.NewInvokeFunctionExpr(
			convertAst(a.Receiver, job, baseSourceSpan),
			args,
			nil,
			nil,
			false,
			nil,
			false,
		)
	case *expression_parser.LiteralPrimitive:
		if value, ok := a.Value.(string); ok && value == "undefined" && isUndefinedKeywordLiteral(a) {
			return output.NewReadVarExpr("undefined", nil, nil, nil)
		}
		return output.NewLiteralExpr(a.Value, nil, nil, nil)
	case *expression_parser.Unary:
		var op output.UnaryOperator
		switch a.Operator {
		case "+":
			op = output.UnaryOperatorPlus
		case "-":
			op = output.UnaryOperatorMinus
		default:
			panic(fmt.Sprintf("AssertionError: unknown unary operator %s", a.Operator))
		}
		return output.NewUnaryOperatorExpr(
			op,
			convertAst(a.Expr, job, baseSourceSpan),
			nil,
			nil,
			false,
			nil,
		)
	case *expression_parser.Binary:
		op, ok := binaryOperators[a.Operation]
		if !ok {
			panic(fmt.Sprintf("AssertionError: unknown binary operator %s", a.Operation))
		}
		return output.NewBinaryOperatorExpr(
			op,
			convertAst(a.Left, job, baseSourceSpan),
			convertAst(a.Right, job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
	case *expression_parser.ThisReceiver:
		return ir.NewContextExpr(job.GetRoot().GetXref())
	case *expression_parser.KeyedRead:
		return output.NewReadKeyExpr(
			convertAst(a.Receiver, job, baseSourceSpan),
			convertAst(a.Key, job, baseSourceSpan),
			nil,
			nil,
			nil,
			false,
		)
	case *expression_parser.Chain:
		panic("AssertionError: Chain in unknown context")
	case *expression_parser.LiteralMap:
		entries := make([]output.LiteralMapEntry, len(a.Keys))
		for idx, key := range a.Keys {
			val := convertAst(a.Values[idx], job, baseSourceSpan)
			if _, ok := key.(*expression_parser.LiteralMapSpreadKey); ok {
				entries[idx] = output.NewLiteralMapSpreadAssignment(val)
			} else if propKey, ok := key.(*expression_parser.LiteralMapPropertyKey); ok {
				entries[idx] = output.NewLiteralMapPropertyAssignment(propKey.Key, val, propKey.Quoted)
			} else {
				panic(fmt.Sprintf("Unhandled LiteralMapKey type %T", key))
			}
		}
		return output.NewLiteralMapExpr(entries, nil, nil, nil)
	case *expression_parser.LiteralArray:
		exprs := make([]output.Expression, len(a.Expressions))
		for i, expr := range a.Expressions {
			exprs[i] = convertAst(expr, job, baseSourceSpan)
		}
		return output.NewLiteralArrayExpr(exprs, nil, nil, nil)
	case *expression_parser.Conditional:
		return output.NewConditionalExpr(
			convertAst(a.Condition, job, baseSourceSpan),
			convertAst(a.TrueExp, job, baseSourceSpan),
			convertAst(a.FalseExp, job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
	case *expression_parser.NonNullAssert:
		return convertAst(a.Expression, job, baseSourceSpan)
	case *expression_parser.BindingPipe:
		args := make([]output.Expression, len(a.Args)+1)
		args[0] = convertAst(a.Exp, job, baseSourceSpan)
		for i, arg := range a.Args {
			args[i+1] = convertAst(arg, job, baseSourceSpan)
		}
		return ir.NewPipeBindingExpr(job.AllocateXrefId(), ir.NewSlotHandle(), a.Name, args)
	case *expression_parser.SafeKeyedRead:
		return ir.NewSafeKeyedReadExpr(
			convertAst(a.Receiver, job, baseSourceSpan),
			convertAst(a.Key, job, baseSourceSpan),
			convertSourceSpan(a.Span(), baseSourceSpan),
		)
	case *expression_parser.SafePropertyRead:
		return ir.NewSafePropertyReadExpr(convertAst(a.Receiver, job, baseSourceSpan), a.Name)
	case *expression_parser.SafeMethodCall:
		args := make([]output.Expression, len(a.Args))
		for i, arg := range a.Args {
			args[i] = convertAst(arg, job, baseSourceSpan)
		}
		return output.NewInvokeFunctionExpr(
			convertAst(a.Receiver, job, baseSourceSpan),
			args,
			nil,
			nil,
			false,
			nil,
			true,
		)
	case *expression_parser.EmptyExpr:
		return ir.NewEmptyExpr(convertSourceSpan(a.Span(), baseSourceSpan))
	case *expression_parser.PrefixNot:
		return output.NewNotExpr(
			convertAst(a.Expression, job, baseSourceSpan),
			nil,
			nil,
		)
	case *expression_parser.TypeofExpression:
		return output.NewTypeofExpr(
			convertAst(a.Expr, job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
	case *expression_parser.VoidExpression:
		return output.NewVoidExpr(
			convertAst(a.Expr, job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
	case *expression_parser.TemplateLiteral:
		return convertTemplateLiteral(a, job, baseSourceSpan)
	case *expression_parser.TaggedTemplateLiteral:
		return output.NewTaggedTemplateLiteralExpr(
			convertAst(a.Tag, job, baseSourceSpan),
			convertTemplateLiteral(a.Template.(*expression_parser.TemplateLiteral), job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
	case *expression_parser.ParenthesizedExpression:
		return output.NewParenthesizedExpr(
			convertAst(a.Expression, job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
	case *expression_parser.RegularExpressionLiteral:
		return output.NewRegularExpressionLiteralExpr(a.Body, &a.Flags, nil, nil)
	case *expression_parser.SpreadElement:
		return output.NewSpreadElementExpr(convertAst(a.Expression, job, baseSourceSpan), nil, nil)
	case *expression_parser.ArrowFunction:
		params := make([]*output.FnParam, len(a.Params))
		for i, param := range a.Params {
			p := param.(*expression_parser.ArrowFunctionIdentifierParameter)
			params[i] = &output.FnParam{Name: p.Name, Type: output.DYNAMIC_TYPE}
		}
		arrowFn := output.NewArrowFunctionExpr(
			params,
			convertAst(a.Body, job, baseSourceSpan),
			nil,
			nil,
			nil,
		)
		return updateParameterReferences(arrowFn)
	default:
		panic(fmt.Sprintf("Unhandled expression type %T", ast))
	}
}

func isUndefinedKeywordLiteral(ast *expression_parser.LiteralPrimitive) bool {
	span := ast.GetSourceSpan()
	return span.End-span.Start == len("undefined")
}

func convertTemplateLiteral(
	ast *expression_parser.TemplateLiteral,
	job compilation.CompilationJob,
	baseSourceSpan *parse_util.ParseSourceSpan,
) *output.TemplateLiteralExpr {
	elements := make([]*output.TemplateLiteralElementExpr, len(ast.Elements))
	for i, el := range ast.Elements {
		elements[i] = output.NewTemplateLiteralElementExpr(
			el.Text,
			nil,
			nil,
			nil,
		)
	}
	exprs := make([]output.Expression, len(ast.Expressions))
	for i, expr := range ast.Expressions {
		exprs[i] = convertAst(expr, job, baseSourceSpan)
	}
	return output.NewTemplateLiteralExpr(
		elements,
		exprs,
		nil,
		nil,
	)
}

func convertAstWithInterpolation(
	job compilation.CompilationJob,
	value any,
	i18nMeta i18n.I18nMeta,
	sourceSpan *parse_util.ParseSourceSpan,
) output.Expression {
	var expression output.Expression
	if interp, ok := value.(*expression_parser.Interpolation); ok {
		exprs := make([]output.Expression, len(interp.Expressions))
		for i, expr := range interp.Expressions {
			exprs[i] = convertAst(expr, job, sourceSpan)
		}
		var keys []string
		if msg := asMessage(i18nMeta); msg != nil {
			for k := range msg.Placeholders {
				keys = append(keys, k)
			}
		}
		expression = ir.NewInterpolation(toStringSlice(interp.Strings), exprs, keys)
	} else if astNode, ok := value.(expression_parser.AST); ok {
		expression = convertAst(astNode, job, sourceSpan)
	} else if strVal, ok := value.(string); ok {
		expression = output.NewLiteralExpr(strVal, nil, nil, nil)
	} else {
		expression = output.NewLiteralExpr(value, nil, nil, nil)
	}
	return expression
}

var bindingKinds = map[template_parser.BindingType]ir.BindingKind{
	template_parser.BindingTypeProperty:        ir.BindingKindProperty,
	template_parser.BindingTypeTwoWay:          ir.BindingKindTwoWayProperty,
	template_parser.BindingTypeAttribute:       ir.BindingKindAttribute,
	template_parser.BindingTypeClass:           ir.BindingKindClassName,
	template_parser.BindingTypeStyle:           ir.BindingKindStyleProperty,
	template_parser.BindingTypeLegacyAnimation: ir.BindingKindLegacyAnimation,
	template_parser.BindingTypeAnimation:       ir.BindingKindAnimation,
}

func isPlainTemplate(tmpl *render3.Template) bool {
	if tmpl.TagName == nil {
		return false
	}
	_, name, _ := tags.SplitNsName(*tmpl.TagName, false)
	return name == ngTemplateTagName
}

func asMessage(i18nMeta i18n.I18nMeta) *i18n.Message {
	if i18nMeta == nil {
		return nil
	}
	msg, ok := i18nMeta.(*i18n.Message)
	if !ok {
		panic("expected i18nMeta to be of type *i18n.Message")
	}
	return msg
}

func asMessageAny(i18nMeta i18n.I18nMeta) any {
	if msg := asMessage(i18nMeta); msg != nil {
		return msg
	}
	return nil
}

func ingestElementBindings(
	unit *compilation.ViewCompilationUnit,
	op *ir.ElementStartOp,
	element *render3.Element,
) {
	var bindings []ir.Op
	i18nAttributeBindingNames := make(map[string]bool)

	ns, elementName, _ := tags.SplitNsName(element.Name, false)
	var namespace string
	if ns != nil {
		namespace = *ns
	} else {
		switch op.Namespace {
		case ir.NamespaceSVG:
			namespace = "svg"
		case ir.NamespaceMath:
			namespace = "math"
		}
	}

	for _, attr := range element.Attributes {
		var nsPrefix string
		if namespace != "" {
			nsPrefix = ":" + namespace + ":"
		}
		securityContext := domSchema.SecurityContext(
			nsPrefix+elementName,
			attr.Name,
			true,
		)
		bindings = append(bindings, ir.CreateBindingOp(
			op.Xref,
			ir.BindingKindAttribute,
			attr.Name,
			convertAstWithInterpolation(unit.Job, attr.Value, attr.I18n, &attr.SourceSpan),
			nil,
			securityContext,
			true,
			false,
			nil,
			asMessageAny(attr.I18n),
			&attr.SourceSpan,
		))
		if attr.I18n != nil {
			i18nAttributeBindingNames[attr.Name] = true
		}
	}

	for _, input := range element.Inputs {
		bindings = append(bindings, ir.CreateBindingOp(
			op.Xref,
			bindingKinds[template_parser.BindingType(input.Type)],
			input.Name,
			convertAstWithInterpolation(unit.Job, astOf(input.Value), input.I18n, &input.SourceSpan),
			input.Unit,
			input.SecurityContext,
			false,
			false,
			nil,
			asMessage(input.I18n),
			&input.SourceSpan,
		))
	}

	for _, b := range bindings {
		if b.Kind() == ir.OpKindExtractedAttribute {
			unit.Create.Push(b)
		} else if b.Kind() == ir.OpKindBinding {
			unit.Update.Push(b)
		}
	}

	for _, output := range element.Outputs {
		if output.Type == template_parser.ParsedEventTypeLegacyAnimation && (output.Phase == nil || *output.Phase == "") {
			panic("Animation listener should have a phase")
		}

		if output.Type == template_parser.ParsedEventTypeTwoWay {
			unit.Create.Push(
				ir.CreateTwoWayListenerOp(
					op.Xref,
					op.Handle(),
					output.Name,
					op.Tag,
					makeTwoWayListenerHandlerOps(unit, output.Handler, &output.SourceSpan),
					&output.SourceSpan,
				),
			)
		} else if output.Type == template_parser.ParsedEventTypeAnimation {
			unit.Create.Push(
				ir.CreateAnimationListenerOp(
					op.Xref,
					op.Handle(),
					output.Name,
					op.Tag,
					makeListenerHandlerOps(unit, output.Handler, &output.SourceSpan),
					func() ir.AnimationKind {
						if strings.HasSuffix(output.Name, "enter") {
							return ir.AnimationKindENTER
						}
						return ir.AnimationKindLEAVE
					}(),
					output.Target,
					false,
					&output.SourceSpan,
				),
			)
		} else {
			unit.Create.Push(
				ir.CreateListenerOp(
					op.Xref,
					op.Handle(),
					output.Name,
					op.Tag,
					makeListenerHandlerOps(unit, output.Handler, &output.SourceSpan),
					output.Phase,
					output.Target, // target
					false,
					&output.SourceSpan,
				),
			)
		}
	}

	// Always allocate an i18nAttributesOp xref, matching ngtsc behavior.
	// In ngtsc, the condition `bindings.some((b) => b?.i18nMessage) !== null` is always
	// true (JS quirk: .some() returns boolean, never null), so this op is always created.
	// Unused ops (no actual i18n messages) are removed later by RemoveUnusedI18nAttributesOps.
	unit.Create.Push(
		ir.CreateI18nAttributesOp(unit.Job.AllocateXrefId(), ir.NewSlotHandle(), op.Xref),
	)
}

func ingestTemplateBindings(
	unit *compilation.ViewCompilationUnit,
	op *ir.TemplateOp,
	template *render3.Template,
	templateKind ir.TemplateKind,
) {
	var bindings []ir.Op
	for _, attr := range template.TemplateAttrs {
		if textAttr, ok := attr.(*render3.TextAttribute); ok {
			securityContext := domSchema.SecurityContext(ngTemplateTagName, textAttr.Name, true)
			bindings = append(bindings, createTemplateBinding(
				unit,
				op.Xref,
				template_parser.BindingTypeAttribute,
				textAttr.Name,
				textAttr.Value,
				nil,
				securityContext,
				true,
				templateKind,
				asMessage(textAttr.I18n),
				&textAttr.SourceSpan,
			))
		} else if boundAttr, ok := attr.(*render3.BoundAttribute); ok {
			bindings = append(bindings, createTemplateBinding(
				unit,
				op.Xref,
				template_parser.BindingType(boundAttr.Type),
				boundAttr.Name,
				astOf(boundAttr.Value),
				boundAttr.Unit,
				boundAttr.SecurityContext,
				true,
				templateKind,
				asMessage(boundAttr.I18n),
				&boundAttr.SourceSpan,
			))
		}
	}

	for _, attr := range template.Attributes {
		securityContext := domSchema.SecurityContext(ngTemplateTagName, attr.Name, true)
		bindings = append(bindings, createTemplateBinding(
			unit,
			op.Xref,
			template_parser.BindingTypeAttribute,
			attr.Name,
			attr.Value,
			nil,
			securityContext,
			false,
			templateKind,
			asMessage(attr.I18n),
			&attr.SourceSpan,
		))
	}

	for _, input := range template.Inputs {
		bindings = append(bindings, createTemplateBinding(
			unit,
			op.Xref,
			template_parser.BindingType(input.Type),
			input.Name,
			astOf(input.Value),
			input.Unit,
			input.SecurityContext,
			false,
			templateKind,
			asMessage(input.I18n),
			&input.SourceSpan,
		))
	}

	for _, b := range bindings {
		if b == nil {
			continue
		}
		if b.Kind() == ir.OpKindExtractedAttribute {
			unit.Create.Push(b)
		} else if b.Kind() == ir.OpKindBinding {
			unit.Update.Push(b)
		}
	}

	for _, output := range template.Outputs {
		if output.Type == template_parser.ParsedEventTypeLegacyAnimation && (output.Phase == nil || *output.Phase == "") {
			panic("Animation listener should have a phase")
		}

		if templateKind == ir.TemplateKindNgTemplate {
			if output.Type == template_parser.ParsedEventTypeTwoWay {
				unit.Create.Push(
					ir.CreateTwoWayListenerOp(
						op.Xref,
						op.Handle(),
						output.Name,
						op.Tag,
						makeTwoWayListenerHandlerOps(unit, output.Handler, &output.SourceSpan),
						&output.SourceSpan,
					),
				)
			} else {
				unit.Create.Push(
					ir.CreateListenerOp(
						op.Xref,
						op.Handle(),
						output.Name,
						op.Tag,
						makeListenerHandlerOps(unit, output.Handler, &output.SourceSpan),
						output.Target, // phase
						output.Target, // target
						false,
						&output.SourceSpan,
					),
				)
			}
		}
		if templateKind == ir.TemplateKindStructural && output.Type != template_parser.ParsedEventTypeLegacyAnimation {
			securityContext := domSchema.SecurityContext(ngTemplateTagName, output.Name, false)
			unit.Create.Push(
				ir.CreateExtractedAttributeOp(
					op.Xref,
					ir.BindingKindProperty,
					nil,
					output.Name,
					nil,
					nil,
					nil,
					securityContext,
				),
			)
		}
	}

	// Always allocate an i18nAttributesOp xref, matching ngtsc behavior.
	// In ngtsc, the condition `bindings.some((b) => b?.i18nMessage) !== null` is always
	// true (JS quirk: .some() returns boolean, never null), so this op is always created.
	// Unused ops (no actual i18n messages) are removed later by RemoveUnusedI18nAttributesOps.
	unit.Create.Push(
		ir.CreateI18nAttributesOp(unit.Job.AllocateXrefId(), ir.NewSlotHandle(), op.Xref),
	)
}

func createTemplateBinding(
	view *compilation.ViewCompilationUnit,
	xref ir.XrefId,
	bindingType template_parser.BindingType,
	name string,
	value any,
	unit *string,
	securityContext core.SecurityContext,
	isStructuralTemplateAttribute bool,
	templateKind ir.TemplateKind,
	i18nMessage *i18n.Message,
	sourceSpan *parse_util.ParseSourceSpan,
) ir.Op {
	_, isTextBinding := value.(string)
	if templateKind == ir.TemplateKindStructural {
		if !isStructuralTemplateAttribute {
			switch bindingType {
			case template_parser.BindingTypeProperty, template_parser.BindingTypeClass, template_parser.BindingTypeStyle:
				return ir.CreateExtractedAttributeOp(
					xref,
					ir.BindingKindProperty,
					nil,
					name,
					nil,
					nil,
					i18nMessage,
					securityContext,
				)
			case template_parser.BindingTypeTwoWay:
				return ir.CreateExtractedAttributeOp(
					xref,
					ir.BindingKindTwoWayProperty,
					nil,
					name,
					nil,
					nil,
					i18nMessage,
					securityContext,
				)
			}
		}
		if !isTextBinding && (bindingType == template_parser.BindingTypeAttribute || bindingType == template_parser.BindingTypeLegacyAnimation || bindingType == template_parser.BindingTypeAnimation) {
			return nil
		}
	}

	bType := bindingKinds[bindingType]
	if templateKind == ir.TemplateKindNgTemplate {
		if bindingType == template_parser.BindingTypeClass || bindingType == template_parser.BindingTypeStyle || (bindingType == template_parser.BindingTypeAttribute && !isTextBinding) {
			bType = ir.BindingKindProperty
		}
	}

	return ir.CreateBindingOp(
		xref,
		bType,
		name,
		convertAstWithInterpolation(view.Job, value, i18nMessage, sourceSpan),
		unit,
		securityContext,
		isTextBinding,
		isStructuralTemplateAttribute,
		templateKind,
		i18nMessage,
		sourceSpan,
	)
}

func makeListenerHandlerOps(
	unit compilation.CompilationUnit,
	handler expression_parser.AST,
	handlerSpan *parse_util.ParseSourceSpan,
) *ir.OpList {
	handler = astOf(handler)
	handlerOps := ir.NewOpList()
	var handlerExprs []expression_parser.AST
	if chain, ok := handler.(*expression_parser.Chain); ok {
		handlerExprs = chain.Expressions
	} else {
		handlerExprs = []expression_parser.AST{handler}
	}
	if len(handlerExprs) == 0 {
		panic("Expected listener to have non-empty expression list.")
	}
	expressions := make([]output.Expression, len(handlerExprs))
	for i, expr := range handlerExprs {
		expressions[i] = convertAst(expr, unit.GetJob(), handlerSpan)
	}
	returnExpr := expressions[len(expressions)-1]
	expressions = expressions[:len(expressions)-1]

	for _, e := range expressions {
		handlerOps.Push(&ir.StatementOp{Statement: output.NewExpressionStatement(e, nil, nil)})
	}
	handlerOps.Push(&ir.StatementOp{Statement: output.NewReturnStatement(returnExpr, nil, nil)})
	return handlerOps
}

func makeTwoWayListenerHandlerOps(
	unit compilation.CompilationUnit,
	handler expression_parser.AST,
	handlerSpan *parse_util.ParseSourceSpan,
) *ir.OpList {
	handler = astOf(handler)
	handlerOps := ir.NewOpList()

	if chain, ok := handler.(*expression_parser.Chain); ok {
		if len(chain.Expressions) == 1 {
			handler = chain.Expressions[0]
		} else {
			panic("Expected two-way listener to have a single expression.")
		}
	}

	handlerExpr := convertAst(handler, unit.GetJob(), handlerSpan)
	eventReference := ir.NewLexicalReadExpr("$event")
	twoWaySetExpr := ir.NewTwoWayBindingSetExpr(handlerExpr, eventReference)

	handlerOps.Push(&ir.StatementOp{Statement: output.NewExpressionStatement(twoWaySetExpr, nil, nil)})
	handlerOps.Push(&ir.StatementOp{Statement: output.NewReturnStatement(eventReference, nil, nil)})
	return handlerOps
}

func astOf(ast expression_parser.AST) expression_parser.AST {
	if astWithSrc, ok := ast.(*expression_parser.ASTWithSource); ok {
		return astWithSrc.Ast
	}
	return ast
}

func convertSourceSpan(
	span expression_parser.ParseSpan,
	baseSourceSpan *parse_util.ParseSourceSpan,
) *parse_util.ParseSourceSpan {
	if baseSourceSpan == nil {
		return nil
	}
	start := baseSourceSpan.Start.MoveBy(span.Start)
	end := baseSourceSpan.Start.MoveBy(span.End)
	var fullStart *parse_util.ParseLocation
	if baseSourceSpan.FullStart != nil {
		fullStart = baseSourceSpan.FullStart.MoveBy(span.Start)
	}
	return parse_util.NewParseSourceSpan(start, end, fullStart, nil)
}

func ingestControlFlowInsertionPoint(
	unit *compilation.ViewCompilationUnit,
	xref ir.XrefId,
	node render3.Node,
) *string {
	var children []render3.Node
	switch n := node.(type) {
	case *render3.IfBlockBranch:
		children = n.Children
	case *render3.SwitchBlockCaseGroup:
		children = n.Children
	case *render3.ForLoopBlock:
		children = n.Children
	case *render3.ForLoopBlockEmpty:
		children = n.Children
	}

	var root render3.Node

	for _, child := range children {
		if _, isComment := child.(*render3.Comment); isComment {
			continue
		}
		if _, isLet := child.(*render3.LetDeclaration); isLet {
			continue
		}

		if root != nil {
			return nil
		}

		if el, isEl := child.(*render3.Element); isEl {
			root = el
		} else if tmpl, isTmpl := child.(*render3.Template); isTmpl && tmpl.TagName != nil {
			root = tmpl
		} else {
			return nil
		}
	}

	if root != nil {
		var rootAttrs []*render3.TextAttribute
		var rootInputs []*render3.BoundAttribute
		var rootName string

		if el, isEl := root.(*render3.Element); isEl {
			rootAttrs = el.Attributes
			rootInputs = el.Inputs
			rootName = el.Name
		} else if tmpl, isTmpl := root.(*render3.Template); isTmpl {
			rootAttrs = tmpl.Attributes
			rootInputs = tmpl.Inputs
			rootName = *tmpl.TagName
		}

		for _, attr := range rootAttrs {
			if !strings.HasPrefix(attr.Name, animatePrefix) {
				securityContext := domSchema.SecurityContext(ngTemplateTagName, attr.Name, true)
				unit.Update.Push(
					ir.CreateBindingOp(
						xref,
						ir.BindingKindAttribute,
						attr.Name,
						output.NewLiteralExpr(attr.Value, nil, nil, nil),
						nil,
						securityContext,
						true,
						false,
						nil,
						asMessage(attr.I18n),
						&attr.SourceSpan,
					),
				)
			}
		}

		for _, attr := range rootInputs {
			if template_parser.BindingType(attr.Type) != template_parser.BindingTypeLegacyAnimation &&
				template_parser.BindingType(attr.Type) != template_parser.BindingTypeAnimation &&
				template_parser.BindingType(attr.Type) != template_parser.BindingTypeAttribute {
				securityContext := domSchema.SecurityContext(ngTemplateTagName, attr.Name, true)
				unit.Create.Push(
					ir.CreateExtractedAttributeOp(
						xref,
						ir.BindingKindProperty,
						nil,
						attr.Name,
						nil,
						nil,
						nil,
						securityContext,
					),
				)
			}
		}

		if rootName == ngTemplateTagName {
			return nil
		}
		return &rootName
	}

	return nil
}

func updateParameterReferences(root *output.ArrowFunctionExpr) *output.ArrowFunctionExpr {
	parameterNames := make(map[string]bool)
	for _, param := range root.Params {
		parameterNames[param.Name] = true
	}

	transformed := ir.TransformExpressionsInExpression(
		root,
		func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			if arrowFn, ok := expr.(*output.ArrowFunctionExpr); ok {
				for _, param := range arrowFn.Params {
					parameterNames[param.Name] = true
				}
			} else if lexRead, ok := expr.(*ir.LexicalReadExpr); ok {
				if parameterNames[lexRead.Name] {
					return output.NewReadVarExpr(lexRead.Name, nil, nil, nil)
				}
			}
			return expr
		},
		ir.VisitorContextFlagNone,
	)
	return transformed.(*output.ArrowFunctionExpr)
}

func namespaceForKey(namespacePrefixKey *string) any {
	if namespacePrefixKey == nil {
		return ir.NamespaceHTML
	}
	switch *namespacePrefixKey {
	case "svg":
		return ir.NamespaceSVG
	case "math":
		return ir.NamespaceMath
	default:
		return ir.NamespaceHTML
	}
}

func prefixWithNamespace(strippedTag string, namespace any) string {
	ns, ok := namespace.(ir.Namespace)
	if !ok || ns == ir.NamespaceHTML {
		return strippedTag
	}
	var nsStr string
	switch ns {
	case ir.NamespaceSVG:
		nsStr = "svg"
	case ir.NamespaceMath:
		nsStr = "math"
	}
	return ":" + nsStr + ":" + strippedTag
}

var binaryOperators = map[string]output.BinaryOperator{
	"&&":         output.BinaryOperatorAnd,
	">":          output.BinaryOperatorBigger,
	">=":         output.BinaryOperatorBiggerEquals,
	"|":          output.BinaryOperatorBitwiseOr,
	"&":          output.BinaryOperatorBitwiseAnd,
	"/":          output.BinaryOperatorDivide,
	"=":          output.BinaryOperatorAssign,
	"==":         output.BinaryOperatorEquals,
	"===":        output.BinaryOperatorIdentical,
	"<":          output.BinaryOperatorLower,
	"<=":         output.BinaryOperatorLowerEquals,
	"-":          output.BinaryOperatorMinus,
	"%":          output.BinaryOperatorModulo,
	"**":         output.BinaryOperatorExponentiation,
	"*":          output.BinaryOperatorMultiply,
	"!=":         output.BinaryOperatorNotEquals,
	"!==":        output.BinaryOperatorNotIdentical,
	"??":         output.BinaryOperatorNullishCoalesce,
	"||":         output.BinaryOperatorOr,
	"+":          output.BinaryOperatorPlus,
	"in":         output.BinaryOperatorIn,
	"instanceof": output.BinaryOperatorInstanceOf,
	"+=":         output.BinaryOperatorAdditionAssignment,
	"-=":         output.BinaryOperatorSubtractionAssignment,
	"*=":         output.BinaryOperatorMultiplicationAssignment,
	"/=":         output.BinaryOperatorDivisionAssignment,
	"%=":         output.BinaryOperatorRemainderAssignment,
	"**=":        output.BinaryOperatorExponentiationAssignment,
	"&&=":        output.BinaryOperatorAndAssignment,
	"||=":        output.BinaryOperatorOrAssignment,
	"??=":        output.BinaryOperatorNullishCoalesceAssignment,
}

func toStringSlice(slice []any) []string {
	res := make([]string, len(slice))
	for i, val := range slice {
		if s, ok := val.(string); ok {
			res[i] = s
		} else {
			res[i] = fmt.Sprintf("%v", val)
		}
	}
	return res
}

func convertParseSourceSpan(span expression_parser.ParseSourceSpan) *parse_util.ParseSourceSpan {
	sourceFile := parse_util.NewParseSourceFile("", "")
	start := parse_util.NewParseLocation(sourceFile, span.Start, 0, 0)
	end := parse_util.NewParseLocation(sourceFile, span.End, 0, 0)
	fullStart := parse_util.NewParseLocation(sourceFile, span.FullStart, 0, 0)
	return parse_util.NewParseSourceSpan(start, end, fullStart, nil)
}

type LocalRefsOp interface {
	GetLocalRefs() any
	SetLocalRefs(refs any)
}

func ingestReferences(op LocalRefsOp, references []*render3.Reference) {
	var refs []ir.LocalRef
	if slice, ok := op.GetLocalRefs().([]ir.LocalRef); ok {
		refs = slice
	}
	for _, ref := range references {
		refs = append(refs, ir.LocalRef{
			Name:   ref.Name,
			Target: ref.Value,
		})
	}
	op.SetLocalRefs(refs)
}

func float64PtrToIntPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	val := int(*f)
	return &val
}

func getTriggerSpan(t interface {
	GetSourceSpan() parse_util.ParseSourceSpan
}) *parse_util.ParseSourceSpan {
	span := t.GetSourceSpan()
	return &span
}
