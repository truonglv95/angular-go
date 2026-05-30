package perf

type PerfPhase int

const (
	/**
	 * The "default" phase which tracks time not spent in any other phase.
	 */
	PerfPhase_Unaccounted PerfPhase = iota

	/**
	 * Time spent setting up the compiler, before a TypeScript program is created.
	 *
	 * This includes operations like configuring the `ts.CompilerHost` and any wrappers.
	 */
	PerfPhase_Setup

	/**
	 * Time spent in `ts.createProgram`, including reading and parsing `ts.SourceFile`s in the
	 * `ts.CompilerHost`.
	 *
	 * This might be an incremental program creation operation.
	 */
	PerfPhase_TypeScriptProgramCreate

	/**
	 * Time spent reconciling the contents of an old `ts.Program` with the new incremental one.
	 *
	 * Only present in incremental compilations.
	 */
	PerfPhase_Reconciliation

	/**
	 * Time spent updating an `NgCompiler` instance with a resource-only change.
	 *
	 * Only present in incremental compilations where the change was resource-only.
	 */
	PerfPhase_ResourceUpdate

	/**
	 * Time spent calculating the plain TypeScript diagnostics (structural and semantic).
	 */
	PerfPhase_TypeScriptDiagnostics

	/**
	 * Time spent in Angular analysis of individual classes in the program.
	 */
	PerfPhase_Analysis

	/**
	 * Time spent in Angular global analysis (synthesis of analysis information into a complete
	 * understanding of the program).
	 */
	PerfPhase_Resolve

	/**
	 * Time spent building the import graph of the program in order to perform cycle detection.
	 */
	PerfPhase_CycleDetection

	/**
	 * Time spent generating the text of Type Check Blocks in order to perform template type checking.
	 */
	PerfPhase_TcbGeneration

	/**
	 * Time spent updating the `ts.Program` with new Type Check Block code.
	 */
	PerfPhase_TcbUpdateProgram

	/**
	 * Time spent by TypeScript performing its emit operations, including downleveling and writing
	 * output files.
	 */
	PerfPhase_TypeScriptEmit

	/**
	 * Time spent by Angular performing code transformations of ASTs as they're about to be emitted.
	 *
	 * This includes the actual code generation step for templates, and occurs during the emit phase
	 * (but is tracked separately from `TypeScriptEmit` time).
	 */
	PerfPhase_Compile

	/**
	 * Time spent performing a `TemplateTypeChecker` autocompletion operation.
	 */
	PerfPhase_TtcAutocompletion

	/**
	 * Time spent computing template type-checking diagnostics.
	 */
	PerfPhase_TtcDiagnostics

	/**
	 * Time spent computing template type-checking suggestion diagnostics.
	 */
	PerfPhase_TtcSuggestionDiagnostics

	/**
	 * Time spent getting a `Symbol` from the `TemplateTypeChecker`.
	 */
	PerfPhase_TtcSymbol

	/**
	 * Time spent by the Angular Language Service calculating a "get references" or a renaming
	 * operation.
	 */
	PerfPhase_LsReferencesAndRenames

	/**
	 * Time spent by the Angular Language Service calculating a "quick info" operation.
	 */
	PerfPhase_LsQuickInfo

	/**
	 * Time spent by the Angular Language Service calculating a "get type definition" or "get
	 * definition" operation.
	 */
	PerfPhase_LsDefinition

	/**
	 * Time spent by the Angular Language Service calculating a "get completions" (AKA autocomplete)
	 * operation.
	 */
	PerfPhase_LsCompletions

	/**
	 * Time spent by the Angular Language Service calculating a "view template typecheck block"
	 * operation.
	 */
	PerfPhase_LsTcb

	/**
	 * Time spent by the Angular Language Service calculating diagnostics.
	 */
	PerfPhase_LsDiagnostics

	/**
	 * Time spent by the Angular Language Service calculating suggestion diagnostics.
	 */
	PerfPhase_LsSuggestionDiagnostics

	/**
	 * Time spent by the Angular Language Service calculating a "get component locations for template"
	 * operation.
	 */
	PerfPhase_LsComponentLocations

	/**
	 * Time spent by the Angular Language Service calculating signature help.
	 */
	PerfPhase_LsSignatureHelp

	/**
	 * Time spent by the Angular Language Service calculating outlining spans.
	 */
	PerfPhase_OutliningSpans

	/**
	 * Time spent by the Angular Language Service calculating code fixes.
	 */
	PerfPhase_LsCodeFixes

	/**
	 * Time spent by the Angular Language Service to fix all detected same type errors.
	 */
	PerfPhase_LsCodeFixesAll

	/**
	 * Time spent computing possible Angular refactorings.
	 */
	PerfPhase_LSComputeApplicableRefactorings

	/**
	 * Time spent computing changes for applying a given refactoring.
	 */
	PerfPhase_LSApplyRefactoring

	/**
	 * Time spent by the Angular Language Service calculating semantic classifications.
	 */
	PerfPhase_LSSemanticClassification

	/**
	 * Tracks the number of `PerfPhase`s, and must appear at the end of the list.
	 */
	PerfPhase_LAST
)

