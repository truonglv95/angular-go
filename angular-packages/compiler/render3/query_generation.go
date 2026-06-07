package render3

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

const CONTEXT_NAME = "ctx"
const RENDER_FLAGS = "rf"
const TEMPORARY_NAME = "_t"

func TemporaryAllocator(pushStatement func(output.Statement), name string) func() *output.ReadVarExpr {
	var temp *output.ReadVarExpr
	return func() *output.ReadVarExpr {
		if temp == nil {
			pushStatement(output.NewDeclareVarStmt(name, nil, output.DYNAMIC_TYPE, 0, nil, nil))
			temp = output.NewReadVarExpr(name, nil, nil, nil)
		}
		return temp
	}
}

func renderFlagCheckIfStmt(flags core.RenderFlags, statements []output.Statement) *output.IfStmt {
	return output.NewIfStmt(output.NewReadVarExpr(RENDER_FLAGS, nil, nil, nil).BitwiseAnd(LiteralExpr(flags), nil), statements, nil, nil, nil)
}

type QueryFlags int

const (
	QueryFlags_None                    QueryFlags = 0b0000
	QueryFlags_Descendants             QueryFlags = 0b0001
	QueryFlags_IsStatic                QueryFlags = 0b0010
	QueryFlags_EmitDistinctChangesOnly QueryFlags = 0b0100
)

func toQueryFlags(query R3QueryMetadata) int {
	flags := QueryFlags_None
	if query.Descendants {
		flags |= QueryFlags_Descendants
	}
	if query.Static {
		flags |= QueryFlags_IsStatic
	}
	if query.EmitDistinctChangesOnly {
		flags |= QueryFlags_EmitDistinctChangesOnly
	}
	return int(flags)
}

func GetQueryPredicate(query R3QueryMetadata, constantPool ConstantPool) output.Expression {
	if predicateArr, ok := query.Predicate.([]string); ok {
		var predicate []output.Expression
		for _, selector := range predicateArr {
			tokens := strings.Split(selector, ",")
			for _, token := range tokens {
				predicate = append(predicate, LiteralExpr(strings.TrimSpace(token)))
			}
		}
		if constantPool != nil {
			return constantPool.GetConstLiteral(LiteralArr(predicate), true)
		}
		return LiteralArr(predicate)
	} else if forwardRef, ok := query.Predicate.(MaybeForwardRefExpression); ok {
		switch forwardRef.ForwardRef {
		case ForwardRefHandlingNone, ForwardRefHandlingUnwrapped:
			return forwardRef.Expression
		case ForwardRefHandlingWrapped:
			return ImportExpr(*Identifiers.ResolveForwardRef).CallFn([]output.Expression{forwardRef.Expression}, nil, false, nil)
		}
	}
	return nil
}

func getQueryCreateParameters(query R3QueryMetadata, constantPool ConstantPool, prependParams []output.Expression) []output.Expression {
	parameters := []output.Expression{}
	if prependParams != nil {
		parameters = append(parameters, prependParams...)
	}
	if query.IsSignal {
		parameters = append(parameters, output.NewReadPropExpr(output.NewReadVarExpr(CONTEXT_NAME, nil, nil, nil), query.PropertyName, nil, nil, nil, false))
	}
	parameters = append(parameters, GetQueryPredicate(query, constantPool), LiteralExpr(toQueryFlags(query)))
	if query.Read != nil {
		parameters = append(parameters, query.Read.(output.Expression))
	}
	return parameters
}

var queryAdvancePlaceholder = &output.ExpressionStatement{}

func collapseAdvanceStatements(statements []output.Statement) []output.Statement {
	result := []output.Statement{}
	advanceCollapseCount := 0

	flushAdvanceCount := func() {
		if advanceCollapseCount > 0 {
			var args []output.Expression
			if advanceCollapseCount != 1 {
				args = []output.Expression{LiteralExpr(advanceCollapseCount)}
			}
			result = append([]output.Statement{ImportExpr(*Identifiers.QueryAdvance).CallFn(args, nil, false, nil).ToStmt(nil)}, result...)
			advanceCollapseCount = 0
		}
	}

	for i := len(statements) - 1; i >= 0; i-- {
		st := statements[i]
		if st == queryAdvancePlaceholder {
			advanceCollapseCount++
		} else {
			flushAdvanceCount()
			result = append([]output.Statement{st}, result...)
		}
	}
	flushAdvanceCount()
	return result
}

