package partial_evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// evalScope tracks variable bindings for static evaluation.
type evalScope struct {
	bindings map[string]ResolvedValue
	parent   *evalScope
}

func newScope(parent *evalScope) *evalScope {
	return &evalScope{bindings: make(map[string]ResolvedValue), parent: parent}
}

func (s *evalScope) set(name string, val ResolvedValue) {
	s.bindings[name] = val
}

func (s *evalScope) get(name string) (ResolvedValue, bool) {
	if v, ok := s.bindings[name]; ok {
		return v, true
	}
	if s.parent != nil {
		return s.parent.get(name)
	}
	return nil, false
}

// PartialEvaluator performs static evaluation of TypeScript expressions.
type PartialEvaluator struct {
	host       reflection.ReflectionHost
	checker    any
	depTracker any
}

// NewPartialEvaluator creates a new PartialEvaluator.
func NewPartialEvaluator(host reflection.ReflectionHost, checker any, dependencyTracker any) *PartialEvaluator {
	return &PartialEvaluator{
		host:       host,
		checker:    checker,
		depTracker: dependencyTracker,
	}
}

// Evaluate statically evaluates an expression node and returns a ResolvedValue.
func (pe *PartialEvaluator) Evaluate(expr *ast.Node, foreignFunctionResolver any) ResolvedValue {
	if expr == nil {
		return nil
	}
	return pe.evaluateNode(expr, nil)
}

// EvaluateWithScope evaluates an expression with an initial scope.
func (pe *PartialEvaluator) EvaluateWithScope(expr *ast.Node, scope *evalScope) ResolvedValue {
	return pe.evaluateNode(expr, scope)
}

// evaluateNode recursively evaluates an AST node in the given scope.
func (pe *PartialEvaluator) evaluateNode(node *ast.Node, scope *evalScope) ResolvedValue {
	if node == nil {
		return nil
	}

	switch {
	case ast.IsObjectLiteralExpression(node):
		return pe.evaluateObjectLiteral(node.AsObjectLiteralExpression(), scope)

	case ast.IsArrayLiteralExpression(node):
		return pe.evaluateArrayLiteralWithSpread(node.AsArrayLiteralExpression(), scope)

	case ast.IsStringLiteral(node):
		return node.AsStringLiteral().Text

	case ast.IsNumericLiteral(node):
		text := node.AsNumericLiteral().Text
		if val, err := strconv.ParseFloat(text, 64); err == nil {
			return val
		}
		return text

	case isTrueKeyword(node):
		return true

	case isFalseKeyword(node):
		return false

	case isNullKeyword(node):
		return nil

	case ast.IsParenthesizedExpression(node):
		return pe.evaluateNode(node.AsParenthesizedExpression().Expression, scope)

	case ast.IsNoSubstitutionTemplateLiteral(node):
		return node.AsNoSubstitutionTemplateLiteral().Text

	case ast.IsTemplateExpression(node):
		return pe.evaluateTemplateExpression(node.AsTemplateExpression(), scope)

	case ast.IsIdentifier(node):
		return pe.evaluateIdentifier(node, scope)

	case ast.IsPropertyAccessExpression(node):
		return pe.evaluatePropertyAccess(node.AsPropertyAccessExpression(), scope)

	case ast.IsElementAccessExpression(node):
		return pe.evaluateElementAccess(node.AsElementAccessExpression(), scope)

	case ast.IsCallExpression(node):
		return pe.evaluateCallExpression(node.AsCallExpression(), scope)

	case ast.IsBinaryExpression(node):
		return pe.evaluateBinaryExpression(node.AsBinaryExpression(), scope)

	case ast.IsPrefixUnaryExpression(node):
		return pe.evaluatePrefixUnary(node.AsPrefixUnaryExpression(), scope)

	case ast.IsConditionalExpression(node):
		return pe.evaluateConditional(node.AsConditionalExpression(), scope)

	case ast.IsAsExpression(node):
		ae := node.AsAsExpression()
		return pe.evaluateNode(ae.Expression, scope)

	case ast.IsSpreadElement(node):
		// Evaluated inside array literal context; by itself it's dynamic
		return &DynamicValue{Node: node, Reason: "spread element"}

	default:
		return &DynamicValue{Node: node, Reason: "unrecognized expression"}
	}
}

