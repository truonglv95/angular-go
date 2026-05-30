package render3

// Port of angular/packages/compiler/src/render3/r3_class_debug_info_any

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// R3ClassDebugInfo holds info needed for runtime errors related to a class.
type R3ClassDebugInfo struct {
	// The class identifier
	Type output.Expression

	// A string literal containing the original class name as appears in its definition.
	ClassName output.Expression

	// A string literal containing the relative path of the file in which the class is defined.
	// Nil if the path failed to be computed.
	FilePath output.Expression // nil if not set

	// A number literal containing the line number in which this class is defined.
	LineNumber output.Expression

	// Whether to check if this component is being rendered without its NgModule being loaded.
	ForbidOrphanRendering bool
}

// CompileClassDebugInfo generates an ngDevMode guarded call to setClassDebugInfo.
func CompileClassDebugInfo(debugInfo R3ClassDebugInfo) output.Expression {
	debugInfoObj := map[string]output.Expression{
		"className": debugInfo.ClassName,
	}

	// Include file path and line number only if the file relative path is calculated successfully.
	if debugInfo.FilePath != nil {
		debugInfoObj["filePath"] = debugInfo.FilePath
		debugInfoObj["lineNumber"] = debugInfo.LineNumber
	}

	// Include forbidOrphanRendering only if it's set to true (to reduce generated code)
	if debugInfo.ForbidOrphanRendering {
		debugInfoObj["forbidOrphanRendering"] = LiteralExpr(true)
	}

	// Build ordered entries to maintain deterministic output.
	// Keys are added in the same order as in the TS source.
	var orderedKeys []string
	orderedKeys = append(orderedKeys, "className")
	if debugInfo.FilePath != nil {
		orderedKeys = append(orderedKeys, "filePath", "lineNumber")
	}
	if debugInfo.ForbidOrphanRendering {
		orderedKeys = append(orderedKeys, "forbidOrphanRendering")
	}

	entries := make([]output.LiteralMapEntry, len(orderedKeys))
	for i, key := range orderedKeys {
		entries[i] = output.NewLiteralMapPropertyAssignment(key, debugInfoObj[key], false)
	}
	mapExpr := output.NewLiteralMapExpr(entries, nil, nil, nil)

	fnCall := ImportExpr(*Identifiers.SetClassDebugInfo).
		CallFn([]output.Expression{debugInfo.Type, mapExpr}, nil, false, nil)
	iife := output.NewArrowFunctionExpr(
		nil,
		[]output.Statement{DevOnlyGuardedExpression(fnCall).ToStmt(nil)},
		nil,
		nil,
		nil,
	)
	return iife.CallFn(nil, nil, false, nil)
}
