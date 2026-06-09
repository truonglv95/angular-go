package partial_evaluator

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	ngtscdiag "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/imports"
	"github.com/microsoft/typescript-go/internal/ast"
)

// undefinedType is a sentinel representing JavaScript's undefined.
type undefinedType struct{}

// Undefined is the sentinel instance representing JavaScript's undefined.
var Undefined = undefinedType{}

// DescribeResolvedType derives a type representation from a resolved value to be reported in a diagnostic.
func DescribeResolvedType(value ResolvedValue, maxDepth int) string {
	if maxDepth < 0 {
		maxDepth = 0
	}
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case undefinedType:
		return "undefined"
	case float64, int, int64, int32:
		return "number"
	case bool:
		return "boolean"
	case string:
		return "string"
	case ResolvedValueMap:
		if maxDepth == 0 {
			return "object"
		}
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var entries []string
		for _, key := range keys {
			entries = append(entries, fmt.Sprintf("%s: %s", quoteKey(key), DescribeResolvedType(v[key], maxDepth-1)))
		}
		if len(entries) > 0 {
			return "{ " + strings.Join(entries, "; ") + " }"
		}
		return "{}"
	case map[string]ResolvedValue:
		if maxDepth == 0 {
			return "object"
		}
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var entries []string
		for _, key := range keys {
			entries = append(entries, fmt.Sprintf("%s: %s", quoteKey(key), DescribeResolvedType(v[key], maxDepth-1)))
		}
		if len(entries) > 0 {
			return "{ " + strings.Join(entries, "; ") + " }"
		}
		return "{}"
	case *ResolvedModule:
		return "(module)"
	case *EnumValue:
		name := ""
		if v.EnumRef != nil {
			name = getNodeDebugName(v.EnumRef)
		}
		if name == "" {
			name = "(anonymous)"
		}
		return name
	case *imports.Reference:
		name := v.DebugName()
		if name == "" {
			name = "(anonymous)"
		}
		return name
	case ResolvedValueArray:
		if maxDepth == 0 {
			return "Array"
		}
		var elems []string
		for _, elem := range v {
			elems = append(elems, DescribeResolvedType(elem, maxDepth-1))
		}
		return "[" + strings.Join(elems, ", ") + "]"
	case []ResolvedValue:
		if maxDepth == 0 {
			return "Array"
		}
		var elems []string
		for _, elem := range v {
			elems = append(elems, DescribeResolvedType(elem, maxDepth-1))
		}
		return "[" + strings.Join(elems, ", ") + "]"
	case *DynamicValue:
		return "(not statically analyzable)"
	case KnownFn:
		return "Function"
	default:
		return "unknown"
	}
}

func quoteKey(key string) string {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", key)
	if matched {
		return key
	}
	return "'" + strings.ReplaceAll(key, "'", "\\'") + "'"
}

func getNodeDebugName(node *ast.Node) string {
	if node == nil {
		return ""
	}
	if ast.IsIdentifier(node) {
		return node.AsIdentifier().Text
	}
	if ast.IsClassDeclaration(node) {
		cd := node.AsClassDeclaration()
		if cd.Name() != nil && ast.IsIdentifier(cd.Name()) {
			return cd.Name().AsIdentifier().Text
		}
	}
	if ast.IsFunctionDeclaration(node) {
		fd := node.AsFunctionDeclaration()
		if fd.Name() != nil && ast.IsIdentifier(fd.Name()) {
			return fd.Name().AsIdentifier().Text
		}
	}
	if ast.IsEnumDeclaration(node) {
		ed := node.AsEnumDeclaration()
		if ed.Name() != nil && ast.IsIdentifier(ed.Name()) {
			return ed.Name().AsIdentifier().Text
		}
	}
	return ""
}

// TraceDynamicValue creates an array of related information diagnostics for a DynamicValue that describe the trace
// of why an expression was evaluated as dynamic.
func TraceDynamicValue(node *ast.Node, value *DynamicValue) []*ast.Diagnostic {
	visitor := &traceDynamicValueVisitor{
		originNode: node,
	}
	return visitor.trace(value)
}

