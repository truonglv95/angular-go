package diagnostics

type ExtendedTemplateDiagnosticName string

const (
	ExtendedTemplateDiagnosticName_INVALID_BANANA_IN_BOX                      ExtendedTemplateDiagnosticName = "invalidBananaInBox"
	ExtendedTemplateDiagnosticName_NULLISH_COALESCING_NOT_NULLABLE            ExtendedTemplateDiagnosticName = "nullishCoalescingNotNullable"
	ExtendedTemplateDiagnosticName_OPTIONAL_CHAIN_NOT_NULLABLE                ExtendedTemplateDiagnosticName = "optionalChainNotNullable"
	ExtendedTemplateDiagnosticName_MISSING_CONTROL_FLOW_DIRECTIVE             ExtendedTemplateDiagnosticName = "missingControlFlowDirective"
	ExtendedTemplateDiagnosticName_MISSING_STRUCTURAL_DIRECTIVE               ExtendedTemplateDiagnosticName = "missingStructuralDirective"
	ExtendedTemplateDiagnosticName_TEXT_ATTRIBUTE_NOT_BINDING                 ExtendedTemplateDiagnosticName = "textAttributeNotBinding"
	ExtendedTemplateDiagnosticName_UNINVOKED_FUNCTION_IN_EVENT_BINDING        ExtendedTemplateDiagnosticName = "uninvokedFunctionInEventBinding"
	ExtendedTemplateDiagnosticName_MISSING_NGFOROF_LET                        ExtendedTemplateDiagnosticName = "missingNgForOfLet"
	ExtendedTemplateDiagnosticName_SUFFIX_NOT_SUPPORTED                       ExtendedTemplateDiagnosticName = "suffixNotSupported"
	ExtendedTemplateDiagnosticName_SKIP_HYDRATION_NOT_STATIC                  ExtendedTemplateDiagnosticName = "skipHydrationNotStatic"
	ExtendedTemplateDiagnosticName_INTERPOLATED_SIGNAL_NOT_INVOKED            ExtendedTemplateDiagnosticName = "interpolatedSignalNotInvoked"
	ExtendedTemplateDiagnosticName_CONTROL_FLOW_PREVENTING_CONTENT_PROJECTION ExtendedTemplateDiagnosticName = "controlFlowPreventingContentProjection"
	ExtendedTemplateDiagnosticName_UNUSED_LET_DECLARATION                     ExtendedTemplateDiagnosticName = "unusedLetDeclaration"
	ExtendedTemplateDiagnosticName_UNINVOKED_TRACK_FUNCTION                   ExtendedTemplateDiagnosticName = "uninvokedTrackFunction"
	ExtendedTemplateDiagnosticName_UNUSED_STANDALONE_IMPORTS                  ExtendedTemplateDiagnosticName = "unusedStandaloneImports"
	ExtendedTemplateDiagnosticName_UNPARENTHESIZED_NULLISH_COALESCING         ExtendedTemplateDiagnosticName = "unparenthesizedNullishCoalescing"
	ExtendedTemplateDiagnosticName_UNINVOKED_FUNCTION_IN_TEXT_INTERPOLATION   ExtendedTemplateDiagnosticName = "uninvokedFunctionInTextInterpolation"
	ExtendedTemplateDiagnosticName_DEFER_TRIGGER_MISCONFIGURATION             ExtendedTemplateDiagnosticName = "deferTriggerMisconfiguration"
)
