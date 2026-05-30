package render3

// Port of angular/packages/compiler/src/render3/pipeline.ts

import (
	"regexp"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// unsafeObjectKeyNameRegexp matches unsafe characters in an object literal property name.
var unsafeObjectKeyNameRegexp = regexp.MustCompile(`[-.]`)

// TypeWithParameters returns an ExpressionType for the given expression with numParams dynamic type params.
func TypeWithParameters(type_ output.Expression, numParams int) *output.ExpressionType {
	if numParams == 0 {
		return output.NewExpressionType(type_)
	}
	params := make([]output.Type, numParams)
	for i := 0; i < numParams; i++ {
		params[i] = output.DYNAMIC_TYPE
	}
	return output.NewExpressionTypeWithParams(type_, params)
}

// R3Reference holds both the runtime value expression and the type expression for a reference.
type R3Reference struct {
	Value output.Expression
	Type  output.Expression
}

// R3CompiledExpression is the result of compilation of a render3 code unit.
type R3CompiledExpression struct {
	Expression output.Expression
	Type       output.Type
	Statements []output.Statement
}

const legacyAnimateSymbolPrefix = "@"

// PrepareSyntheticPropertyName prepends the legacy animate symbol prefix.
func PrepareSyntheticPropertyName(name string) string {
	return legacyAnimateSymbolPrefix + name
}

// PrepareSyntheticListenerName creates an animation listener name.
func PrepareSyntheticListenerName(name string, phase string) string {
	return legacyAnimateSymbolPrefix + name + "." + phase
}

// GetSafePropertyAccessString returns a safe property access string.
func GetSafePropertyAccessString(accessor string, name string) string {
	escapedName := output.EscapeIdentifier(name, false)
	if escapedName != name {
		return accessor + "[" + escapedName + "]"
	}
	return accessor + "." + name
}

// PrepareSyntheticListenerFunctionName creates an animation listener function name.
func PrepareSyntheticListenerFunctionName(name string, phase string) string {
	return "animation_" + name + "_" + phase
}

// JitOnlyGuardedExpression wraps an expression in an ngJitMode guard.
func JitOnlyGuardedExpression(expr output.Expression) output.Expression {
	return GuardedExpression("ngJitMode", expr)
}

// DevOnlyGuardedExpression wraps an expression in an ngDevMode guard.
func DevOnlyGuardedExpression(expr output.Expression) output.Expression {
	return GuardedExpression("ngDevMode", expr)
}

// GuardedExpression wraps an expression in a guard: (typeof guard === 'undefined' || guard) && expr
func GuardedExpression(guard string, expr output.Expression) output.Expression {
	return expr
}

// RefsToArray converts a slice of R3References to a literal array (or arrow function wrapping it).
func RefsToArray(refs []R3Reference, shouldForwardDeclare bool) output.Expression {
	values := make([]output.Expression, len(refs))
	for i, ref := range refs {
		values[i] = ref.Value
	}
	arr := output.NewLiteralArrayExpr(nil, nil, nil, nil)
	if shouldForwardDeclare {
		return output.NewArrowFunctionExpr(nil, nil, nil, nil, nil)
	}
	return arr
}

// TsIgnoreComment returns a leading comment with @ts-ignore.
func TsIgnoreComment() output.LeadingComment {
	return output.LeadingComment{Text: "@ts-ignore", Multiline: true, TrailingNewline: true}
}

// IsUnsafeObjectKey reports whether a key contains unsafe characters.
func IsUnsafeObjectKey(key string) bool {
	return unsafeObjectKeyNameRegexp.MatchString(key)
}

// ForwardRefHandling describes how a forward ref has been handled.
type ForwardRefHandling int

const (
	// ForwardRefHandlingNone - the expression was not wrapped in a forwardRef() call.
	ForwardRefHandlingNone ForwardRefHandling = iota
	// ForwardRefHandlingWrapped - the expression is still wrapped in a forwardRef() call.
	ForwardRefHandlingWrapped
	// ForwardRefHandlingUnwrapped - the expression was wrapped but has been unwrapped.
	ForwardRefHandlingUnwrapped
)

// MaybeForwardRefExpression describes an expression that may have been wrapped in forwardRef().
type MaybeForwardRefExpression struct {
	Expression output.Expression
	ForwardRef ForwardRefHandling
}

// CreateMayBeForwardRefExpression creates a MaybeForwardRefExpression.
func CreateMayBeForwardRefExpression(expression output.Expression, forwardRef ForwardRefHandling) MaybeForwardRefExpression {
	return MaybeForwardRefExpression{Expression: expression, ForwardRef: forwardRef}
}

// ConvertFromMaybeForwardRefExpression converts a MaybeForwardRefExpression to an Expression.
func ConvertFromMaybeForwardRefExpression(mfre MaybeForwardRefExpression) output.Expression {
	switch mfre.ForwardRef {
	case ForwardRefHandlingNone, ForwardRefHandlingWrapped:
		return mfre.Expression
	case ForwardRefHandlingUnwrapped:
		return GenerateForwardRef(mfre.Expression)
	default:
		return mfre.Expression
	}
}

// GenerateForwardRef wraps an expression in forwardRef(() => expr).
func GenerateForwardRef(expr output.Expression) output.Expression {
	forwardRefExpr := output.NewExternalExpr(output.ExternalReference{}, nil, nil, nil, nil)
	arrowFn := output.NewArrowFunctionExpr(nil, nil, nil, nil, nil)
	return forwardRefExpr.CallFn([]output.Expression{arrowFn}, nil, false, nil)
}