// evaluateIdentifier resolves an identifier: checks scope, then walks source file declarations.
func (pe *PartialEvaluator) evaluateIdentifier(node *ast.Node, scope *evalScope) ResolvedValue {
	id := node.AsIdentifier()
	name := id.Text

	if name == "undefined" {
		return nil
	}

	// Check scope first
	if scope != nil {
		if val, ok := scope.get(name); ok {
			return val
		}
	}

	// Try using reflection host to find the declaration of the identifier (e.g. imported class)
	if pe.host != nil {
		decl := pe.host.GetDeclarationOfIdentifier(node)
		if decl != nil && decl.Node != nil {
			return decl.Node
		}
	}

	// Walk the source file to find a variable declaration with this name
	sf := getSourceFile(node)
	if sf != nil {
		if val, found := pe.findDeclarationValue(sf, name, scope); found {
			return val
		}
	}

	return &DynamicValue{Node: node, Reason: "identifier reference: " + name}
}

// getSourceFile walks up the parent chain to find the SourceFile node.
func getSourceFile(node *ast.Node) *ast.SourceFile {
	if node == nil {
		return nil
	}
	for n := node; n != nil; n = n.Parent {
		if ast.IsSourceFile(n) {
			return n.AsSourceFile()
		}
	}
	return nil
}

// findDeclarationValue searches for a variable/const/let declaration with the given name.
func (pe *PartialEvaluator) findDeclarationValue(sf *ast.SourceFile, name string, scope *evalScope) (ResolvedValue, bool) {
	var result ResolvedValue
	found := false

	var walk func(n *ast.Node) bool
	walk = func(n *ast.Node) bool {
		if found {
			return true
		}
		if ast.IsVariableDeclaration(n) {
			vd := n.AsVariableDeclaration()
			declName := vd.Name()
			if declName != nil && ast.IsIdentifier(declName) {
				if declName.AsIdentifier().Text == name {
					if vd.Initializer != nil {
						result = pe.evaluateNode(vd.Initializer, scope)
					}
					found = true
					return true
				}
			}
			// Handle destructuring patterns
			if declName != nil {
				if destructured, ok := pe.resolveDestructuredBinding(declName, name, vd.Initializer, scope); ok {
					result = destructured
					found = true
					return true
				}
			}
		}
		return n.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)
	return result, found
}

// resolveDestructuredBinding resolves a name from a destructuring pattern.
func (pe *PartialEvaluator) resolveDestructuredBinding(pattern *ast.Node, targetName string, initializer *ast.Node, scope *evalScope) (ResolvedValue, bool) {
	if pattern == nil || initializer == nil {
		return nil, false
	}
	initVal := pe.evaluateNode(initializer, scope)

	return pe.resolveBindingPattern(pattern, targetName, initVal, scope)
}

// resolveBindingPattern recursively resolves binding patterns.
func (pe *PartialEvaluator) resolveBindingPattern(pattern *ast.Node, targetName string, val ResolvedValue, scope *evalScope) (ResolvedValue, bool) {
	if pattern == nil {
		return nil, false
	}

	if ast.IsArrayBindingPattern(pattern) {
		arr, ok := val.(ResolvedValueArray)
		if !ok {
			return &DynamicValue{Node: pattern, Reason: "array destructuring non-array"}, false
		}
		bp := pattern.AsBindingPattern()
		if bp.Elements == nil {
			return nil, false
		}
		for i, elem := range bp.Elements.Nodes {
			if !ast.IsBindingElement(elem) {
				continue
			}
			be := elem.AsBindingElement()
			elemName := be.Name()
			if elemName == nil {
				continue
			}
			var elemVal ResolvedValue
			if i < len(arr) {
				elemVal = arr[i]
			} else {
				elemVal = nil
			}
			if ast.IsIdentifier(elemName) {
				if elemName.AsIdentifier().Text == targetName {
					return elemVal, true
				}
			} else {
				// Nested destructuring
				if v, ok := pe.resolveBindingPattern(elemName, targetName, elemVal, scope); ok {
					return v, true
				}
			}
		}
		return nil, false
	}

	if ast.IsObjectBindingPattern(pattern) {
		m, ok := val.(ResolvedValueMap)
		if !ok {
			return &DynamicValue{Node: pattern, Reason: "object destructuring non-map"}, false
		}
		bp := pattern.AsBindingPattern()
		if bp.Elements == nil {
			return nil, false
		}
		for _, elem := range bp.Elements.Nodes {
			if !ast.IsBindingElement(elem) {
				continue
			}
			be := elem.AsBindingElement()
			elemName := be.Name()
			if elemName == nil {
				continue
			}
			// Determine property key
			propKey := ""
			if be.PropertyName != nil {
				propKey = propertyNameToString(be.PropertyName)
			} else if ast.IsIdentifier(elemName) {
				propKey = elemName.AsIdentifier().Text
			}
			propVal := m[propKey]

			if ast.IsIdentifier(elemName) {
				if elemName.AsIdentifier().Text == targetName {
					return propVal, true
				}
			} else {
				// Nested destructuring
				if v, ok := pe.resolveBindingPattern(elemName, targetName, propVal, scope); ok {
					return v, true
				}
			}
		}
		return nil, false
	}

	return nil, false
}

