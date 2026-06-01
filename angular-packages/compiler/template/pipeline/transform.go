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
	phases.GenerateNgContainerOps(job)
	phases.RemoveContentSelectors(job)
	phases.OptimizeRegularExpressions(job)
	phases.EmitNamespaceChanges(job)
	phases.DeduplicateTextBindings(job)
	phases.SpecializeStyleBindings(job)
	phases.SpecializeBindings(job)
	phases.ConvertAnimations(job)
	phases.ExtractAttributes(job)
	phases.ParseExtractedStyles(job)
	phases.CollectElementConsts(job)
	phases.RemoveEmptyBindings(job)
	phases.CollapseSingletonInterpolations(job)
	phases.OrderOps(job)
	phases.CreatePipes(job)
	phases.CreateVariadicPipes(job)
	phases.GenerateTrackVariables(job)
	phases.OptimizeTrackFns(job)
	phases.GenerateArrowFunctions(job)
	phases.GeneratePureLiteralStructures(job)
	phases.ExtractPureFunctions(job)
	phases.GenerateProjectionDefs(job)
	phases.GenerateLocalLetReferences(job)
	phases.GenerateVariables(job)
	phases.SaveAndRestoreView(job)
	phases.DeleteAnyCasts(job)
	phases.GenerateConditionalExpressions(job)
	phases.ResolveDollarEvent(job)
	phases.ResolveNames(job)
	phases.LiftLocalRefs(job)
	phases.ResolveContexts(job)
	phases.ResolveSanitizers(job)
	phases.ExpandSafeReads(job)
	phases.StripNonrequiredParentheses(job)
	phases.OptimizeStoreLet(job)
	phases.TransformTwoWayBindingSet(job)
	phases.AllocateSlots(job)
	phases.GenerateAdvance(job)
	phases.GenerateTemporaryVariables(job)
	phases.RemoveUnusedI18nAttributesOps(job)
	phases.CollapseEmptyInstructions(job)
	phases.OptimizeVariables(job)
	phases.NameFunctionsAndVariables(job)
	phases.CountVariables(job)
	phases.Reify(job)
	phases.Chain(job)
}

func TransformHostBinding(job *compilation.HostBindingCompilationJob) {
	phases.ParseHostStyleProperties(job)
	phases.SpecializeStyleBindings(job)
	phases.SpecializeBindings(job)
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
