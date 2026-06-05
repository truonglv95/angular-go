package render3

// Port of angular/packages/compiler/src/render3/r3_hmr_any

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// R3HmrMetadata holds metadata necessary to compile HMR-related code.
type R3HmrMetadata struct {
	// Component class for which HMR is being enabled.
	Type output.Expression

	// Name of the component class.
	ClassName string

	// File path of the component class.
	FilePath string

	// Namespace dependencies for HMR update functions.
	NamespaceDependencies []R3HmrNamespaceDependency

	// Local dependencies passed as function parameters.
	LocalDependencies []R3HmrLocalDependency
}

// R3HmrNamespaceDependency describes an HMR dependency on a namespace import.
type R3HmrNamespaceDependency struct {
	// Module name of the import.
	ModuleName string

	// Name under which to refer to the namespace inside HMR-related code.
	AssignedName string
}

// R3HmrLocalDependency describes a local dependency passed as a function parameter.
type R3HmrLocalDependency struct {
	Name                  string
	RuntimeRepresentation output.Expression
}

// Helper equivalent to JavaScript's encodeURIComponent
func encodeURIComponent(str string) string {
	var result []byte
	for i := 0; i < len(str); i++ {
		c := str[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '!' || c == '~' || c == '*' || c == '\'' || c == '(' || c == ')' {
			result = append(result, c)
		} else {
			result = append(result, '%')
			result = append(result, fmt.Sprintf("%02X", c)...)
		}
	}
	return string(result)
}

// CompileHmrInitializer compiles the expression that initializes HMR for a class.
func CompileHmrInitializer(meta R3HmrMetadata) output.Expression {
	moduleName := "m"
	dataName := "d"
	timestampName := "t"
	idName := "id"
	importCallbackName := fmt.Sprintf("%s_HmrLoad", meta.ClassName)

	namespaces := make([]output.Expression, len(meta.NamespaceDependencies))
	for i, dep := range meta.NamespaceDependencies {
		modName := dep.ModuleName
		namespaces[i] = output.NewExternalExpr(output.ExternalReference{ModuleName: &modName, Name: nil}, nil, nil, nil, nil)
	}

	// m.default
	defaultRead := VariableExpr(moduleName).Prop("default", nil)

	// ɵɵreplaceMetadata(Comp, m.default, [...namespaces], [...locals], import.meta, id);
	localDeps := make([]output.Expression, len(meta.LocalDependencies))
	for i, l := range meta.LocalDependencies {
		localDeps[i] = l.RuntimeRepresentation
	}
	replaceCall := ImportExpr(*Identifiers.ReplaceMetadata).
		CallFn([]output.Expression{
			meta.Type,
			defaultRead,
			LiteralArr(namespaces),
			LiteralArr(localDeps),
			VariableExpr("import").Prop("meta", nil),
			VariableExpr(idName),
		}, nil, false, nil)

	// (m) => m.default && ɵɵreplaceMetadata(...)
	replaceCallback := output.NewArrowFunctionExpr(
		[]*output.FnParam{output.NewFnParam(moduleName, output.DYNAMIC_TYPE)},
		defaultRead.And(replaceCall, nil),
		nil,
		nil,
		nil,
	)

	// getReplaceMetadataURL(id, timestamp, import.meta.url)
	urlExpr := ImportExpr(*Identifiers.GetReplaceMetadataURL).
		CallFn([]output.Expression{
			VariableExpr(idName),
			VariableExpr(timestampName),
			VariableExpr("import").Prop("meta", nil).Prop("url", nil),
		}, nil, false, nil)

	// function Cmp_HmrLoad(t) { import(/* @vite-ignore */ url).then((m) => ...); }
	viteIgnore := "@vite-ignore"
	importCallback := output.NewDeclareFunctionStmt(
		importCallbackName,
		[]*output.FnParam{output.NewFnParam(timestampName, output.DYNAMIC_TYPE)},
		[]output.Statement{
			output.NewDynamicImportExpr(urlExpr, nil, &viteIgnore, nil).
				Prop("then", nil).
				CallFn([]output.Expression{replaceCallback}, nil, false, nil).
				ToStmt(nil),
		},
		nil,
		output.StmtModifierFinal,
		nil,
		nil,
	)

	// (d) => d.id === id && Cmp_HmrLoad(d.timestamp)
	updateCallback := output.NewArrowFunctionExpr(
		[]*output.FnParam{output.NewFnParam(dataName, output.DYNAMIC_TYPE)},
		VariableExpr(dataName).Prop("id", nil).
			Identical(VariableExpr(idName), nil).
			And(
				VariableExpr(importCallbackName).CallFn(
					[]output.Expression{VariableExpr(dataName).Prop("timestamp", nil)},
					nil, false, nil,
				),
				nil,
			),
		nil,
		nil,
		nil,
	)

	// import.meta.hot
	hotRead := VariableExpr("import").Prop("meta", nil).Prop("hot", nil)

	// import.meta.hot.on('angular:component-update', () => ...);
	hotListener := hotRead.Clone().(*output.ReadPropExpr).
		Prop("on", nil).
		CallFn([]output.Expression{LiteralExpr("angular:component-update"), updateCallback}, nil, false, nil)

	// The encoded component ID
	componentId := encodeURIComponent(fmt.Sprintf("%s@%s", meta.FilePath, meta.ClassName))

	// Wrap everything in an IIFE:
	// (() => {
	//   const id = <encoded-id>;
	//   function Cmp_HmrLoad(t) {...}
	//   ngDevMode && import.meta.hot && import.meta.hot.on(...);
	// })()
	body := []output.Statement{
		// const id = <id>;
		output.NewDeclareVarStmt(
			idName,
			LiteralExpr(componentId),
			nil,
			output.StmtModifierFinal,
			nil,
			nil,
		),
		// function Cmp_HmrLoad() {...}
		importCallback,
		// ngDevMode && import.meta.hot && import.meta.hot.on(...)
		DevOnlyGuardedExpression(hotRead.And(hotListener, nil)).ToStmt(nil),
	}

	iife := output.NewArrowFunctionExpr(nil, body, nil, nil, nil)
	return iife.CallFn(nil, nil, false, nil)
}

// CompileHmrUpdateCallback compiles the HMR update callback for a class.
func CompileHmrUpdateCallback(
	definitions []R3HmrDefinitionField,
	constantStatements []output.Statement,
	meta R3HmrMetadata,
) *output.DeclareFunctionStmt {
	namespaces := "ɵɵnamespaces"
	params := []*output.FnParam{
		output.NewFnParam(meta.ClassName, output.DYNAMIC_TYPE),
		output.NewFnParam(namespaces, output.DYNAMIC_TYPE),
	}
	var body []output.Statement

	for _, local := range meta.LocalDependencies {
		params = append(params, output.NewFnParam(local.Name, output.DYNAMIC_TYPE))
	}

	// Declare variables that read out the individual namespaces.
	for i, ns := range meta.NamespaceDependencies {
		body = append(body, output.NewDeclareVarStmt(
			ns.AssignedName,
			VariableExpr(namespaces).Key(LiteralExpr(i), nil, nil),
			output.DYNAMIC_TYPE,
			output.StmtModifierFinal,
			nil,
			nil,
		))
	}

	body = append(body, constantStatements...)

	for _, field := range definitions {
		if field.Initializer != nil {
			body = append(body,
				VariableExpr(meta.ClassName).Prop(field.Name, nil).
					Set(field.Initializer).ToStmt(nil),
			)
			body = append(body, field.Statements...)
		}
	}

	fnName := meta.ClassName + "_UpdateMetadata"
	return output.NewDeclareFunctionStmt(
		fnName,
		params,
		body,
		nil,
		output.StmtModifierFinal,
		nil,
		nil,
	)
}

// R3HmrDefinitionField represents a compiled definition field for HMR updates.
type R3HmrDefinitionField struct {
	Name        string
	Initializer output.Expression // nil if no initializer
	Statements  []output.Statement
}