type traceDynamicValueVisitor struct {
	originNode           *ast.Node
	currentContainerNode *ast.Node
}

func (v *traceDynamicValueVisitor) shouldTrace(node *ast.Node) bool {
	if node == v.originNode {
		return false
	}
	container := getContainerNode(node)
	if container == v.currentContainerNode {
		return false
	}
	v.currentContainerNode = container
	return true
}

func getContainerNode(node *ast.Node) *ast.Node {
	for curr := node; curr != nil; curr = curr.Parent {
		if ast.IsExpressionStatement(curr) ||
			ast.IsVariableStatement(curr) ||
			ast.IsReturnStatement(curr) ||
			ast.IsIfStatement(curr) ||
			ast.IsSwitchStatement(curr) ||
			ast.IsDoStatement(curr) ||
			ast.IsWhileStatement(curr) ||
			ast.IsForStatement(curr) ||
			ast.IsForInStatement(curr) ||
			ast.IsForOfStatement(curr) ||
			ast.IsContinueStatement(curr) ||
			ast.IsBreakStatement(curr) ||
			ast.IsThrowStatement(curr) ||
			ast.IsObjectBindingPattern(curr) ||
			ast.IsArrayBindingPattern(curr) {
			return curr
		}
	}
	if node != nil {
		for curr := node; curr != nil; curr = curr.Parent {
			if ast.IsSourceFile(curr) {
				return curr
			}
		}
	}
	return node
}

func (v *traceDynamicValueVisitor) trace(value *DynamicValue) []*ast.Diagnostic {
	if value == nil {
		return nil
	}

	var trace []*ast.Diagnostic

	if value.fromDynamicInput {
		if childDv, ok := value.Reason.(*DynamicValue); ok {
			trace = v.trace(childDv)
		}
		if v.shouldTrace(value.Node) {
			info := ngtscdiag.MakeRelatedInformation(value.Node, "Unable to evaluate this expression statically.")
			trace = append([]*ast.Diagnostic{info}, trace...)
		}
		return trace
	}

	if value.Reason != nil {
		if _, ok := value.Reason.(*SyntheticValue); ok {
			return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, "Unable to evaluate this expression further.")}
		}
	}

	if value.fromDynamicString {
		return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, "A string value could not be determined statically.")}
	}

	if value.fromExternalReference {
		name := ""
		if ref, ok := value.Reason.(*imports.Reference); ok {
			name = ref.DebugName()
		}
		description := "an anonymous declaration"
		if name != "" {
			description = fmt.Sprintf("'%s'", name)
		}
		msg := fmt.Sprintf("A value for %s cannot be determined statically, as it is an external declaration.", description)
		return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, msg)}
	}

	if value.fromComplexFunctionCall {
		var fnNode *ast.Node
		if node, ok := value.Reason.(*ast.Node); ok {
			fnNode = node
		}
		return []*ast.Diagnostic{
			ngtscdiag.MakeRelatedInformation(value.Node, "Unable to evaluate function call of complex function. A function must have exactly one return statement."),
			ngtscdiag.MakeRelatedInformation(fnNode, "Function is declared here."),
		}
	}

	if value.fromInvalidExpressionType {
		return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, "Unable to evaluate an invalid expression.")}
	}

	if value.fromDynamicType {
		return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, "Dynamic type.")}
	}

	if value.fromUnsupportedSyntax {
		return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, "This syntax is not supported.")}
	}

	reasonStr := ""
	if s, ok := value.Reason.(string); ok {
		reasonStr = s
	}
	if strings.HasPrefix(reasonStr, "identifier reference:") || (value.Node != nil && ast.IsIdentifier(value.Node)) {
		return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, "Unknown reference.")}
	}

	return []*ast.Diagnostic{ngtscdiag.MakeRelatedInformation(value.Node, "Unable to evaluate statically.")}
}