// evaluatePropertyAccess evaluates object.property.
func (pe *PartialEvaluator) evaluatePropertyAccess(node *ast.PropertyAccessExpression, scope *evalScope) ResolvedValue {
	objVal := pe.evaluateNode(node.Expression, scope)
	if dv, ok := objVal.(*DynamicValue); ok {
		return &DynamicValue{Node: node.AsNode(), Reason: dv, fromDynamicInput: true}
	}

	propName := ""
	if node.Name() != nil {
		propName = propertyNameToString(node.Name())
	}

	if m, ok := objVal.(ResolvedValueMap); ok {
		if v, exists := m[propName]; exists {
			return v
		}
		return nil
	}

	// Array length
	if arr, ok := objVal.(ResolvedValueArray); ok {
		if propName == "length" {
			return float64(len(arr))
		}
	}

	// String properties
	if s, ok := objVal.(string); ok {
		if propName == "length" {
			return float64(len(s))
		}
	}

	return &DynamicValue{Node: node.AsNode(), Reason: fmt.Sprintf("property access .%s on non-map", propName)}
}

// evaluateElementAccess evaluates obj[key].
func (pe *PartialEvaluator) evaluateElementAccess(node *ast.ElementAccessExpression, scope *evalScope) ResolvedValue {
	objVal := pe.evaluateNode(node.Expression, scope)
	if dv, ok := objVal.(*DynamicValue); ok {
		return &DynamicValue{Node: node.AsNode(), Reason: dv, fromDynamicInput: true}
	}
	keyVal := pe.evaluateNode(node.ArgumentExpression, scope)

	// Handle array access
	if arr, ok := objVal.(ResolvedValueArray); ok {
		switch kv := keyVal.(type) {
		case float64:
			// Non-integer floats cannot be array indices
			if kv != math.Trunc(kv) {
				return &DynamicValue{Node: node.AsNode(), Reason: keyVal, fromInvalidExpressionType: true}
			}
			// Out-of-bounds (including negative) → undefined
			idx := int(kv)
			if idx < 0 || idx >= len(arr) {
				return nil
			}
			return arr[idx]
		case string:
			if kv == "length" {
				return float64(len(arr))
			}
			// slice()/concat() handled via call expression
			return &DynamicValue{Node: node.AsNode(), Reason: fmt.Sprintf("string key '%s' on array", kv)}
		default:
			return &DynamicValue{Node: node.AsNode(), Reason: keyVal, fromInvalidExpressionType: true}
		}
	}

	// Handle map access
	if m, ok := objVal.(ResolvedValueMap); ok {
		switch kv := keyVal.(type) {
		case string:
			if v, exists := m[kv]; exists {
				return v
			}
			return nil
		default:
			return &DynamicValue{Node: node.AsNode(), Reason: keyVal, fromInvalidExpressionType: true}
		}
	}

	// Handle string access
	if s, ok := objVal.(string); ok {
		switch kv := keyVal.(type) {
		case string:
			if kv == "length" {
				return float64(len(s))
			}
			// Methods like concat handled via call expression
			return &DynamicValue{Node: node.AsNode(), Reason: fmt.Sprintf("string key '%s' on string", kv)}
		case float64:
			idx := int(kv)
			if idx < 0 || idx >= len(s) {
				return nil
			}
			return string(s[idx])
		default:
			return &DynamicValue{Node: node.AsNode(), Reason: keyVal, fromInvalidExpressionType: true}
		}
	}

	return &DynamicValue{Node: node.AsNode(), Reason: fmt.Sprintf("element access on non-indexable: %T", objVal)}
}

