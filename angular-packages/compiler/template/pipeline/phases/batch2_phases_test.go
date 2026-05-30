package phases

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func TestCreatePipes(t *testing.T) {
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

	unit := job.Root

	elementOp := &ir.ElementStartOp{
		ElementOpBase: ir.ElementOpBase{
			ElementOrContainerOpBase: ir.ElementOrContainerOpBase{
				Xref: 10,
			},
		},
	}

	pipeBindingExpr := &ir.PipeBindingExpr{
		Target:     20,
		PipeName:   "date",
		Args:       []output.Expression{output.NewLiteralExpr("val", nil, nil, nil)},
	}
	pipeBindingExpr.Self = pipeBindingExpr

	bindingOp := &ir.BindingOp{
		Target:     10,
		Expression: pipeBindingExpr,
	}

	unit.GetCreate().Push(elementOp)
	unit.GetUpdate().Push(bindingOp)

	CreatePipes(job)

	// Verify that a PipeOp was inserted into the create block.
	createOps := unit.GetCreate().Ops
	if len(createOps) != 2 {
		t.Fatalf("Expected 2 create ops, got %d", len(createOps))
	}

	pipeOp, ok := createOps[1].(*ir.PipeOp)
	if !ok {
		t.Fatalf("Expected second create op to be a PipeOp, got %T", createOps[1])
	}

	if pipeOp.Xref != 20 {
		t.Errorf("Expected PipeOp Xref to be 20, got %d", pipeOp.Xref)
	}

	if pipeOp.Name != "date" {
		t.Errorf("Expected PipeOp name to be 'date', got %s", pipeOp.Name)
	}
}

func TestCreateVariadicPipes(t *testing.T) {
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

	unit := job.Root

	// Pipe with 1 arg (should remain unchanged)
	pipeBinding1 := &ir.PipeBindingExpr{
		Target:   20,
		PipeName: "shortPipe",
		Args:     []output.Expression{output.NewLiteralExpr("a", nil, nil, nil)},
	}
	pipeBinding1.Self = pipeBinding1

	// Pipe with 5 args (should become variadic)
	pipeBinding2 := &ir.PipeBindingExpr{
		Target:   30,
		PipeName: "largePipe",
		Args: []output.Expression{
			output.NewLiteralExpr("a", nil, nil, nil),
			output.NewLiteralExpr("b", nil, nil, nil),
			output.NewLiteralExpr("c", nil, nil, nil),
			output.NewLiteralExpr("d", nil, nil, nil),
			output.NewLiteralExpr("e", nil, nil, nil),
		},
	}
	pipeBinding2.Self = pipeBinding2

	bindingOp1 := &ir.BindingOp{
		Target:     10,
		Expression: pipeBinding1,
	}
	bindingOp2 := &ir.BindingOp{
		Target:     10,
		Expression: pipeBinding2,
	}

	unit.GetUpdate().Push(bindingOp1, bindingOp2)

	CreateVariadicPipes(job)

	// Verify update ops transformation
	if _, ok := bindingOp1.Expression.(*ir.PipeBindingExpr); !ok {
		t.Errorf("Expected bindingOp1 expression to remain PipeBindingExpr, got %T", bindingOp1.Expression)
	}

	variadicExpr, ok := bindingOp2.Expression.(*ir.PipeBindingVariadicExpr)
	if !ok {
		t.Fatalf("Expected bindingOp2 expression to be transformed into PipeBindingVariadicExpr, got %T", bindingOp2.Expression)
	}

	if variadicExpr.NumArgs != 5 {
		t.Errorf("Expected NumArgs to be 5, got %d", variadicExpr.NumArgs)
	}

	literalArr, ok := variadicExpr.Args.(*output.LiteralArrayExpr)
	if !ok {
		t.Fatalf("Expected Args to be a LiteralArrayExpr, got %T", variadicExpr.Args)
	}

	if len(literalArr.Entries) != 5 {
		t.Errorf("Expected LiteralArrayExpr to have 5 entries, got %d", len(literalArr.Entries))
	}
}

