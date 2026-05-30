package render3

// Port of angular/packages/compiler/src/render3/r3_module_any

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// R3SelectorScopeMode describes how the selector scope of an NgModule should be emitted.
type R3SelectorScopeMode int

const (
	// R3SelectorScopeModeInline emits the declarations inline into the module definition.
	R3SelectorScopeModeInline R3SelectorScopeMode = iota
	// R3SelectorScopeModeSideEffect emits the declarations using a side-effectful function call.
	R3SelectorScopeModeSideEffect
	// R3SelectorScopeModeOmit does not generate selector scopes at all.
	R3SelectorScopeModeOmit
)

// R3NgModuleMetadataKind describes the type of NgModule metadata.
type R3NgModuleMetadataKind int

const (
	// R3NgModuleMetadataKindGlobal is used for full and partial compilation modes.
	R3NgModuleMetadataKindGlobal R3NgModuleMetadataKind = iota
	// R3NgModuleMetadataKindLocal is used for the local compilation mode.
	R3NgModuleMetadataKindLocal
	// R3NgModuleMetadataKindIsolated is used for isolated declarations mode.
	R3NgModuleMetadataKindIsolated
)

// R3NgModuleMetadataCommon holds common fields for all NgModule metadata kinds.
type R3NgModuleMetadataCommon struct {
	Kind              R3NgModuleMetadataKind
	Type              R3Reference
	SelectorScopeMode R3SelectorScopeMode
	Schemas           []R3Reference     // nil if none
	Id                output.Expression // nil if not set
}

// R3NgModuleMetadataGlobal holds metadata for full/partial mode NgModule pipeline.
type R3NgModuleMetadataGlobal struct {
	R3NgModuleMetadataCommon
	Bootstrap              []R3Reference
	Declarations           []R3Reference
	PublicDeclarationTypes []output.Expression // nil if all declarations are public
	Imports                []R3Reference
	IncludeImportTypes     bool
	Exports                []R3Reference
	ContainsForwardDecls   bool
}

// R3NgModuleMetadataLocal holds metadata for local compilation mode NgModule pipeline.
type R3NgModuleMetadataLocal struct {
	R3NgModuleMetadataCommon
	BootstrapExpression    output.Expression // nil if not set
	DeclarationsExpression output.Expression // nil if not set
	ImportsExpression      output.Expression // nil if not set
	ExportsExpression      output.Expression // nil if not set
}

// R3NgModuleMetadataIsolated holds metadata for isolated declarations mode NgModule pipeline.
type R3NgModuleMetadataIsolated struct {
	R3NgModuleMetadataCommon
	ImportsExpression output.Expression // nil if not set
	ExportsExpression output.Expression // nil if not set
}

// R3NgModuleMetadata is the union type for NgModule metadata.
type R3NgModuleMetadata interface {
	getCommon() R3NgModuleMetadataCommon
}

func (m *R3NgModuleMetadataGlobal) getCommon() R3NgModuleMetadataCommon {
	return m.R3NgModuleMetadataCommon
}
func (m *R3NgModuleMetadataLocal) getCommon() R3NgModuleMetadataCommon {
	return m.R3NgModuleMetadataCommon
}
func (m *R3NgModuleMetadataIsolated) getCommon() R3NgModuleMetadataCommon {
	return m.R3NgModuleMetadataCommon
}