// evaluateCallExpression handles function calls including array.slice(), array.concat(), str.concat().
func (pe *PartialEvaluator) evaluateCallExpression(node *ast.CallExpression, scope *evalScope) ResolvedValue {
	// Check if it's an element access call: obj['method'](args)
	if ast.IsElementAccessExpression(node.Expression) {
		eae := node.Expression.AsElementAccessExpression()
		objVal := pe.evaluateNode(eae.Expression, scope)
		methodKey := pe.evaluateNode(eae.ArgumentExpression, scope)
		methodName, isStr := methodKey.(string)
		if !isStr {
			return &DynamicValue{Node: node.AsNode(), Reason: "non-string method key"}
		}

		// Gather arguments
		args, dynamic := pe.evaluateArguments(node, scope)

		// Array methods
		if arr, ok := objVal.(ResolvedValueArray); ok {
			switch methodName {
			case "slice":
				return append(ResolvedValueArray{}, arr...)
			case "concat":
				if dynamic != nil {
					return &DynamicValue{Node: node.AsNode(), Reason: dynamic}
				}
				result := append(ResolvedValueArray{}, arr...)
				for _, a := range args {
					if inner, ok := a.(ResolvedValueArray); ok {
						result = append(result, inner...)
					} else {
						result = append(result, a)
					}
				}
				return result
			}
		}

		// String methods
		if s, ok := objVal.(string); ok {
			switch methodName {
			case "concat":
				if dynamic != nil {
					return &DynamicValue{Node: node.AsNode(), Reason: dynamic}
				}
				var sb strings.Builder
				sb.WriteString(s)
				for _, a := range args {
					sb.WriteString(resolvedValueToString(a))
				}
				return sb.String()
			}
		}

		return &DynamicValue{Node: node.AsNode(), Reason: fmt.Sprintf("unknown method '%s'", methodName)}
	}

	// Check if it's a property access call: obj.method(args)
	if ast.IsPropertyAccessExpression(node.Expression) {
		pae := node.Expression.AsPropertyAccessExpression()
		objVal := pe.evaluateNode(pae.Expression, scope)
		methodName := ""
		if pae.Name() != nil {
			methodName = propertyNameToString(pae.Name())
		}

		args, dynamic := pe.evaluateArguments(node, scope)

		// Array methods
		if arr, ok := objVal.(ResolvedValueArray); ok {
			switch methodName {
			case "slice":
				return append(ResolvedValueArray{}, arr...)
			case "concat":
				if dynamic != nil {
					return &DynamicValue{Node: node.AsNode(), Reason: dynamic}
				}
				result := append(ResolvedValueArray{}, arr...)
				for _, a := range args {
					if inner, ok := a.(ResolvedValueArray); ok {
						result = append(result, inner...)
					} else {
						result = append(result, a)
					}
				}
				return result
			}
		}

		// String concat
		if s, ok := objVal.(string); ok {
			switch methodName {
			case "concat":
				if dynamic != nil {
					return &DynamicValue{Node: node.AsNode(), Reason: dynamic}
				}
				var sb strings.Builder
				sb.WriteString(s)
				for _, a := range args {
					sb.WriteString(resolvedValueToString(a))
				}
				return sb.String()
			}
		}

		// Map method on resolved map (shouldn't happen in TS source but for completeness)
		if dv, ok := objVal.(*DynamicValue); ok {
			return &DynamicValue{Node: node.AsNode(), Reason: dv, fromDynamicInput: true}
		}

		return &DynamicValue{Node: node.AsNode(), Reason: fmt.Sprintf("unknown method call .%s()", methodName)}
	}

	// Direct function call: fn(args)
	if ast.IsIdentifier(node.Expression) {
		fnName := node.Expression.AsIdentifier().Text
		sf := getSourceFile(node.AsNode())
		if sf != nil {
			fn := pe.findFunctionDeclaration(sf, fnName)
			if fn != nil {
				return pe.evaluateFunctionCall(fn, node, scope)
			}
		}
	}

	return &DynamicValue{Node: node.AsNode(), Reason: "function call not statically evaluable"}
}

// evaluateArguments evaluates a call expression's arguments, handling spreads.
// Returns the evaluated args and any dynamic value encountered.
func (pe *PartialEvaluator) evaluateArguments(node *ast.CallExpression, scope *evalScope) ([]ResolvedValue, *DynamicValue) {
	if node.Arguments == nil {
		return nil, nil
	}
	var args []ResolvedValue
	for _, arg := range node.Arguments.Nodes {
		if ast.IsSpreadElement(arg) {
			se := arg.AsSpreadElement()
			spreadVal := pe.evaluateNode(se.Expression, scope)
			if arr, ok := spreadVal.(ResolvedValueArray); ok {
				args = append(args, arr...)
			} else if dv, ok := spreadVal.(*DynamicValue); ok {
				return nil, dv
			} else {
				args = append(args, spreadVal)
			}
		} else {
			val := pe.evaluateNode(arg, scope)
			args = append(args, val)
		}
	}
	return args, nil
}

// findFunctionDeclaration finds a function declaration by name in the source file.
func (pe *PartialEvaluator) findFunctionDeclaration(sf *ast.SourceFile, name string) *ast.FunctionDeclaration {
	var result *ast.FunctionDeclaration
	var walk func(n *ast.Node) bool
	walk = func(n *ast.Node) bool {
		if result != nil {
			return true
		}
		if ast.IsFunctionDeclaration(n) {
			fd := n.AsFunctionDeclaration()
			if fd.Name() != nil && ast.IsIdentifier(fd.Name()) {
				if fd.Name().AsIdentifier().Text == name {
					result = fd
					return true
				}
			}
		}
		return n.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)
	return result
}