func CreateViewQueriesFunction(viewQueries []R3QueryMetadata, constantPool ConstantPool, name string) output.Expression {
	createStatements := []output.Statement{}
	updateStatements := []output.Statement{}
	tempAllocator := TemporaryAllocator(func(st output.Statement) {
		updateStatements = append(updateStatements, st)
	}, TEMPORARY_NAME)

	var viewQuerySignalCall output.Expression
	var viewQueryCall output.Expression

	for _, query := range viewQueries {
		params := getQueryCreateParameters(query, constantPool, nil)

		if query.IsSignal {
			if viewQuerySignalCall == nil {
				viewQuerySignalCall = ImportExpr(*Identifiers.ViewQuerySignal)
			}
			viewQuerySignalCall = viewQuerySignalCall.CallFn(params, nil, false, nil)
		} else {
			if viewQueryCall == nil {
				viewQueryCall = ImportExpr(*Identifiers.ViewQuery)
			}
			viewQueryCall = viewQueryCall.CallFn(params, nil, false, nil)
		}

		if query.IsSignal {
			updateStatements = append(updateStatements, queryAdvancePlaceholder)
			continue
		}

		temporary := tempAllocator()
		getQueryList := ImportExpr(*Identifiers.LoadQuery).CallFn(nil, nil, false, nil)
		refresh := ImportExpr(*Identifiers.QueryRefresh).CallFn([]output.Expression{temporary.Set(getQueryList)}, nil, false, nil)
		var tmpVal output.Expression = temporary
		if query.First {
			tmpVal = temporary.Prop("first", nil)
		}
		updateDirective := output.NewReadVarExpr(CONTEXT_NAME, nil, nil, nil).Prop(query.PropertyName, nil).Set(tmpVal)
		updateStatements = append(updateStatements, refresh.And(updateDirective, nil).ToStmt(nil))
	}

	if viewQuerySignalCall != nil {
		createStatements = append(createStatements, output.NewExpressionStatement(viewQuerySignalCall, nil, nil))
	}
	if viewQueryCall != nil {
		createStatements = append(createStatements, output.NewExpressionStatement(viewQueryCall, nil, nil))
	}

	var viewQueryFnName *string
	if name != "" {
		s := name + "_Query"
		viewQueryFnName = &s
	}

	return output.NewFunctionExpr(
		[]*output.FnParam{output.NewFnParam(RENDER_FLAGS, output.NUMBER_TYPE), output.NewFnParam(CONTEXT_NAME, output.DYNAMIC_TYPE)},
		[]output.Statement{
			renderFlagCheckIfStmt(core.RenderFlagsCreate, createStatements),
			renderFlagCheckIfStmt(core.RenderFlagsUpdate, collapseAdvanceStatements(updateStatements)),
		},
		output.INFERRED_TYPE,
		nil,
		viewQueryFnName,
		nil,
	)
}

func CreateContentQueriesFunction(queries []R3QueryMetadata, constantPool ConstantPool, name string) output.Expression {
	createStatements := []output.Statement{}
	updateStatements := []output.Statement{}
	tempAllocator := TemporaryAllocator(func(st output.Statement) {
		updateStatements = append(updateStatements, st)
	}, TEMPORARY_NAME)

	var contentQuerySignalCall output.Expression
	var contentQueryCall output.Expression

	for _, query := range queries {
		params := getQueryCreateParameters(query, constantPool, []output.Expression{output.NewReadVarExpr("dirIndex", nil, nil, nil)})

		if query.IsSignal {
			if contentQuerySignalCall == nil {
				contentQuerySignalCall = ImportExpr(*Identifiers.ContentQuerySignal)
			}
			contentQuerySignalCall = contentQuerySignalCall.CallFn(params, nil, false, nil)
		} else {
			if contentQueryCall == nil {
				contentQueryCall = ImportExpr(*Identifiers.ContentQuery)
			}
			contentQueryCall = contentQueryCall.CallFn(params, nil, false, nil)
		}

		if query.IsSignal {
			updateStatements = append(updateStatements, queryAdvancePlaceholder)
			continue
		}

		temporary := tempAllocator()
		getQueryList := ImportExpr(*Identifiers.LoadQuery).CallFn(nil, nil, false, nil)
		refresh := ImportExpr(*Identifiers.QueryRefresh).CallFn([]output.Expression{temporary.Set(getQueryList)}, nil, false, nil)
		var tmpVal output.Expression = temporary
		if query.First {
			tmpVal = temporary.Prop("first", nil)
		}
		updateDirective := output.NewReadVarExpr(CONTEXT_NAME, nil, nil, nil).Prop(query.PropertyName, nil).Set(tmpVal)
		updateStatements = append(updateStatements, refresh.And(updateDirective, nil).ToStmt(nil))
	}

	if contentQuerySignalCall != nil {
		createStatements = append(createStatements, output.NewExpressionStatement(contentQuerySignalCall, nil, nil))
	}
	if contentQueryCall != nil {
		createStatements = append(createStatements, output.NewExpressionStatement(contentQueryCall, nil, nil))
	}

	var contentQueriesFnName *string
	if name != "" {
		s := name + "_ContentQueries"
		contentQueriesFnName = &s
	}

	return output.NewFunctionExpr(
		[]*output.FnParam{output.NewFnParam(RENDER_FLAGS, output.NUMBER_TYPE), output.NewFnParam(CONTEXT_NAME, output.DYNAMIC_TYPE), output.NewFnParam("dirIndex", output.NUMBER_TYPE)},
		[]output.Statement{
			renderFlagCheckIfStmt(core.RenderFlagsCreate, createStatements),
			renderFlagCheckIfStmt(core.RenderFlagsUpdate, collapseAdvanceStatements(updateStatements)),
		},
		output.INFERRED_TYPE,
		nil,
		contentQueriesFnName,
		nil,
	)
}