type PerfEvent int

const (
	/**
	 * Counts the number of `.d.ts` files in the program.
	 */
	PerfEvent_InputDtsFile PerfEvent = iota

	/**
	 * Counts the number of non-`.d.ts` files in the program.
	 */
	PerfEvent_InputTsFile

	/**
	 * An `@Component` class was analyzed.
	 */
	PerfEvent_AnalyzeComponent

	/**
	 * An `@Directive` class was analyzed.
	 */
	PerfEvent_AnalyzeDirective

	/**
	 * An `@Injectable` class was analyzed.
	 */
	PerfEvent_AnalyzeInjectable

	/**
	 * An `@NgModule` class was analyzed.
	 */
	PerfEvent_AnalyzeNgModule

	/**
	 * An `@Pipe` class was analyzed.
	 */
	PerfEvent_AnalyzePipe

	/**
	 * An `@Service` class was analyzed.
	 */
	PerfEvent_AnalyzeService

	/**
	 * A trait was analyzed.
	 *
	 * In theory, this should be the sum of the `Analyze` counters for each decorator type.
	 */
	PerfEvent_TraitAnalyze

	/**
	 * A trait had a prior analysis available from an incremental program, and did not need to be
	 * re-analyzed.
	 */
	PerfEvent_TraitReuseAnalysis

	/**
	 * A `ts.SourceFile` directly changed between the prior program and a new incremental compilation.
	 */
	PerfEvent_SourceFilePhysicalChange

	/**
	 * A `ts.SourceFile` did not physically changed, but according to the file dependency graph, has
	 * logically changed between the prior program and a new incremental compilation.
	 */
	PerfEvent_SourceFileLogicalChange

	/**
	 * A `ts.SourceFile` has not logically changed and all of its analysis results were thus available
	 * for reuse.
	 */
	PerfEvent_SourceFileReuseAnalysis

	/**
	 * A Type Check Block (TCB) was generated.
	 */
	PerfEvent_GenerateTcb

	/**
	 * A Type Check Block (TCB) could not be generated because inlining was disabled, and the block
	 * would've required inlining.
	 */
	PerfEvent_SkipGenerateTcbNoInline

	/**
	 * A `.ngtypecheck.ts` file could be reused from the previous program and did not need to be
	 * regenerated.
	 */
	PerfEvent_ReuseTypeCheckFile

	/**
	 * The template type-checking program required changes and had to be updated in an incremental
	 * step.
	 */
	PerfEvent_UpdateTypeCheckProgram

	/**
	 * The compiler was able to prove that a `ts.SourceFile` did not need to be re-emitted.
	 */
	PerfEvent_EmitSkipSourceFile

	/**
	 * A `ts.SourceFile` was emitted.
	 */
	PerfEvent_EmitSourceFile

	/**
	 * Tracks the number of `PerfEvent`s, and must appear at the end of the list.
	 */
	PerfEvent_LAST
)

type PerfCheckpoint int

const (
	/**
	 * The point at which the `PerfRecorder` was created, and ideally tracks memory used before any
	 * compilation structures are created.
	 */
	PerfCheckpoint_Initial PerfCheckpoint = iota

	/**
	 * The point just after the `ts.Program` has been created.
	 */
	PerfCheckpoint_TypeScriptProgramCreate

	/**
	 * The point just before Angular analysis starts.
	 *
	 * In the main usage pattern for the compiler, TypeScript diagnostics have been calculated at this
	 * point, so the `ts.TypeChecker` has fully ingested the current program, all `ts.Type` structures
	 * and `ts.Symbol`s have been created.
	 */
	PerfCheckpoint_PreAnalysis

	/**
	 * The point just after Angular analysis completes.
	 */
	PerfCheckpoint_Analysis

	/**
	 * The point just after Angular resolution is complete.
	 */
	PerfCheckpoint_Resolve

	/**
	 * The point just after Type Check Blocks (TCBs) have been generated.
	 */
	PerfCheckpoint_TtcGeneration

	/**
	 * The point just after the template type-checking program has been updated with any new TCBs.
	 */
	PerfCheckpoint_TtcUpdateProgram

	/**
	 * The point just before emit begins.
	 *
	 * In the main usage pattern for the compiler, all template type-checking diagnostics have been
	 * requested at this point.
	 */
	PerfCheckpoint_PreEmit

	/**
	 * The point just after the program has been fully emitted.
	 */
	PerfCheckpoint_Emit

	/**
	 * Tracks the number of `PerfCheckpoint`s, and must appear at the end of the list.
	 */
	PerfCheckpoint_LAST
)

type PerfRecorder interface {
	Phase(phase PerfPhase) PerfPhase
	InPhase(phase PerfPhase, fn func() interface{}) interface{}
	Memory(after PerfCheckpoint)
	EventCount(event PerfEvent, incrementBy ...int)
	Reset()
}