// CompileNgModule constructs an R3NgModuleDef for the given R3NgModuleMetadata.
func CompileNgModule(meta R3NgModuleMetadata) R3CompiledExpression {
	var statements []output.Statement
	definitionMap := NewDefinitionMap()
	common := meta.getCommon()
	definitionMap.Set("type", common.Type.Value)

	switch m := meta.(type) {
	case *R3NgModuleMetadataGlobal:
		// Assign bootstrap definition.
		if len(m.Bootstrap) > 0 {
			definitionMap.Set("bootstrap", RefsToArray(m.Bootstrap, m.ContainsForwardDecls))
		}

		if m.SelectorScopeMode == R3SelectorScopeModeInline {
			if len(m.Declarations) > 0 {
				definitionMap.Set("declarations", RefsToArray(m.Declarations, m.ContainsForwardDecls))
			}
			if len(m.Imports) > 0 {
				definitionMap.Set("imports", RefsToArray(m.Imports, m.ContainsForwardDecls))
			}
			if len(m.Exports) > 0 {
				definitionMap.Set("exports", RefsToArray(m.Exports, m.ContainsForwardDecls))
			}
		} else if m.SelectorScopeMode == R3SelectorScopeModeSideEffect {
			setNgModuleScopeCall := generateSetNgModuleScopeCall(meta)
			if setNgModuleScopeCall != nil {
				statements = append(statements, setNgModuleScopeCall)
			}
		}
		// Omit: do nothing

	case *R3NgModuleMetadataLocal:
		if m.SelectorScopeMode == R3SelectorScopeModeSideEffect {
			setNgModuleScopeCall := generateSetNgModuleScopeCall(meta)
			if setNgModuleScopeCall != nil {
				statements = append(statements, setNgModuleScopeCall)
			}
		}

	case *R3NgModuleMetadataIsolated:
		// Omit: isolated mode always omits scope in JS definition.
	}

	if len(common.Schemas) > 0 {
		schemaVals := make([]output.Expression, len(common.Schemas))
		for i, ref := range common.Schemas {
			schemaVals[i] = ref.Value
		}
		definitionMap.Set("schemas", LiteralArr(schemaVals))
	}

	if common.Id != nil {
		definitionMap.Set("id", common.Id)
		// Generate a side-effectful call to register this NgModule by its id.
		statements = append(statements,
			ImportExpr(*Identifiers.RegisterNgModuleType).
				CallFn([]output.Expression{common.Type.Value, common.Id}, nil, false, nil).
				ToStmt(nil),
		)
	}

	expression := ImportExpr(*Identifiers.DefineNgModule).
		CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, true, nil)
	type_ := CreateNgModuleType(meta)

	return R3CompiledExpression{Expression: expression, Type: type_, Statements: statements}
}

// CompileNgModuleDeclarationExpression generates the call to ɵɵdefineNgModule() from ɵɵngDeclareNgModule().
// Used in JIT mode.
func CompileNgModuleDeclarationExpression(meta R3NgModuleDeclarationFacade) output.Expression {
	definitionMap := NewDefinitionMap()
	definitionMap.Set("type", output.NewLiteralExpr(meta.Type, nil, nil, nil))
	if meta.Bootstrap != nil {
		definitionMap.Set("bootstrap", output.NewLiteralExpr(meta.Bootstrap, nil, nil, nil))
	}
	if meta.Declarations != nil {
		definitionMap.Set("declarations", output.NewLiteralExpr(meta.Declarations, nil, nil, nil))
	}
	if meta.Imports != nil {
		definitionMap.Set("imports", output.NewLiteralExpr(meta.Imports, nil, nil, nil))
	}
	if meta.Exports != nil {
		definitionMap.Set("exports", output.NewLiteralExpr(meta.Exports, nil, nil, nil))
	}
	if meta.Schemas != nil {
		definitionMap.Set("schemas", output.NewLiteralExpr(meta.Schemas, nil, nil, nil))
	}
	if meta.Id != nil {
		definitionMap.Set("id", output.NewLiteralExpr(meta.Id, nil, nil, nil))
	}
	return ImportExpr(*Identifiers.DefineNgModule).
		CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
}

// R3NgModuleDeclarationFacade corresponds to R3DeclareNgModuleFacade from compiler_facade_interface.ts.
type R3NgModuleDeclarationFacade struct {
	Type         interface{}
	Bootstrap    interface{} // nil if undefined
	Declarations interface{} // nil if undefined
	Imports      interface{} // nil if undefined
	Exports      interface{} // nil if undefined
	Schemas      interface{} // nil if undefined
	Id           interface{} // nil if undefined
}

