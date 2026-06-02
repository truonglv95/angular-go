package pipeline

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/phases"
)

func init() {
	render3.Transform = Transform
	render3.TransformHostBinding = TransformHostBinding
	compilation.ParseSelectorToR3Selector = func(selector *string) []any {
		if selector == nil {
			return nil
		}
		res := compiler.ParseSelectorToR3Selector(selector)
		var anyRes []any
		for _, r := range res {
			anyRes = append(anyRes, r)
		}
		return anyRes
	}
}

func Transform(job *compilation.ComponentCompilationJob) {
	phases.RemoveContentSelectors(job)
	phases.OptimizeRegularExpressions(job)
	phases.EmitNamespaceChanges(job)
	phases.PropagateI18nBlocks(job)
	phases.WrapI18nIcus(job)
	phases.DeduplicateTextBindings(job)
	phases.SpecializeBindings(job)
	phases.SpecializeControlProperties(job)
	phases.ConvertAnimations(job)
	phases.ExtractAttributes(job)
	phases.CreateI18nContexts(job)
	phases.ParseExtractedStyles(job)
	phases.RemoveEmptyBindings(job)
	phases.CollapseSingletonInterpolations(job)
	phases.OrderOps(job)
	phases.GenerateConditionalExpressions(job)
	phases.CreatePipes(job)
	phases.ConfigureDeferInstructions(job)
	phases.InsertIncrementalHydrationRuntime(job)
	phases.CreateVariadicPipes(job)
	phases.GenerateArrowFunctions(job)
	phases.GeneratePureLiteralStructures(job)
	phases.GenerateProjectionDefs(job)
	phases.GenerateLocalLetReferences(job)
	phases.GenerateVariables(job)
	phases.SaveAndRestoreView(job)
	phases.DeleteAnyCasts(job)
	phases.RemoveSafeNavigationMigration(job)
	phases.ResolveDollarEvent(job)
	phases.GenerateTrackVariables(job)
	phases.RemoveIllegalLetReferences(job)
	phases.ResolveNames(job)
	phases.ResolveDeferTargetNames(job)
	phases.TransformTwoWayBindingSet(job)
	phases.OptimizeTrackFns(job)
	phases.ResolveContexts(job)
	phases.ResolveSanitizers(job)
	phases.LiftLocalRefs(job)
	phases.ExpandSafeReads(job)
	phases.StripNonrequiredParentheses(job)
	phases.GenerateTemporaryVariables(job)
	phases.OptimizeVariables(job)
	phases.OptimizeStoreLet(job)
	phases.ConvertI18nText(job)
	phases.ConvertI18nBindings(job)
	phases.RemoveUnusedI18nAttributesOps(job)
	phases.AssignI18nSlotDependencies(job)
	phases.ApplyI18nExpressions(job)
	phases.AllocateSlots(job)
	phases.ResolveI18nElementPlaceholders(job)
	phases.ResolveI18nExpressionPlaceholders(job)
	phases.ExtractI18nMessages(job)
	phases.CollectI18nConsts(job)
	phases.ResolveI18nAttrSanitizers(job)
	phases.CollectConstExpressions(job)
	phases.CollectElementConsts(job)
	phases.RemoveI18nContexts(job)
	phases.CountVariables(job)
	phases.GenerateAdvance(job)
	phases.NameFunctionsAndVariables(job)
	phases.ResolveDeferDepsFns(job)
	phases.MergeNextContextExpressions(job)
	phases.GenerateNgContainerOps(job)
	phases.CollapseEmptyInstructions(job)
	phases.DisableBindings(job)
	phases.ExtractPureFunctions(job)
	phases.Reify(job)
	phases.Chain(job)
}

func TransformHostBinding(job *compilation.HostBindingCompilationJob) {
	phases.ParseHostStyleProperties(job)
	phases.SpecializeBindings(job)
	phases.DeleteAnyCasts(job)
	phases.ResolveDollarEvent(job)
	phases.ResolveNames(job)
	phases.ResolveContexts(job)
	phases.ResolveSanitizers(job)
	phases.ExpandSafeReads(job)
	phases.StripNonrequiredParentheses(job)
	phases.OptimizeStoreLet(job)
	phases.TransformTwoWayBindingSet(job)
	phases.GenerateAdvance(job)
	phases.GenerateTemporaryVariables(job)
	phases.CollapseEmptyInstructions(job)
	phases.OptimizeVariables(job)
	phases.NameFunctionsAndVariables(job)
	phases.CountVariables(job)
	phases.Reify(job)
	phases.Chain(job)
}
