package phases

import (
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// OpListReplace replaces an operation in an OpList slice.
func OpListReplace(list *ir.OpList, target ir.Op, newOp ir.Op) {
	for i, op := range list.Ops {
		if op == target {
			list.Ops[i] = newOp
			return
		}
	}
}

// createOpXrefMap constructs a map from XrefId to its corresponding slot-consuming create op.
func createOpXrefMap(unit compilation.CompilationUnit) map[ir.XrefId]ir.Op {
	m := make(map[ir.XrefId]ir.Op)
	for _, op := range unit.GetCreate().Elements() {
		if trait, ok := op.(ir.ConsumesSlotOpTrait); ok {
			m[trait.GetXref()] = op
		}
		if repeater, ok := op.(*ir.RepeaterCreateOp); ok {
			// In Go, RepeaterCreateOp has EmptyView XrefId.
			m[repeater.EmptyView] = op
		}
	}
	return m
}

// isAriaAttribute checks if the name corresponds to an ARIA attribute.
func isAriaAttribute(name string) bool {
	return strings.HasPrefix(name, "aria-") && len(name) > len("aria-")
}

// hyphenateRegexp compiles the regex used for camelCase hyphenation.
var hyphenateRegexp = regexp.MustCompile(`([a-z])([A-Z])`)

// hyphenate converts camelCase names to hyphenated-case.
func hyphenate(value string) string {
	res := hyphenateRegexp.ReplaceAllString(value, "${1}-${2}")
	return strings.ToLower(res)
}

// parseProperty parses a property name and unit suffix.
func parseProperty(name string) (string, *string) {
	overrideIndex := strings.Index(name, "!important")
	if overrideIndex != -1 {
		if overrideIndex > 0 {
			name = name[:overrideIndex]
		} else {
			name = ""
		}
	}

	var suffix *string
	property := name
	unitIndex := strings.LastIndex(name, ".")
	if unitIndex > 0 {
		s := name[unitIndex+1:]
		suffix = &s
		property = name[:unitIndex]
	}
	return property, suffix
}

// setNonBindable sets the NonBindable property on element-like operations.
func setNonBindable(op ir.Op, val bool) {
	switch o := op.(type) {
	case *ir.ElementStartOp:
		o.NonBindable = val
	case *ir.ElementOp:
		o.NonBindable = val
	case *ir.TemplateOp:
		o.NonBindable = val
	case *ir.ConditionalCreateOp:
		o.NonBindable = val
	case *ir.ConditionalBranchCreateOp:
		o.NonBindable = val
	case *ir.ContainerStartOp:
		o.NonBindable = val
	case *ir.ContainerOp:
		o.NonBindable = val
	}
}

// ConstantPool defines the required interface for ConstantPool integration.
type ConstantPool interface {
	GetSharedConstant(key any, expr output.Expression) output.Expression
	UniqueName(name string) string
}

// LinkedOp is implemented by ops that track their doubly-linked list neighbours.
type LinkedOp interface {
	Prev() ir.Op
	Next() ir.Op
}

// sanitizeIdentifier converts a string into a valid TypeScript/JS identifier
// by replacing non-alphanumeric characters with underscores.
var sanitizeIdentifierRegexp = regexp.MustCompile(`[^a-zA-Z0-9_$]`)

func sanitizeIdentifier(name string) string {
	return sanitizeIdentifierRegexp.ReplaceAllString(name, "_")
}
