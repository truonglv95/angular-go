package compiler

// Port of packages/compiler/src/constant_pool.ts

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

const constantPrefix = "_c"

// poolInclusionLengthThresholdForStrings is the length threshold above which
// string literals are eligible for the constant pool.
const poolInclusionLengthThresholdForStrings = 50

// keyContext is the sentinel value used to distinguish key-generation visits
// from normal expression visits (mirrors KEY_CONTEXT = {} in TypeScript).
var keyContext = &struct{}{}

// FixupExpression is a placeholder Expression that can be replaced once the
// actual shared constant variable is known.
// Mirrors the private class FixupExpression extends o.Expression in TS.
type FixupExpression struct {
	output.BaseExpression
	Resolved output.Expression // the current resolved expression (variable ref once shared)
	original output.Expression // the original constant expression
	Shared   bool
}

func newFixupExpression(resolved output.Expression) *FixupExpression {
	f := &FixupExpression{
		Resolved: resolved,
		original: resolved,
		Shared:   false,
	}
	f.Type = resolved.GetType()
	f.Self = f
	return f
}

func (f *FixupExpression) VisitExpression(visitor output.ExpressionVisitor, context any) any {
	if context == keyContext {
		// When producing a key traverse the original constant, not the variable.
		return f.original.VisitExpression(visitor, context)
	}
	return f.Resolved.VisitExpression(visitor, context)
}

func (f *FixupExpression) IsEquivalent(e output.Expression) bool {
	other, ok := e.(*FixupExpression)
	return ok && f.Resolved.IsEquivalent(other.Resolved)
}

func (f *FixupExpression) IsConstant() bool { return true }

func (f *FixupExpression) Clone() output.Expression {
	panic("FixupExpression.Clone() is not supported")
}

func (f *FixupExpression) fixup(expression output.Expression) {
	f.Resolved = expression
	f.Shared = true
}

// ConstantPool allows a code emitter to share constants in an output context.
// Mirrors export class ConstantPool in constant_pool.ts.
type ConstantPool struct {
	Statements []output.Statement

	literals         map[string]*FixupExpression
	literalFactories map[string]output.Expression
	sharedConstants  map[string]output.Expression
	claimedNames     map[string]int

	isClosureCompilerEnabled bool
}

// NewConstantPool creates a new ConstantPool.
// Mirrors: constructor(private readonly isClosureCompilerEnabled: boolean = false)
func NewConstantPool(isClosureCompilerEnabled bool) *ConstantPool {
	return &ConstantPool{
		Statements:               []output.Statement{},
		literals:                 make(map[string]*FixupExpression),
		literalFactories:         make(map[string]output.Expression),
		sharedConstants:          make(map[string]output.Expression),
		claimedNames:             make(map[string]int),
		isClosureCompilerEnabled: isClosureCompilerEnabled,
	}
}

// GetConstLiteral returns (or creates) a shared constant for the given literal.
// Mirrors: getConstLiteral(literal: o.Expression, forceShared?: boolean): o.Expression
func (cp *ConstantPool) GetConstLiteral(literal output.Expression, forceShared bool) output.Expression {
	// Do not pool simple literals or existing fixups.
	if le, ok := literal.(*output.LiteralExpr); ok && !isLongStringLiteral(le) {
		return literal
	}
	if _, ok := literal.(*FixupExpression); ok {
		return literal
	}

	key := GenericKeyFnInstance.KeyOf(literal)
	fixup, exists := cp.literals[key]
	newValue := false
	if !exists {
		fixup = newFixupExpression(literal)
		cp.literals[key] = fixup
		newValue = true
	}

	if (!newValue && !fixup.Shared) || (newValue && forceShared) {
		// Replace the expression with a variable reference.
		name := cp.freshName()
		var value output.Expression
		var usage output.Expression

		if cp.isClosureCompilerEnabled && isLongStringLiteral(literal) {
			// Wrap in a function to prevent Closure from inlining large strings.
			// const myStr = function() { return "very long string"; };
			// const usage = myStr();
			value = output.NewFunctionExpr(
				nil, // params
				[]output.Statement{
					output.NewReturnStatement(literal, nil, nil),
				},
				nil, nil, nil, nil,
			)
			readVar := output.NewReadVarExpr(name, nil, nil, nil)
			usage = readVar.CallFn(nil, nil, false, nil)
		} else {
			value = literal
			usage = output.NewReadVarExpr(name, nil, nil, nil)
		}

		cp.Statements = append(cp.Statements,
			output.NewDeclareVarStmt(name, value, output.INFERRED_TYPE, output.StmtModifierFinal, nil, nil),
		)
		fixup.fixup(usage)
	}

	return fixup
}