// evaluateFunctionCall evaluates a simple function call (single return statement).
func (pe *PartialEvaluator) evaluateFunctionCall(fn *ast.FunctionDeclaration, call *ast.CallExpression, scope *evalScope) ResolvedValue {
	if fn.Body == nil {
		return &DynamicValue{Node: call.AsNode(), Reason: "function has no body"}
	}

	// Count statements to check for "complex" functions
	body := fn.Body
	if !ast.IsBlock(body) {
		// Expression body - evaluate directly in function scope
		fnScope := newScope(scope)
		pe.bindFunctionArgs(fn, call, fnScope, scope)
		return pe.evaluateNode(body, fnScope)
	}

	block := body.AsBlock()
	if block.Statements == nil {
		return nil
	}
	stmts := block.Statements.Nodes
	if len(stmts) != 1 {
		// "complex" function - multiple statements
		return &DynamicValue{Node: call.AsNode(), Reason: fn.AsNode(), fromComplexFunctionCall: true}
	}

	stmt := stmts[0]
	if !ast.IsReturnStatement(stmt) {
		return &DynamicValue{Node: call.AsNode(), Reason: fn.AsNode(), fromComplexFunctionCall: true}
	}

	rs := stmt.AsReturnStatement()
	if rs.Expression == nil {
		return nil
	}

	// Build function scope with argument bindings
	fnScope := newScope(scope)
	pe.bindFunctionArgs(fn, call, fnScope, scope)
	return pe.evaluateNode(rs.Expression, fnScope)
}

// bindFunctionArgs binds the function parameters to the call arguments in the scope.
func (pe *PartialEvaluator) bindFunctionArgs(fn *ast.FunctionDeclaration, call *ast.CallExpression, fnScope *evalScope, callScope *evalScope) {
	if fn.Parameters == nil {
		return
	}
	params := fn.Parameters.Nodes
	var callArgs []*ast.Node
	if call.Arguments != nil {
		callArgs = call.Arguments.Nodes
	}

	argIdx := 0
	for _, param := range params {
		pd := param.AsParameterDeclaration()
		paramName := pd.Name()
		if paramName == nil {
			argIdx++
			continue
		}

		isRest := pd.DotDotDotToken != nil

		if isRest {
			// Collect remaining args into an array
			var restArr ResolvedValueArray
			for argIdx < len(callArgs) {
				arg := callArgs[argIdx]
				if ast.IsSpreadElement(arg) {
					se := arg.AsSpreadElement()
					spreadVal := pe.evaluateNode(se.Expression, callScope)
					if arr, ok := spreadVal.(ResolvedValueArray); ok {
						restArr = append(restArr, arr...)
					} else {
						restArr = append(restArr, spreadVal)
					}
				} else {
					restArr = append(restArr, pe.evaluateNode(arg, callScope))
				}
				argIdx++
			}
			if ast.IsIdentifier(paramName) {
				fnScope.set(paramName.AsIdentifier().Text, restArr)
			}
			break
		}

		var argVal ResolvedValue
		if argIdx < len(callArgs) {
			arg := callArgs[argIdx]
			if ast.IsSpreadElement(arg) {
				se := arg.AsSpreadElement()
				spreadVal := pe.evaluateNode(se.Expression, callScope)
				if arr, ok := spreadVal.(ResolvedValueArray); ok && len(arr) > 0 {
					argVal = arr[0]
				}
			} else {
				argVal = pe.evaluateNode(arg, callScope)
			}
			argIdx++
		} else if pd.Initializer != nil {
			// Use default value
			argVal = pe.evaluateNode(pd.Initializer, fnScope)
		}

		if ast.IsIdentifier(paramName) {
			fnScope.set(paramName.AsIdentifier().Text, argVal)
		}
	}
}

