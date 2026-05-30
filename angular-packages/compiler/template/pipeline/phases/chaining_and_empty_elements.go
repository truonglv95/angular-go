package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CollapseEmptyInstructions replaces sequences of mergeable instructions (e.g. ElementStart and
// ElementEnd) with consolidated instructions (e.g. Element).
func CollapseEmptyInstructions(job compilation.CompilationJob) {
	replacements := map[ir.OpKind][2]ir.OpKind{
		ir.OpKindElementEnd:   {ir.OpKindElementStart, ir.OpKindElement},
		ir.OpKindContainerEnd: {ir.OpKindContainerStart, ir.OpKindContainer},
		ir.OpKindI18nEnd:      {ir.OpKindI18nStart, ir.OpKindI18n},
	}

	ignoredOpKinds := map[ir.OpKind]bool{
		ir.OpKindPipe: true,
	}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			opReplacements, ok := replacements[op.Kind()]
			if !ok {
				continue
			}
			startKind := opReplacements[0]
			mergedKind := opReplacements[1]

			// Locate the previous (non-ignored) op via linked list.
			prevOp := op.Prev()
			for prevOp != nil && ignoredOpKinds[prevOp.Kind()] {
				prevOp = prevOp.Prev()
			}

			// If the previous op is the corresponding start op, merge.
			if prevOp != nil && prevOp.Kind() == startKind {
				if setter, ok2 := prevOp.(interface{ SetKind(ir.OpKind) }); ok2 {
					setter.SetKind(mergedKind)
				}
				unit.GetCreate().Remove(op)
			}
		}
	}
}

// Chain post-processes a reified view compilation and converts sequential calls to chainable
// instructions into chain calls.
func Chain(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		chainOpsInList(unit.GetCreate())
		chainOpsInList(unit.GetUpdate())
	}
}

const maxChainLength = 256

// chainCompatibility maps each chainable instruction name to the successor instruction it can chain with.
var chainCompatibility = map[string]string{
	"ariaProperty":             "ariaProperty",
	"attribute":                "attribute",
	"classProp":                "classProp",
	"element":                  "element",
	"elementContainer":         "elementContainer",
	"elementContainerEnd":      "elementContainerEnd",
	"elementContainerStart":    "elementContainerStart",
	"elementEnd":               "elementEnd",
	"elementStart":             "elementStart",
	"domProperty":              "domProperty",
	"i18nExp":                  "i18nExp",
	"listener":                 "listener",
	"property":                 "property",
	"styleProp":                "styleProp",
	"syntheticHostListener":    "syntheticHostListener",
	"syntheticHostProperty":    "syntheticHostProperty",
	"templateCreate":           "templateCreate",
	"twoWayProperty":           "twoWayProperty",
	"twoWayListener":           "twoWayListener",
	"declareLet":               "declareLet",
	"conditionalCreate":        "conditionalBranchCreate",
	"conditionalBranchCreate":  "conditionalBranchCreate",
	"domElement":               "domElement",
	"domElementStart":          "domElementStart",
	"domElementEnd":            "domElementEnd",
	"domElementContainer":      "domElementContainer",
	"domElementContainerStart": "domElementContainerStart",
	"domElementContainerEnd":   "domElementContainerEnd",
	"domListener":              "domListener",
	"domTemplate":              "domTemplate",
	"animationEnter":           "animationEnter",
	"animationLeave":           "animationLeave",
	"animationEnterListener":   "animationEnterListener",
	"animationLeaveListener":   "animationLeaveListener",
}

type chainState struct {
	op          ir.Op
	instruction string
	expression  output.Expression
	length      int
}

func chainOpsInList(opList *ir.OpList) {
	var chain *chainState
	for _, op := range opList.Elements() {
		stmtOp, ok := op.(interface{ GetStatement() output.Statement })
		if !ok {
			chain = nil
			continue
		}
		exprStmt, ok := stmtOp.GetStatement().(*output.ExpressionStatement)
		if !ok {
			chain = nil
			continue
		}
		invoke, ok := exprStmt.Expr.(*output.InvokeFunctionExpr)
		if !ok {
			chain = nil
			continue
		}
		ext, ok := invoke.Fn.(*output.ExternalExpr)
		if !ok {
			chain = nil
			continue
		}

		instructionName := ""
		if ext.Value.Name != nil {
			instructionName = *ext.Value.Name
		}
		successor, chainable := chainCompatibility[instructionName]
		if !chainable {
			chain = nil
			continue
		}

		if chain != nil &&
			chainCompatibility[chain.instruction] == successor &&
			chain.length < maxChainLength {
			// Add to existing chain by calling fn on the chain expression.
			prevInvoke := chain.expression.(*output.InvokeFunctionExpr)
			newExpr := prevInvoke.CallFn(invoke.Args, invoke.SourceSpan, invoke.Pure, nil)
			chain.expression = newExpr
			if setter, ok2 := chain.op.(interface{ SetStatement(output.Statement) }); ok2 {
				setter.SetStatement(newExpr.ToStmt(nil))
			}
			chain.length++
			opList.Remove(op)
		} else {
			chain = &chainState{
				op:          op,
				instruction: instructionName,
				expression:  invoke,
				length:      1,
			}
		}
	}
}