// GetSharedConstant returns (or creates) a shared constant defined by def.
// Mirrors: getSharedConstant(def: SharedConstantDefinition, expr: o.Expression): o.Expression
func (cp *ConstantPool) GetSharedConstant(def SharedConstantDefinition, expr output.Expression) output.Expression {
	key := def.KeyOf(expr)
	if _, ok := cp.sharedConstants[key]; !ok {
		id := cp.freshName()
		cp.sharedConstants[key] = output.NewReadVarExpr(id, nil, nil, nil)
		cp.Statements = append(cp.Statements, def.ToSharedConstantDeclaration(id, expr))
	}
	return cp.sharedConstants[key]
}

// GetSharedFunctionReference returns (or creates) a shared function/arrow reference.
// Mirrors: getSharedFunctionReference(fn, prefix, useUniqueName = true): o.Expression
func (cp *ConstantPool) GetSharedFunctionReference(fn output.Expression, prefix string, useUniqueName bool) output.Expression {
	_, isArrow := fn.(*output.ArrowFunctionExpr)

	for _, current := range cp.Statements {
		if isArrow {
			// Arrow functions are saved as DeclareVarStmt; compare the value.
			if dvs, ok := current.(*output.DeclareVarStmt); ok && dvs.Value != nil && dvs.Value.IsEquivalent(fn) {
				return output.NewReadVarExpr(dvs.Name, nil, nil, nil)
			}
		} else {
			// FunctionExpr instances are saved as DeclareFunctionStmt.
			// FunctionExpr.IsEquivalent accepts an Expression, but DeclareFunctionStmt is
			// a Statement. In TypeScript fn.isEquivalent(current) works because
			// FunctionExpr.isEquivalent also handles DeclareFunctionStmt via duck-typing.
			// In Go we replicate by comparing params + statements directly.
			if dfs, ok := current.(*output.DeclareFunctionStmt); ok {
				if fe, ok2 := fn.(*output.FunctionExpr); ok2 {
					// Compare by re-checking via a temporary FunctionExpr wrapping dfs fields.
					tmp := output.NewFunctionExpr(dfs.Params, dfs.Statements, dfs.Type, nil, nil, nil)
					if fe.IsEquivalent(tmp) {
						return output.NewReadVarExpr(dfs.Name, nil, nil, nil)
					}
				}
			}
		}
	}

	// Otherwise declare the function.
	var name string
	if useUniqueName {
		name = cp.UniqueName(prefix, true)
	} else {
		name = prefix
	}

	var stmt output.Statement
	if fe, ok := fn.(*output.FunctionExpr); ok {
		stmt = fe.ToDeclStmt(name, output.StmtModifierFinal)
	} else {
		var sourceSpan output.ParseSourceSpan
		if sp, ok := fn.GetSourceSpan().(output.ParseSourceSpan); ok {
			sourceSpan = sp
		}
		stmt = output.NewDeclareVarStmt(name, fn, output.INFERRED_TYPE, output.StmtModifierFinal, sourceSpan, nil)
	}
	cp.Statements = append(cp.Statements, stmt)
	return output.NewReadVarExpr(name, nil, nil, nil)
}

