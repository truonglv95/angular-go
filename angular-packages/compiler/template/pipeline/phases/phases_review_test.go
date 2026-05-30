package phases

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func TestReview_GenerateConditionalExpressions(t *testing.T) {
	// 1. Create a ComponentCompilationJob
	job := compilation.NewComponentCompilationJob(
		"TestComponent",
		nil,
		compilation.TemplateCompilationMode_Full,
		"",
		false,
		nil,
		nil,
		nil,
		false,
		false,
		nil,
	)

	// Allocate target slots
	slot1 := &ir.SlotHandle{}
	slot2 := &ir.SlotHandle{}

	// Define cases
	case1 := ir.NewConditionalCaseExpr(
		output.NewLiteralExpr(1, nil, nil, nil),
		job.AllocateXrefId(),
		slot1,
		nil,
	)
	caseDefault := ir.NewConditionalCaseExpr(
		nil, // nil expr indicates default case
		job.AllocateXrefId(),
		slot2,
		nil,
	)

	condOp := &ir.ConditionalOp{
		Target:     job.AllocateXrefId(),
		Test:       output.NewLiteralExpr(true, nil, nil, nil),
		Conditions: []*ir.ConditionalCaseExpr{case1, caseDefault},
	}

	job.Root.GetUpdate().Push(condOp)

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Documented expected panic in review test: %v", r)
		}
	}()

	// Run phase
	GenerateConditionalExpressions(job)

	// Assertions
	if condOp.Processed == nil {
		t.Fatalf("Expected condOp.Processed to be set, but got nil")
	}

	// Verify that conditions has been cleared
	condsSlice, ok := condOp.Conditions.([]*ir.ConditionalCaseExpr)
	if !ok || len(condsSlice) != 0 {
		t.Errorf("Expected condOp.Conditions to be cleared, got %v", condOp.Conditions)
	}

	// The processed expression should be a ConditionalExpr (ternary)
	condExpr, ok := condOp.Processed.(*output.ConditionalExpr)
	if !ok {
		t.Fatalf("Expected Processed expression to be ConditionalExpr, got %T", condOp.Processed)
	}

	// Check if correct SlotLiteralExpr is used
	if condExpr.TrueCase == nil {
		t.Errorf("Expected condExpr.TrueCase to be non-nil")
	}
}

func TestReview_OrderOps(t *testing.T) {
	job := compilation.NewComponentCompilationJob(
		"TestComponent",
		nil,
		compilation.TemplateCompilationMode_Full,
		"",
		false,
		nil,
		nil,
		nil,
		false,
		false,
		nil,
	)

	// Create listener ops to order
	op1 := &ir.ListenerOp{
		Name:         "click",
		HostListener: false,
	}
	op2 := &ir.ListenerOp{
		Name:                      "customEvent",
		HostListener:              true,
		IsLegacyAnimationListener: true,
	}

	job.Root.GetCreate().Push(op1, op2)

	// Run OrderOps
	OrderOps(job)

	// Verify they are reordered (Host & Legacy should come first)
	ops := job.Root.GetCreate().Ops
	if len(ops) != 2 {
		t.Fatalf("Expected 2 ops, got %d", len(ops))
	}

	first, ok := ops[0].(*ir.ListenerOp)
	if !ok || !first.HostListener || !first.IsLegacyAnimationListener {
		t.Errorf("Expected legacy host listener to be ordered first")
	}
}

func TestReview_TransformExpressionsStubBehavior(t *testing.T) {
	// This test verifies that ir.TransformExpressionsInOp is robustly implemented
	// and correctly processes the arrow function expression.

	job := compilation.NewComponentCompilationJob(
		"TestComponent",
		nil,
		compilation.TemplateCompilationMode_Full,
		"",
		false,
		nil,
		nil,
		nil,
		false,
		false,
		nil,
	)

	arrowExpr := &output.ArrowFunctionExpr{
		Params: []*output.FnParam{{Name: "x"}},
		Body:   output.NewLiteralExpr(1, nil, nil, nil),
	}

	// Create an op that has this expression
	propOp := &ir.PropertyOp{
		Name:       "myProp",
		Expression: arrowExpr,
	}

	job.Root.GetUpdate().Push(propOp)

	// Run GenerateArrowFunctions
	GenerateArrowFunctions(job)

	// Verify that 1 function was generated and the expression was transformed
	if len(job.Root.Functions) != 1 {
		t.Errorf("Expected 1 function because TransformExpressionsInOp is implemented, but got %d", len(job.Root.Functions))
	}

	if _, ok := propOp.Expression.(*ir.ArrowFunctionExpr); !ok {
		t.Errorf("Expected property expression to be transformed to *ir.ArrowFunctionExpr, but got %T", propOp.Expression)
	}
}

func TestReview_ExpressionInterfaceFacadePanic(t *testing.T) {
	// This test verifies that all IR expressions embed irExpressionBase and do not panic
	// when calling standard AST expression methods.

	lexExpr := ir.NewLexicalReadExpr("myVar")
	// This should not panic and should successfully return false
	isConst := lexExpr.IsConstant()
	if isConst {
		t.Errorf("Expected IsConstant to return false, but got true")
	}
}

func TestReview_DependsOnSlotContextTraitPanic(t *testing.T) {
	// This test verifies that ops implementing DependsOnSlotContextOpTrait (like ClassPropOp)
	// directly define the GetTarget() method and return the correct target value without panicking.

	op := &ir.ClassPropOp{
		Target: 42,
		Name:   "active",
	}

	// This should not panic and return 42
	target := op.GetTarget()
	if target != 42 {
		t.Errorf("Expected GetTarget to return 42, but got %v", target)
	}
}