// evaluateBinaryExpression evaluates a binary expression.
func (pe *PartialEvaluator) evaluateBinaryExpression(node *ast.BinaryExpression, scope *evalScope) ResolvedValue {
	op := node.OperatorToken.Kind

	// Short-circuit for && and ||
	switch op {
	case ast.KindAmpersandAmpersandToken:
		left := pe.evaluateNode(node.Left, scope)
		if isTruthy(left) {
			return pe.evaluateNode(node.Right, scope)
		}
		return left

	case ast.KindBarBarToken:
		left := pe.evaluateNode(node.Left, scope)
		if isTruthy(left) {
			return left
		}
		return pe.evaluateNode(node.Right, scope)
	}

	left := pe.evaluateNode(node.Left, scope)
	right := pe.evaluateNode(node.Right, scope)

	// If left is dynamic, propagate with reason
	if dv, ok := left.(*DynamicValue); ok {
		return &DynamicValue{Node: node.AsNode(), Reason: dv, fromDynamicInput: true}
	}
	if dv, ok := right.(*DynamicValue); ok {
		return &DynamicValue{Node: node.AsNode(), Reason: dv, fromDynamicInput: true}
	}

	switch op {
	case ast.KindPlusToken:
		// String concatenation or number addition
		_, lIsStr := left.(string)
		_, rIsStr := right.(string)
		lNum, lIsNum := toNumber(left)
		rNum, rIsNum := toNumber(right)

		if lIsStr || rIsStr {
			ls := resolvedValueToString(left)
			rs := resolvedValueToString(right)
			return ls + rs
		}
		if lIsNum && rIsNum {
			return lNum + rNum
		}
		// Invalid types for +
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindMinusToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return lNum - rNum
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindAsteriskToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return lNum * rNum
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindSlashToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return lNum / rNum
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindPercentToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return math.Mod(lNum, rNum)
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindAmpersandToken:
		lInt, lOk := toInt32(left)
		rInt, rOk := toInt32(right)
		if lOk && rOk {
			return float64(lInt & rInt)
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindBarToken:
		lInt, lOk := toInt32(left)
		rInt, rOk := toInt32(right)
		if lOk && rOk {
			return float64(lInt | rInt)
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindCaretToken:
		lInt, lOk := toInt32(left)
		rInt, rOk := toInt32(right)
		if lOk && rOk {
			return float64(lInt ^ rInt)
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindAsteriskAsteriskToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return math.Pow(lNum, rNum)
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindLessThanLessThanToken:
		lInt, lOk := toInt32(left)
		rInt, rOk := toInt32(right)
		if lOk && rOk {
			return float64(lInt << uint(rInt&0x1f))
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindGreaterThanGreaterThanToken:
		lInt, lOk := toInt32(left)
		rInt, rOk := toInt32(right)
		if lOk && rOk {
			return float64(lInt >> uint(rInt&0x1f))
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	case ast.KindGreaterThanGreaterThanGreaterThanToken:
		lInt, lOk := toUint32(left)
		rInt, rOk := toUint32(right)
		if lOk && rOk {
			return float64(lInt >> (rInt & 0x1f))
		}
		reason := &DynamicValue{Node: node.Left, Reason: left, fromInvalidExpressionType: true}
		return &DynamicValue{Node: node.AsNode(), Reason: reason}

	// Comparison operators
	case ast.KindLessThanToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return lNum < rNum
		}
		lStr, lIsStr := left.(string)
		rStr, rIsStr := right.(string)
		if lIsStr && rIsStr {
			return lStr < rStr
		}
		return &DynamicValue{Node: node.AsNode(), Reason: "comparison on non-numbers"}

	case ast.KindLessThanEqualsToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return lNum <= rNum
		}
		lStr, lIsStr := left.(string)
		rStr, rIsStr := right.(string)
		if lIsStr && rIsStr {
			return lStr <= rStr
		}
		return &DynamicValue{Node: node.AsNode(), Reason: "comparison on non-numbers"}

	case ast.KindGreaterThanToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return lNum > rNum
		}
		lStr, lIsStr := left.(string)
		rStr, rIsStr := right.(string)
		if lIsStr && rIsStr {
			return lStr > rStr
		}
		return &DynamicValue{Node: node.AsNode(), Reason: "comparison on non-numbers"}

	case ast.KindGreaterThanEqualsToken:
		lNum, lOk := toNumber(left)
		rNum, rOk := toNumber(right)
		if lOk && rOk {
			return lNum >= rNum
		}
		lStr, lIsStr := left.(string)
		rStr, rIsStr := right.(string)
		if lIsStr && rIsStr {
			return lStr >= rStr
		}
		return &DynamicValue{Node: node.AsNode(), Reason: "comparison on non-numbers"}

	case ast.KindEqualsEqualsEqualsToken:
		return strictEqual(left, right)

	case ast.KindExclamationEqualsEqualsToken:
		return !strictEqual(left, right)

	case ast.KindEqualsEqualsToken:
		return looseEqual(left, right)

	case ast.KindExclamationEqualsToken:
		return !looseEqual(left, right)

	default:
		return &DynamicValue{Node: node.AsNode(), Reason: "unsupported binary operator", fromUnsupportedSyntax: true}
	}
}

// evaluatePrefixUnary evaluates prefix unary expressions.
func (pe *PartialEvaluator) evaluatePrefixUnary(pue *ast.PrefixUnaryExpression, scope *evalScope) ResolvedValue {
	switch pue.Operator {
	case ast.KindMinusToken:
		operand := pe.evaluateNode(pue.Operand, scope)
		if n, ok := toNumber(operand); ok {
			return -n
		}
		return &DynamicValue{Node: pue.AsNode(), Reason: "prefix minus on non-number"}

	case ast.KindPlusToken:
		operand := pe.evaluateNode(pue.Operand, scope)
		if n, ok := toNumber(operand); ok {
			return n
		}
		return &DynamicValue{Node: pue.AsNode(), Reason: "prefix plus on non-number"}

	case ast.KindExclamationToken:
		operand := pe.evaluateNode(pue.Operand, scope)
		if dv, ok := operand.(*DynamicValue); ok {
			return &DynamicValue{Node: pue.AsNode(), Reason: dv}
		}
		return !isTruthy(operand)

	case ast.KindTildeToken:
		operand := pe.evaluateNode(pue.Operand, scope)
		if n, ok := toInt32(operand); ok {
			return float64(^n)
		}
		return &DynamicValue{Node: pue.AsNode(), Reason: "bitwise NOT on non-integer"}

	default:
		// PlusPlus, MinusMinus etc.
		return &DynamicValue{Node: pue.AsNode(), Reason: "unsupported unary operator", fromUnsupportedSyntax: true}
	}
}

// evaluateConditional evaluates a ternary conditional expression.
func (pe *PartialEvaluator) evaluateConditional(node *ast.ConditionalExpression, scope *evalScope) ResolvedValue {
	cond := pe.evaluateNode(node.Condition, scope)
	if dv, ok := cond.(*DynamicValue); ok {
		return &DynamicValue{Node: node.AsNode(), Reason: dv}
	}
	if isTruthy(cond) {
		return pe.evaluateNode(node.WhenTrue, scope)
	}
	return pe.evaluateNode(node.WhenFalse, scope)
}

// evaluateTemplateExpression evaluates a template literal expression.
func (pe *PartialEvaluator) evaluateTemplateExpression(node *ast.TemplateExpression, scope *evalScope) ResolvedValue {
	var sb strings.Builder
	if node.Head != nil {
		sb.WriteString(node.Head.AsTemplateHead().Text)
	}
	if node.TemplateSpans != nil {
		for _, span := range node.TemplateSpans.Nodes {
			ts := span.AsTemplateSpan()
			val := pe.evaluateNode(ts.Expression, scope)
			if dv, ok := val.(*DynamicValue); ok {
				return &DynamicValue{Node: node.AsNode(), Reason: dv, fromDynamicInput: true}
			}
			sb.WriteString(resolvedValueToString(val))
			if ts.Literal != nil {
				// TemplateMiddle or TemplateTail
				if ast.IsTemplateMiddle(ts.Literal) {
					sb.WriteString(ts.Literal.AsTemplateMiddle().Text)
				} else if ast.IsTemplateTail(ts.Literal) {
					sb.WriteString(ts.Literal.AsTemplateTail().Text)
				}
			}
		}
	}
	return sb.String()
}

// evaluateObjectLiteral evaluates an object literal to a ResolvedValueMap.
func (pe *PartialEvaluator) evaluateObjectLiteral(node *ast.ObjectLiteralExpression, scope *evalScope) ResolvedValue {
	result := make(ResolvedValueMap)
	if node.Properties == nil {
		return result
	}
	for _, prop := range node.Properties.Nodes {
		switch {
		case ast.IsPropertyAssignment(prop):
			pa := prop.AsPropertyAssignment()
			name := propertyNameToString(pa.Name())
			if name == "" {
				continue
			}
			val := pe.evaluateNode(pa.Initializer, scope)
			result[name] = val
		case ast.IsShorthandPropertyAssignment(prop):
			spa := prop.AsShorthandPropertyAssignment()
			name := spa.Name().AsIdentifier().Text
			val := pe.evaluateNode(spa.Name(), scope)
			result[name] = val
		case ast.IsSpreadAssignment(prop):
			spreadNode := prop.AsSpreadAssignment()
			spreadVal := pe.evaluateNode(spreadNode.Expression, scope)
			if _, ok := spreadVal.(*DynamicValue); ok {
				dvProp := &DynamicValue{Node: prop, Reason: spreadVal, fromInvalidExpressionType: true}
				return &DynamicValue{Node: prop, Reason: dvProp, fromDynamicInput: true}
			}
			if m, ok := spreadVal.(ResolvedValueMap); ok {
				for k, v := range m {
					result[k] = v
				}
			} else {
				// Non-map spread is also invalid
				dvProp := &DynamicValue{Node: prop, Reason: spreadVal, fromInvalidExpressionType: true}
				return &DynamicValue{Node: prop, Reason: dvProp, fromDynamicInput: true}
			}
		}
	}
	return result
}

// evaluateArrayLiteralWithSpread handles spread elements properly.
func (pe *PartialEvaluator) evaluateArrayLiteralWithSpread(node *ast.ArrayLiteralExpression, scope *evalScope) ResolvedValue {
	var result ResolvedValueArray
	if node.Elements == nil {
		return result
	}
	for _, elem := range node.Elements.Nodes {
		if ast.IsSpreadElement(elem) {
			se := elem.AsSpreadElement()
			spreadVal := pe.evaluateNode(se.Expression, scope)
			if arr, ok := spreadVal.(ResolvedValueArray); ok {
				result = append(result, arr...)
			} else if dv, ok := spreadVal.(*DynamicValue); ok {
				// Append a dynamic value representing this invalid spread
				badSpread := &DynamicValue{Node: elem, Reason: dv.Reason, fromInvalidExpressionType: true}
				result = append(result, badSpread)
			} else {
				// Non-array, non-dynamic spread - mark as invalid
				badSpread := &DynamicValue{Node: elem, Reason: spreadVal, fromInvalidExpressionType: true}
				result = append(result, badSpread)
			}
		} else {
			val := pe.evaluateNode(elem, scope)
			result = append(result, val)
		}
	}
	return result
}

// propertyNameToString converts a property name node to a string.
func propertyNameToString(node *ast.Node) string {
	if node == nil {
		return ""
	}
	switch {
	case ast.IsIdentifier(node):
		return node.AsIdentifier().Text
	case ast.IsStringLiteral(node):
		return node.AsStringLiteral().Text
	case ast.IsNumericLiteral(node):
		return node.AsNumericLiteral().Text
	case ast.IsPrivateIdentifier(node):
		return node.AsPrivateIdentifier().Text
	}
	return ""
}

// isTrueKeyword checks if the node is the `true` keyword.
func isTrueKeyword(node *ast.Node) bool {
	return node.Kind == ast.KindTrueKeyword
}

// isFalseKeyword checks if the node is the `false` keyword.
func isFalseKeyword(node *ast.Node) bool {
	return node.Kind == ast.KindFalseKeyword
}

// isNullKeyword checks if the node is the `null` keyword.
func isNullKeyword(node *ast.Node) bool {
	return node.Kind == ast.KindNullKeyword
}

// isTruthy returns whether a ResolvedValue is truthy in JavaScript semantics.
func isTruthy(val ResolvedValue) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v != 0 && !math.IsNaN(v)
	case string:
		return v != ""
	case ResolvedValueArray:
		return true // arrays are truthy
	case ResolvedValueMap:
		return true // objects are truthy
	case *DynamicValue:
		return false // treat unknown as falsy conservatively? Actually shouldn't matter since we check for dynamic before calling isTruthy
	default:
		return true
	}
}