// UniqueName produces a unique name in the context of this pool.
// Mirrors: uniqueName(name: string, alwaysIncludeSuffix = true): string
func (cp *ConstantPool) UniqueName(name string, alwaysIncludeSuffix bool) string {
	count := cp.claimedNames[name]
	var result string
	if count == 0 && !alwaysIncludeSuffix {
		result = name
	} else {
		result = fmt.Sprintf("%s%d", name, count)
	}
	cp.claimedNames[name] = count + 1
	return result
}

// freshName returns the next auto-generated constant name (_c0, _c1, …).
func (cp *ConstantPool) freshName() string {
	return cp.UniqueName(constantPrefix, true)
}

// ---- ExpressionKeyFn / SharedConstantDefinition interfaces ----

// ExpressionKeyFn mirrors export interface ExpressionKeyFn in TS.
type ExpressionKeyFn interface {
	KeyOf(expr output.Expression) string
}

// SharedConstantDefinition mirrors export interface SharedConstantDefinition in TS.
type SharedConstantDefinition interface {
	ExpressionKeyFn
	ToSharedConstantDeclaration(declName string, keyExpr output.Expression) output.Statement
}

// ---- GenericKeyFn ----

// GenericKeyFn mirrors export class GenericKeyFn implements ExpressionKeyFn.
type GenericKeyFn struct{}

// GenericKeyFnInstance is the singleton instance (mirrors GenericKeyFn.INSTANCE).
var GenericKeyFnInstance = &GenericKeyFn{}

// KeyOf produces a stable string key for an output expression.
// Mirrors: keyOf(expr: o.Expression): string
func (g *GenericKeyFn) KeyOf(expr output.Expression) string {
	switch e := expr.(type) {
	case *output.LiteralExpr:
		switch v := e.Value.(type) {
		case string:
			return `"` + v + `"`
		case nil:
			return "null"
		default:
			return fmt.Sprintf("%v", v)
		}

	case *output.RegularExpressionLiteralExpr:
		flags := ""
		if e.Flags != nil {
			flags = *e.Flags
		}
		return fmt.Sprintf("/%s/%s", e.Body, flags)

	case *output.LiteralArrayExpr:
		parts := make([]string, len(e.Entries))
		for i, entry := range e.Entries {
			parts[i] = g.KeyOf(entry)
		}
		return "[" + strings.Join(parts, ",") + "]"

	case *output.LiteralMapExpr:
		parts := make([]string, len(e.Entries))
		for i, entry := range e.Entries {
			switch en := entry.(type) {
			case *output.LiteralMapSpreadAssignment:
				parts[i] = "..." + g.KeyOf(en.Expression)
			case *output.LiteralMapPropertyAssignment:
				key := en.Key
				if en.Quoted {
					key = `"` + key + `"`
				}
				parts[i] = key + ":" + g.KeyOf(en.Value)
			}
		}
		return "{" + strings.Join(parts, ",") + "}"

	case *output.ExternalExpr:
		modName := ""
		if e.Value.ModuleName != nil {
			modName = *e.Value.ModuleName
		}
		name := ""
		if e.Value.Name != nil {
			name = *e.Value.Name
		}
		return fmt.Sprintf(`import("%s", %s)`, modName, name)

	case *output.ReadVarExpr:
		return fmt.Sprintf("read(%s)", e.Name)

	case *output.TypeofExpr:
		return fmt.Sprintf("typeof(%s)", g.KeyOf(e.Expr))

	case *output.SpreadElementExpr:
		return "..." + g.KeyOf(e.Expression)

	default:
		panic(fmt.Sprintf("GenericKeyFn does not handle expressions of type %T", expr))
	}
}

// ---- helpers ----

// isLongStringLiteral reports whether expr is a string literal at or above the
// pool inclusion threshold. Mirrors: function isLongStringLiteral(expr)
func isLongStringLiteral(expr output.Expression) bool {
	if le, ok := expr.(*output.LiteralExpr); ok {
		if s, ok2 := le.Value.(string); ok2 {
			return len(s) >= poolInclusionLengthThresholdForStrings
		}
	}
	return false
}
