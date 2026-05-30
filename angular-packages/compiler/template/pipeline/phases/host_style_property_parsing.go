package phases

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ParseHostStyleProperties parses host style binding properties.
func ParseHostStyleProperties(job compilation.CompilationJob) {
	for _, op := range job.GetRoot().GetUpdate().Elements() {
		bindingOp, ok := op.(*ir.BindingOp)
		if !ok {
			continue
		}
		kind, ok := bindingOp.BindingKind.(ir.BindingKind)
		if !ok || kind != ir.BindingKindProperty {
			continue
		}

		if strings.HasSuffix(bindingOp.Name, "!important") {
			bindingOp.Name = bindingOp.Name[:len(bindingOp.Name)-len("!important")]
		}

		if strings.HasPrefix(bindingOp.Name, "style.") {
			bindingOp.BindingKind = ir.BindingKindStyleProperty
			bindingOp.Name = bindingOp.Name[len("style."):]

			if !strings.HasPrefix(bindingOp.Name, "--") {
				bindingOp.Name = hyphenate(bindingOp.Name)
			}

			prop, suffix := parseProperty(bindingOp.Name)
			bindingOp.Name = prop
			bindingOp.Unit = suffix
		} else if strings.HasPrefix(bindingOp.Name, "style!") {
			bindingOp.BindingKind = ir.BindingKindStyleProperty
			bindingOp.Name = "style"
		} else if strings.HasPrefix(bindingOp.Name, "class.") {
			bindingOp.BindingKind = ir.BindingKindClassName
			prop, _ := parseProperty(bindingOp.Name[len("class."):])
			bindingOp.Name = prop
		} else if strings.HasPrefix(bindingOp.Name, "class!") {
			bindingOp.BindingKind = ir.BindingKindClassName
			prop, _ := parseProperty(bindingOp.Name[len("class!"):])
			bindingOp.Name = prop
		}
	}
}