func TestConfigureDeferInstructions(t *testing.T) {
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

	unit := job.Root

	placeholderTime := 1000
	loadingTime := 2000
	loadingAfter := 500

	deferOp := &ir.DeferOp{
		Xref:                   10,
		PlaceholderMinimumTime: &placeholderTime,
		LoadingMinimumTime:     &loadingTime,
		LoadingAfterTime:       &loadingAfter,
	}

	unit.GetCreate().Push(deferOp)

	ConfigureDeferInstructions(job)

	// Verify collected configurations
	placeholderConfig, ok := deferOp.PlaceholderConfig.(*ir.ConstCollectedExpr)
	if !ok {
		t.Fatalf("Expected PlaceholderConfig to be ConstCollectedExpr, got %T", deferOp.PlaceholderConfig)
	}

	pArr, ok := placeholderConfig.Expr.(*output.LiteralArrayExpr)
	if !ok {
		t.Fatalf("Expected collected placeholder expression to be LiteralArrayExpr, got %T", placeholderConfig.Expr)
	}

	if len(pArr.Entries) != 1 {
		t.Fatalf("Expected 1 entry in placeholder config array, got %d", len(pArr.Entries))
	}

	pLit, ok := pArr.Entries[0].(*output.LiteralExpr)
	if !ok || pLit.Value != 1000 {
		t.Errorf("Expected entry to be LiteralExpr with value 1000, got %v", pArr.Entries[0])
	}

	loadingConfig, ok := deferOp.LoadingConfig.(*ir.ConstCollectedExpr)
	if !ok {
		t.Fatalf("Expected LoadingConfig to be ConstCollectedExpr, got %T", deferOp.LoadingConfig)
	}

	lArr, ok := loadingConfig.Expr.(*output.LiteralArrayExpr)
	if !ok {
		t.Fatalf("Expected collected loading expression to be LiteralArrayExpr, got %T", loadingConfig.Expr)
	}

	if len(lArr.Entries) != 2 {
		t.Fatalf("Expected 2 entries in loading config array, got %d", len(lArr.Entries))
	}

	lLit1, ok := lArr.Entries[0].(*output.LiteralExpr)
	if !ok || lLit1.Value != 2000 {
		t.Errorf("Expected loading min time entry to be LiteralExpr with value 2000, got %v", lArr.Entries[0])
	}

	lLit2, ok := lArr.Entries[1].(*output.LiteralExpr)
	if !ok || lLit2.Value != 500 {
		t.Errorf("Expected loading after time entry to be LiteralExpr with value 500, got %v", lArr.Entries[1])
	}
}

func TestInsertIncrementalHydrationRuntime(t *testing.T) {
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

	unit := job.Root

	deferOp1 := &ir.DeferOp{
		Xref:  10,
		Flags: ir.TDeferDetailsFlagsHasHydrateTriggers,
	}
	deferOp2 := &ir.DeferOp{
		Xref:  20,
		Flags: ir.TDeferDetailsFlagsHasHydrateTriggers,
	}

	unit.GetCreate().Push(deferOp1, deferOp2)

	InsertIncrementalHydrationRuntime(job)

	// Verify insertion of the hydration activator before the first defer op
	createOps := unit.GetCreate().Ops
	if len(createOps) != 3 {
		t.Fatalf("Expected 3 create ops, got %d", len(createOps))
	}

	if _, ok := createOps[0].(*ir.EnableIncrementalHydrationRuntimeOp); !ok {
		t.Errorf("Expected first op to be EnableIncrementalHydrationRuntimeOp, got %T", createOps[0])
	}

	if op, ok := createOps[1].(*ir.DeferOp); !ok || op.Xref != 10 {
		t.Errorf("Expected second op to be DeferOp with Xref 10, got %v", createOps[1])
	}

	if op, ok := createOps[2].(*ir.DeferOp); !ok || op.Xref != 20 {
		t.Errorf("Expected third op to be DeferOp with Xref 20, got %v", createOps[2])
	}
}
