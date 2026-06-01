package phases

import (
	"strings"

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
		for i := 0; i < len(unit.GetCreate().Elements()); {
			op := unit.GetCreate().Elements()[i]
			opReplacements, ok := replacements[op.Kind()]
			if !ok {
				i++
				continue
			}
			startKind := opReplacements[0]
			mergedKind := opReplacements[1]

			var prevOp ir.Op
			for j := i - 1; j >= 0; j-- {
				candidate := unit.GetCreate().Elements()[j]
				if !ignoredOpKinds[candidate.Kind()] {
					prevOp = candidate
					break
				}
			}

			if prevOp != nil && prevOp.Kind() == startKind {
				if setter, ok2 := prevOp.(interface{ SetKind(ir.OpKind) }); ok2 {
					setter.SetKind(mergedKind)
				}
				unit.GetCreate().Remove(op)
				// Do not increment i, as the next element shifted into the current index
			} else {
				i++
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
	for i := 0; i < len(opList.Elements()); {
		op := opList.Elements()[i]
		stmtOp, ok := op.(interface{ GetStatement() output.Statement })
		if !ok {
			chain = nil
			i++
			continue
		}
		exprStmt, ok := stmtOp.GetStatement().(*output.ExpressionStatement)
		if !ok {
			chain = nil
			i++
			continue
		}
		invoke, ok := exprStmt.Expr.(*output.InvokeFunctionExpr)
		if !ok {
			chain = nil
			i++
			continue
		}

		var instructionName string
		if ext, ok := invoke.Fn.(*output.ExternalExpr); ok && ext.Value.Name != nil {
			instructionName = *ext.Value.Name
		} else if read, ok := invoke.Fn.(*output.ReadVarExpr); ok {
			instructionName = read.Name
		}

		if !strings.HasPrefix(instructionName, "ɵɵ") {
			chain = nil
			i++
			continue
		}
		instructionName = strings.TrimPrefix(instructionName, "ɵɵ")

		successor, chainable := chainCompatibility[instructionName]
		if !chainable {
			chain = nil
			i++
			continue
		}

		if chain != nil &&
			chainCompatibility[chain.instruction] == successor &&
			chain.length < maxChainLength {
			// Chain the call by wrapping the old expression in the new invoke's Fn.
			invoke.Fn = chain.expression
			chain.expression = invoke
			if setter, ok2 := chain.op.(interface{ SetStatement(output.Statement) }); ok2 {
				setter.SetStatement(output.NewExpressionStatement(invoke, nil, nil))
			}
			chain.instruction = instructionName
			chain.length++
			opList.Remove(op)
			// Do not increment i, as the next element shifted into the current index
		} else {
			chain = &chainState{
				op:          op,
				instruction: instructionName,
				expression:  invoke,
				length:      1,
			}
			i++
		}
	}
}