// CreateNgModuleType creates the type expression for an NgModule.
func CreateNgModuleType(meta R3NgModuleMetadata) *output.ExpressionType {
	switch m := meta.(type) {
	case *R3NgModuleMetadataLocal:
		return output.NewExpressionType(m.Type.Value)

	case *R3NgModuleMetadataIsolated:
		var importsType output.Type = output.NONE_TYPE
		if m.ImportsExpression != nil {
			importsType = output.NewExpressionType(m.ImportsExpression)
		}
		var exportsType output.Type = output.NONE_TYPE
		if m.ExportsExpression != nil {
			exportsType = output.NewExpressionType(m.ExportsExpression)
		}
		return output.NewExpressionType(
			output.NewExternalExpr(
				*Identifiers.NgModuleDeclaration,
				nil,
				[]output.Type{
					output.NewExpressionType(m.Type.Type),
					output.NONE_TYPE,
					importsType,
					exportsType,
				},
				nil,
				nil,
			),
		)

	case *R3NgModuleMetadataGlobal:
		var declarationsType output.Type
		if m.PublicDeclarationTypes == nil {
			declarationsType = tupleTypeOf(m.Declarations)
		} else {
			declarationsType = tupleOfTypes(m.PublicDeclarationTypes)
		}
		var importsType output.Type = output.NONE_TYPE
		if m.IncludeImportTypes {
			importsType = tupleTypeOf(m.Imports)
		}
		return output.NewExpressionType(
			output.NewExternalExpr(
				*Identifiers.NgModuleDeclaration,
				nil,
				[]output.Type{
					output.NewExpressionType(m.Type.Type),
					declarationsType,
					importsType,
					tupleTypeOf(m.Exports),
				},
				nil,
				nil,
			),
		)
	}
	return output.NewExpressionType(meta.getCommon().Type.Value)
}

// generateSetNgModuleScopeCall generates the ɵɵsetNgModuleScope call.
func generateSetNgModuleScopeCall(meta R3NgModuleMetadata) output.Statement {
	common := meta.getCommon()
	scopeMap := NewDefinitionMap()

	switch m := meta.(type) {
	case *R3NgModuleMetadataGlobal:
		if len(m.Declarations) > 0 {
			scopeMap.Set("declarations", RefsToArray(m.Declarations, m.ContainsForwardDecls))
		}
		if len(m.Imports) > 0 {
			scopeMap.Set("imports", RefsToArray(m.Imports, m.ContainsForwardDecls))
		}
		if len(m.Exports) > 0 {
			scopeMap.Set("exports", RefsToArray(m.Exports, m.ContainsForwardDecls))
		}

	case *R3NgModuleMetadataLocal:
		if m.DeclarationsExpression != nil {
			scopeMap.Set("declarations", m.DeclarationsExpression)
		}
		if m.ImportsExpression != nil {
			scopeMap.Set("imports", m.ImportsExpression)
		}
		if m.ExportsExpression != nil {
			scopeMap.Set("exports", m.ExportsExpression)
		}
		if m.BootstrapExpression != nil {
			scopeMap.Set("bootstrap", m.BootstrapExpression)
		}
	}

	if len(scopeMap.Values) == 0 {
		return nil
	}

	// setNgModuleScope(...)
	fnCall := output.NewInvokeFunctionExpr(
		ImportExpr(*Identifiers.SetNgModuleScope),
		[]output.Expression{common.Type.Value, scopeMap.ToLiteralMap()},
		nil,
		nil,
		false,
		nil,
		false,
	)

	// (ngJitMode guard) && setNgModuleScope(...)
	guardedCall := JitOnlyGuardedExpression(fnCall)

	// function() { (ngJitMode guard) && setNgModuleScope(...); }
	iife := output.NewFunctionExpr(nil, []output.Statement{guardedCall.ToStmt(nil)}, nil, nil, nil, nil)

	// (function() { ... })()
	iifeCall := output.NewInvokeFunctionExpr(iife, nil, nil, nil, false, nil, false)

	return iifeCall.ToStmt(nil)
}

// tupleTypeOf creates a tuple type from R3References.
func tupleTypeOf(exp []R3Reference) output.Type {
	types := make([]output.Expression, len(exp))
	for i, ref := range exp {
		types[i] = output.NewTypeofExpr(ref.Type, nil, nil, nil)
	}
	if len(exp) > 0 {
		return output.NewExpressionType(LiteralArr(types))
	}
	return output.NONE_TYPE
}

// tupleOfTypes creates a tuple type from expressions.
func tupleOfTypes(types []output.Expression) output.Type {
	typeofTypes := make([]output.Expression, len(types))
	for i, t := range types {
		typeofTypes[i] = output.NewTypeofExpr(t, nil, nil, nil)
	}
	if len(types) > 0 {
		return output.NewExpressionType(LiteralArr(typeofTypes))
	}
	return output.NONE_TYPE
}
