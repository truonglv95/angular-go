package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

type DeclareComponentTemplateInfo struct {
	Content                         string
	SourceUrl                       string
	IsInline                        bool
	InlineTemplateLiteralExpression output.Expression // nullable
}

// CompileDeclareComponentFromMetadata Compiles a component declaration defined by the R3ComponentMetadata.
func CompileDeclareComponentFromMetadata(
	meta render3.R3ComponentMetadata[render3.R3TemplateDependencyMetadata],
	template template_parser.ParsedTemplate,
	additionalTemplateInfo DeclareComponentTemplateInfo,
) render3.R3CompiledExpression {
	definitionMap := CreateComponentDefinitionMap(meta, template, additionalTemplateInfo)

	expression := render3.ImportExpr(*render3.Identifiers.DeclareComponent).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
	typeExpr := output.INFERRED_TYPE

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func CreateComponentDefinitionMap(
	meta render3.R3ComponentMetadata[render3.R3TemplateDependencyMetadata],
	template template_parser.ParsedTemplate,
	templateInfo DeclareComponentTemplateInfo,
) *render3.DefinitionMap {
	definitionMap := CreateDirectiveDefinitionMap(meta.R3DirectiveMetadata)
	blockVisitor := &BlockPresenceVisitor{}
	blockVisitor.Impl = blockVisitor

	// Assuming render3.VisitAll exists
	var r3nodes []render3.Node
	for _, n := range template.Nodes {
		if r3node, ok := n.(render3.Node); ok {
			r3nodes = append(r3nodes, r3node)
		}
	}
	render3.VisitAll(blockVisitor, r3nodes)

	definitionMap.Set("template", getTemplateExpression(template, templateInfo))

	if templateInfo.IsInline {
		definitionMap.Set("isInline", render3.LiteralExpr(true))
	}

	if blockVisitor.hasBlocks {
		definitionMap.Set("minVersion", render3.LiteralExpr("17.0.0"))
	}

	if styles := ToOptionalLiteralArray(meta.Styles, func(s string) output.Expression { return render3.LiteralExpr(s) }); styles != nil {
		definitionMap.Set("styles", styles)
	}

	if deps := compileUsedDependenciesMetadata(meta); deps != nil {
		definitionMap.Set("dependencies", deps)
	}

	if meta.ViewProviders != nil {
		definitionMap.Set("viewProviders", meta.ViewProviders.(output.Expression))
	}

	if meta.Animations != nil {
		definitionMap.Set("animations", meta.Animations.(output.Expression))
	}

	if meta.ChangeDetection != nil {
		cdInt, _ := meta.ChangeDetection.(int)
		// core.ChangeDetectionStrategy to property Name mapping - we assume it has a String() or we do a cast
		// TS does: o.importExpr(R3.ChangeDetectionStrategy).prop(core.ChangeDetectionStrategy[meta.changeDetection])
		// For Go, we'll map the int to string using a helper or assume it's represented correctly
		var cdName string
		switch core.ChangeDetectionStrategy(cdInt) {
		case core.ChangeDetectionStrategyOnPush:
			cdName = "OnPush"
		case core.ChangeDetectionStrategyDefault:
			cdName = "Default"
		}
		definitionMap.Set("changeDetection", render3.ImportExpr(*render3.Identifiers.ChangeDetectionStrategy).Prop(cdName, nil))
	}

	if meta.Encapsulation != 0 /* Emulated */ {
		var encName string
		switch core.ViewEncapsulation(meta.Encapsulation) {
		case core.ViewEncapsulationEmulated:
			encName = "Emulated"
		case core.ViewEncapsulationNone:
			encName = "None"
		case core.ViewEncapsulationShadowDom:
			encName = "ShadowDom"
		}
		definitionMap.Set("encapsulation", render3.ImportExpr(*render3.Identifiers.ViewEncapsulation).Prop(encName, nil))
	}

	if meta.Defer.Mode == render3.DeferBlockDepsEmitMode_PerBlock {
		var resolvers []output.Expression
		hasResolvers := false

		for _, deps := range meta.Defer.Blocks {
			if deps == nil {
				resolvers = append(resolvers, render3.LiteralExpr(nil))
			} else {
				resolvers = append(resolvers, deps.(output.Expression))
				hasResolvers = true
			}
		}

		if hasResolvers {
			definitionMap.Set("deferBlockDependencies", render3.LiteralArr(resolvers))
		}
	} else if true {
		panic("Unsupported defer function emit mode in partial compilation")
	}

	return definitionMap
}

func getTemplateExpression(
	template template_parser.ParsedTemplate,
	templateInfo DeclareComponentTemplateInfo,
) output.Expression {
	if templateInfo.InlineTemplateLiteralExpression != nil {
		return templateInfo.InlineTemplateLiteralExpression
	}

	if templateInfo.IsInline {
		return output.NewLiteralExpr(templateInfo.Content, nil, nil, nil)
	}

	contents := templateInfo.Content
	return output.NewLiteralExpr(contents, nil, nil, nil)
}

func computeEndLocation(file *parse_util.ParseSourceFile, contents string) *parse_util.ParseLocation {
	length := len(contents)
	lineStart := 0
	lastLineStart := 0
	line := 0

	for {
		// find next newline
		foundIndex := -1
		for i := lastLineStart; i < length; i++ {
			if contents[i] == '\n' {
				foundIndex = i
				break
			}
		}

		lineStart = foundIndex
		if lineStart != -1 {
			lastLineStart = lineStart + 1
			line++
		} else {
			break
		}
	}

	return parse_util.NewParseLocation(file, length, line, length-lastLineStart)
}

func compileUsedDependenciesMetadata(
	meta render3.R3ComponentMetadata[render3.R3TemplateDependencyMetadata],
) *output.LiteralArrayExpr {

	wrapType := func(expr output.Expression) output.Expression {
		return expr
	}
	if meta.DeclarationListEmitMode != render3.DeclarationListEmitMode_Direct {
		wrapType = render3.GenerateForwardRef
	}

	if meta.DeclarationListEmitMode == render3.DeclarationListEmitMode_RuntimeResolved {
		panic("Unsupported emit mode")
	}

	return ToOptionalLiteralArray(meta.Declarations, func(decl render3.R3TemplateDependencyMetadata) output.Expression {
		switch d := decl.(type) {
		case render3.R3DirectiveDependencyMetadata:
			dirMeta := render3.NewDefinitionMap()
			kindStr := "directive"
			if d.IsComponent {
				kindStr = "component"
			}
			dirMeta.Set("kind", render3.LiteralExpr(kindStr))
			dirMeta.Set("type", wrapType(d.Type.(output.Expression)))
			dirMeta.Set("selector", render3.LiteralExpr(d.Selector))
			if inputs := ToOptionalLiteralArray(d.Inputs, func(s string) output.Expression { return render3.LiteralExpr(s) }); inputs != nil {
				dirMeta.Set("inputs", inputs)
			}
			if outputs := ToOptionalLiteralArray(d.Outputs, func(s string) output.Expression { return render3.LiteralExpr(s) }); outputs != nil {
				dirMeta.Set("outputs", outputs)
			}
			if exportAs := ToOptionalLiteralArray(d.ExportAs, func(s string) output.Expression { return render3.LiteralExpr(s) }); exportAs != nil {
				dirMeta.Set("exportAs", exportAs)
			}
			return dirMeta.ToLiteralMap()
		case render3.R3PipeDependencyMetadata:
			pipeMeta := render3.NewDefinitionMap()
			pipeMeta.Set("kind", render3.LiteralExpr("pipe"))
			pipeMeta.Set("type", wrapType(d.Type.(output.Expression)))
			pipeMeta.Set("name", render3.LiteralExpr(d.Name))
			return pipeMeta.ToLiteralMap()
		case render3.R3NgModuleDependencyMetadata:
			ngModuleMeta := render3.NewDefinitionMap()
			ngModuleMeta.Set("kind", render3.LiteralExpr("ngmodule"))
			ngModuleMeta.Set("type", wrapType(d.Type.(output.Expression)))
			return ngModuleMeta.ToLiteralMap()
		default:
			return nil
		}
	})
}

type BlockPresenceVisitor struct {
	render3.RecursiveVisitor
	hasBlocks bool
}

func (v *BlockPresenceVisitor) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitIfBlock(block *render3.IfBlock) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	v.hasBlocks = true
	return nil
}

func (v *BlockPresenceVisitor) VisitSwitchBlockCaseGroup(block *render3.SwitchBlockCaseGroup) interface{} {
	v.hasBlocks = true
	return nil
}