// toNumber tries to convert a ResolvedValue to a float64.
func toNumber(val ResolvedValue) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// toInt32 converts a value to a signed 32-bit integer (JavaScript semantics).
func toInt32(val ResolvedValue) (int32, bool) {
	n, ok := toNumber(val)
	if !ok {
		return 0, false
	}
	return int32(n), true
}

// toUint32 converts a value to an unsigned 32-bit integer (JavaScript semantics).
func toUint32(val ResolvedValue) (uint32, bool) {
	n, ok := toNumber(val)
	if !ok {
		return 0, false
	}
	return uint32(int32(n)), true
}

// resolvedValueToString converts a ResolvedValue to its string representation.
func resolvedValueToString(val ResolvedValue) string {
	if val == nil {
		return "null"
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == math.Trunc(v) && !math.IsInf(v, 0) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", val)
}

// strictEqual implements JavaScript === semantics.
func strictEqual(left, right ResolvedValue) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	switch l := left.(type) {
	case float64:
		if r, ok := right.(float64); ok {
			return l == r
		}
		return false
	case string:
		if r, ok := right.(string); ok {
			return l == r
		}
		return false
	case bool:
		if r, ok := right.(bool); ok {
			return l == r
		}
		return false
	}
	return false
}

// looseEqual implements JavaScript == semantics (simplified).
func looseEqual(left, right ResolvedValue) bool {
	if strictEqual(left, right) {
		return true
	}
	// Number to string coercion
	lStr := resolvedValueToString(left)
	rStr := resolvedValueToString(right)
	return lStr == rStr
}
