package output

import (
	"fmt"
	"strings"
)

type ExternalReferenceResolver interface {
	ResolveExternalReference(ref *ExternalReference) any
}

type JitEvaluator struct{}

func NewJitEvaluator() *JitEvaluator {
	return &JitEvaluator{}
}

func (e *JitEvaluator) EvaluateStatements(
	sourceUrl string,
	statements []Statement,
	refResolver ExternalReferenceResolver,
	createSourceMaps bool,
) map[string]any {
	converter := NewJitEmitterVisitor(refResolver)
	ctx := NewEmitterVisitorContext(0)

	if len(statements) > 0 && !isUseStrictStatement(statements[0]) {
		statements = append([]Statement{NewLiteralExpr("use strict", nil, nil, nil).ToStmt(nil)}, statements...)
	}

	converter.VisitAllStatements(statements, ctx)
	converter.CreateReturnStmt(ctx)
	return e.EvaluateCode(sourceUrl, ctx, converter.GetArgs(), createSourceMaps).(map[string]any)
}

func (e *JitEvaluator) EvaluateCode(
	sourceUrl string,
	ctx *EmitterVisitorContext,
	vars map[string]any,
	createSourceMap bool,
) any {
	fnBody := fmt.Sprintf("\"use strict\";%s\n//# sourceURL=%s", ctx.ToSource(), sourceUrl)
	var fnArgNames []string
	var fnArgValues []any
	for argName, argValue := range vars {
		fnArgValues = append(fnArgValues, argValue)
		fnArgNames = append(fnArgNames, argName)
	}

	if createSourceMap {
		// Cannot get .toString() easily in Go func
		emptyFnStr := "function anonymous() { return null; }"
		headerLines := strings.Count(emptyFnStr[:strings.Index(emptyFnStr, "return null;")], "\n") - 1
		if headerLines < 0 {
			headerLines = 0
		}
		fnBody += fmt.Sprintf("\n%s", ctx.ToSourceMapGenerator(sourceUrl, headerLines).ToJsComment())
	}

	argsForFn := append(fnArgNames, fnBody)
	fn := NewTrustedFunctionForJIT(argsForFn...)
	return e.ExecuteFunction(fn, fnArgValues)
}

func (e *JitEvaluator) ExecuteFunction(fn func(any) any, args []any) any {
	// Dummy execution since we can't eval JS in Go
	return fn(args)
}

type JitEmitterVisitor struct {
	AbstractJsEmitterVisitor
	refResolver      ExternalReferenceResolver
	evalArgNames     []string
	evalArgValues    []any
	evalExportedVars []string
}

func NewJitEmitterVisitor(refResolver ExternalReferenceResolver) *JitEmitterVisitor {
	v := &JitEmitterVisitor{
		refResolver: refResolver,
	}
	v.AbstractJsEmitterVisitor.Init()
	return v
}

func (v *JitEmitterVisitor) CreateReturnStmt(ctx *EmitterVisitorContext) {
	var props []LiteralMapEntry
	for _, resultVar := range v.evalExportedVars {
		props = append(props, &LiteralMapPropertyAssignment{
			Key:    resultVar,
			Value:  NewReadVarExpr(resultVar, nil, nil, nil),
			Quoted: false,
		})
	}
	stmt := NewReturnStatement(NewLiteralMapExpr(props, nil, nil, nil), nil, nil)
	stmt.VisitStatement(v, ctx)
}

func (v *JitEmitterVisitor) GetArgs() map[string]any {
	result := make(map[string]any)
	for i, name := range v.evalArgNames {
		result[name] = v.evalArgValues[i]
	}
	return result
}

func (v *JitEmitterVisitor) VisitExternalExpr(ast *ExternalExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.emitReferenceToExternal(ast, v.refResolver.ResolveExternalReference((&ast.Value)), ctx)
	return nil
}

func (v *JitEmitterVisitor) VisitWrappedNodeExpr(ast *WrappedNodeExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.emitReferenceToExternal(ast, ast.Node, ctx)
	return nil
}

func (v *JitEmitterVisitor) VisitDeclareVarStmt(stmt *DeclareVarStmt, context any) any {
	ctx := context.(*EmitterVisitorContext)
	if stmt.HasModifier(StmtModifierExported) {
		v.evalExportedVars = append(v.evalExportedVars, stmt.Name)
	}
	return v.AbstractJsEmitterVisitor.VisitDeclareVarStmt(stmt, ctx)
}

func (v *JitEmitterVisitor) VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context any) any {
	ctx := context.(*EmitterVisitorContext)
	if stmt.HasModifier(StmtModifierExported) {
		v.evalExportedVars = append(v.evalExportedVars, stmt.Name)
	}
	return v.AbstractJsEmitterVisitor.VisitDeclareFunctionStmt(stmt, ctx)
}

func (v *JitEmitterVisitor) emitReferenceToExternal(ast Expression, value any, ctx *EmitterVisitorContext) {
	id := -1
	for i, val := range v.evalArgValues {
		if val == value { // Note: Simple equality check might not be enough depending on 'value' type in Go
			id = i
			break
		}
	}
	if id == -1 {
		id = len(v.evalArgValues)
		v.evalArgValues = append(v.evalArgValues, value)
		// For simplicity, we just use a generic name.
		name := "val"
		v.evalArgNames = append(v.evalArgNames, fmt.Sprintf("jit_%s_%d", name, id))
	}
	ctx.Print(ast, v.evalArgNames[id], false)
}

func isUseStrictStatement(statement Statement) bool {
	return statement.IsEquivalent(NewLiteralExpr("use strict", nil, nil, nil).ToStmt(nil))
}
