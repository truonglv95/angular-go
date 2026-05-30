package diagnostics

var COMPILER_ERRORS_WITH_GUIDES = map[ErrorCode]bool{
	ErrorCode_DECORATOR_ARG_NOT_LITERAL:             true,
	ErrorCode_IMPORT_CYCLE_DETECTED:                 true,
	ErrorCode_PARAM_MISSING_TOKEN:                   true,
	ErrorCode_SCHEMA_INVALID_ELEMENT:                true,
	ErrorCode_SCHEMA_INVALID_ATTRIBUTE:              true,
	ErrorCode_MISSING_REFERENCE_TARGET:              true,
	ErrorCode_COMPONENT_INVALID_SHADOW_DOM_SELECTOR: true,
	ErrorCode_WARN_NGMODULE_ID_UNNECESSARY:          true,
}
