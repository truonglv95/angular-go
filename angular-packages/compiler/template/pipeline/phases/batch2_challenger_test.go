package phases

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// TestCreatePipes_MissingSlotPanic verifies that CreatePipes panics when encountering
// an update operation that does not have an assignable target slot context or type.
func TestCreatePipes_MissingSlotPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected CreatePipes to panic, but it did not")
		}
		errMsg, ok := r.(string)
		if !ok || !strings.Contains(errMsg, "expected slot handle to be assigned for pipe creation") {
			t.Fatalf("Expected panic message to contain 'expected slot handle to be assigned for pipe creation', got: %v", r)
		}
	}()

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

	pipeBindingExpr := &ir.PipeBindingExpr{
		Target:   20,
		PipeName: "testPipe",
		Args:     []output.Expression{output.NewLiteralExpr("val", nil, nil, nil)},
	}
	pipeBindingExpr.Self = pipeBindingExpr

	// VariableOp has an Initializer but is NOT handled by getTargetXref, and does not implement DependsOnSlotContextOpTrait.
	// This will force getTargetXref to return (0, false) and trigger the panic.
	varOp := &ir.VariableOp{
		Xref:        10,
		Initializer: pipeBindingExpr,
	}

	unit.GetUpdate().Push(varOp)

	CreatePipes(job)
}

// TestCreatePipes_NoMatchingSlotInCreationPanic verifies that CreatePipes panics
// when a pipe's target slot is assigned, but there is no corresponding slot-consuming
// operation in the creation block (meaning we can't find an insertion point).
func TestCreatePipes_NoMatchingSlotInCreationPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected CreatePipes to panic, but it did not")
		}
		errMsg, ok := r.(string)
		if !ok || !strings.Contains(errMsg, "unable to find insertion point for pipe") {
			t.Fatalf("Expected panic message to contain 'unable to find insertion point for pipe', got: %v", r)
		}
	}()

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

	pipeBindingExpr := &ir.PipeBindingExpr{
		Target:   20,
		PipeName: "testPipe",
		Args:     []output.Expression{output.NewLiteralExpr("val", nil, nil, nil)},
	}
	pipeBindingExpr.Self = pipeBindingExpr

	// BindingOp targets Xref 9999, but we won't add any ConsumesSlotOp with Xref 9999 in GetCreate().
	bindingOp := &ir.BindingOp{
		Target:     9999,
		Expression: pipeBindingExpr,
	}

	unit.GetUpdate().Push(bindingOp)

	CreatePipes(job)
}

// TestCreatePipes_ChildExpressionPanic verifies that CreatePipes panics
// when a PipeBindingExpr appears inside a child operation (VisitorContextFlagInChildOperation is active).
func TestCreatePipes_ChildExpressionPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected CreatePipes to panic, but it did not")
		}
		errMsg, ok := r.(string)
		if !ok || !strings.Contains(errMsg, "pipe bindings should not appear in child expressions") {
			t.Fatalf("Expected panic message to contain 'pipe bindings should not appear in child expressions', got: %v", r)
		}
	}()

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

	pipeBindingExpr := &ir.PipeBindingExpr{
		Target:   20,
		PipeName: "testPipe",
		Args:     []output.Expression{output.NewLiteralExpr("val", nil, nil, nil)},
	}
	pipeBindingExpr.Self = pipeBindingExpr

	// BindingOp inside child handler list
	childBinding := &ir.BindingOp{
		Target:     10,
		Expression: pipeBindingExpr,
	}

	childList := ir.NewOpList()
	childList.Push(childBinding)

	// AnimationOp with child handler ops
	animationOp := &ir.AnimationOp{
		Target:     10,
		Name:       "myAnim",
		HandlerOps: childList,
	}

	unit.GetUpdate().Push(animationOp)

	CreatePipes(job)
}

// TestCreatePipes_MultiplePipesSameSlot verifies that multiple pipes targeting the same slot
// are inserted in the correct sequential order.
func TestCreatePipes_MultiplePipesSameSlot(t *testing.T) {
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

	// Add an element op consuming slot 10
	elementOp := &ir.ElementStartOp{
		ElementOpBase: ir.ElementOpBase{
			ElementOrContainerOpBase: ir.ElementOrContainerOpBase{
				Xref: 10,
			},
		},
	}

	unit.GetCreate().Push(elementOp)

	// Create three different pipe bindings on slot 10
	pipe1 := &ir.PipeBindingExpr{
		Target:   21,
		PipeName: "pipeOne",
		Args:     []output.Expression{output.NewLiteralExpr("val1", nil, nil, nil)},
	}
	pipe1.Self = pipe1

	pipe2 := &ir.PipeBindingExpr{
		Target:   22,
		PipeName: "pipeTwo",
		Args:     []output.Expression{output.NewLiteralExpr("val2", nil, nil, nil)},
	}
	pipe2.Self = pipe2

	pipe3 := &ir.PipeBindingExpr{
		Target:   23,
		PipeName: "pipeThree",
		Args:     []output.Expression{output.NewLiteralExpr("val3", nil, nil, nil)},
	}
	pipe3.Self = pipe3

	// Associate bindings in the update block
	binding1 := &ir.BindingOp{
		Target:     10,
		Expression: pipe1,
	}
	binding2 := &ir.BindingOp{
		Target:     10,
		Expression: pipe2,
	}
	binding3 := &ir.BindingOp{
		Target:     10,
		Expression: pipe3,
	}

	unit.GetUpdate().Push(binding1, binding2, binding3)

	CreatePipes(job)

	// Verify insertion sequence in creation block: Element(10), Pipe(pipeOne), Pipe(pipeTwo), Pipe(pipeThree)
	createOps := unit.GetCreate().Ops
	if len(createOps) != 4 {
		t.Fatalf("Expected 4 ops in creation block, got %d", len(createOps))
	}

	if el, ok := createOps[0].(*ir.ElementStartOp); !ok || el.Xref != 10 {
		t.Errorf("Expected first op to be ElementStartOp (10), got %v", createOps[0])
	}

	if p1, ok := createOps[1].(*ir.PipeOp); !ok || p1.Name != "pipeOne" || p1.Xref != 21 {
		t.Errorf("Expected second op to be PipeOp 'pipeOne' (21), got %v", createOps[1])
	}

	if p2, ok := createOps[2].(*ir.PipeOp); !ok || p2.Name != "pipeTwo" || p2.Xref != 22 {
		t.Errorf("Expected third op to be PipeOp 'pipeTwo' (22), got %v", createOps[2])
	}

	if p3, ok := createOps[3].(*ir.PipeOp); !ok || p3.Name != "pipeThree" || p3.Xref != 23 {
		t.Errorf("Expected fourth op to be PipeOp 'pipeThree' (23), got %v", createOps[3])
	}
}

// TestCreateVariadicPipes_AdversarialArgs stress tests different numbers of arguments in pipe expressions,
// ensuring the variadic threshold (<=4 args stay unchanged, >4 args converted to PipeBindingVariadicExpr) is correct.
func TestCreateVariadicPipes_AdversarialArgs(t *testing.T) {
	tests := []struct {
		name         string
		argCount     int
		wantVariadic bool
	}{
		{"0 args", 0, false},
		{"1 arg", 1, false},
		{"2 args", 2, false},
		{"3 args", 3, false},
		{"4 args", 4, false},
		{"5 args (boundary)", 5, true},
		{"10 args", 10, true},
		{"100 args (extreme)", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			args := make([]output.Expression, tt.argCount)
			for i := 0; i < tt.argCount; i++ {
				args[i] = output.NewLiteralExpr(i, nil, nil, nil)
			}

			pipeBinding := &ir.PipeBindingExpr{
				Target:   10,
				PipeName: "testPipe",
				Args:     args,
			}
			pipeBinding.Self = pipeBinding

			bindingOp := &ir.BindingOp{
				Target:     10,
				Expression: pipeBinding,
			}
			unit.GetUpdate().Push(bindingOp)

			CreateVariadicPipes(job)

			expr := bindingOp.Expression
			if tt.wantVariadic {
				variadicExpr, ok := expr.(*ir.PipeBindingVariadicExpr)
				if !ok {
					t.Fatalf("Expected PipeBindingVariadicExpr for %d args, got %T", tt.argCount, expr)
				}
				if variadicExpr.NumArgs != tt.argCount {
					t.Errorf("Expected NumArgs to be %d, got %d", tt.argCount, variadicExpr.NumArgs)
				}
				literalArr, ok := variadicExpr.Args.(*output.LiteralArrayExpr)
				if !ok {
					t.Fatalf("Expected Args to be LiteralArrayExpr, got %T", variadicExpr.Args)
				}
				if len(literalArr.Entries) != tt.argCount {
					t.Errorf("Expected literal array to have %d entries, got %d", tt.argCount, len(literalArr.Entries))
				}
			} else {
				regularExpr, ok := expr.(*ir.PipeBindingExpr)
				if !ok {
					t.Fatalf("Expected regular PipeBindingExpr for %d args, got %T", tt.argCount, expr)
				}
				if len(regularExpr.Args) != tt.argCount {
					t.Errorf("Expected regular pipe args count to be %d, got %d", tt.argCount, len(regularExpr.Args))
				}
			}
		})
	}
}

// TestConfigureDeferInstructions_TimingEdgeCases stress-tests timing parameter configurations inside DeferOp,
// checking combinations of nil, zero, and negative values.
func TestConfigureDeferInstructions_TimingEdgeCases(t *testing.T) {
	zero := 0
	minusTen := -10
	val100 := 100
	val500 := 500

	tests := []struct {
		name               string
		placeholderTime    *int
		loadingMinTime     *int
		loadingAfterTime   *int
		expectPlaceholder  bool
		expectLoading      bool
		expectedPlacVal    any
		expectedMinVal     any
		expectedAfterVal   any
	}{
		{
			name:               "All nil",
			placeholderTime:    nil,
			loadingMinTime:     nil,
			loadingAfterTime:   nil,
			expectPlaceholder:  false,
			expectLoading:      false,
		},
		{
			name:               "Only placeholder non-nil (positive)",
			placeholderTime:    &val100,
			loadingMinTime:     nil,
			loadingAfterTime:   nil,
			expectPlaceholder:  true,
			expectLoading:      false,
			expectedPlacVal:    100,
		},
		{
			name:               "Only placeholder non-nil (zero)",
			placeholderTime:    &zero,
			loadingMinTime:     nil,
			loadingAfterTime:   nil,
			expectPlaceholder:  true,
			expectLoading:      false,
			expectedPlacVal:    0,
		},
		{
			name:               "Only placeholder non-nil (negative)",
			placeholderTime:    &minusTen,
			loadingMinTime:     nil,
			loadingAfterTime:   nil,
			expectPlaceholder:  true,
			expectLoading:      false,
			expectedPlacVal:    -10,
		},
		{
			name:               "Only loading min non-nil",
			placeholderTime:    nil,
			loadingMinTime:     &val500,
			loadingAfterTime:   nil,
			expectPlaceholder:  false,
			expectLoading:      true,
			expectedMinVal:     500,
			expectedAfterVal:   nil,
		},
		{
			name:               "Only loading after non-nil",
			placeholderTime:    nil,
			loadingMinTime:     nil,
			loadingAfterTime:   &val100,
			expectPlaceholder:  false,
			expectLoading:      true,
			expectedMinVal:     nil,
			expectedAfterVal:   100,
		},
		{
			name:               "Loading min and loading after both non-nil",
			placeholderTime:    nil,
			loadingMinTime:     &val500,
			loadingAfterTime:   &val100,
			expectPlaceholder:  false,
			expectLoading:      true,
			expectedMinVal:     500,
			expectedAfterVal:   100,
		},
		{
			name:               "All non-nil including zeros/negatives",
			placeholderTime:    &minusTen,
			loadingMinTime:     &zero,
			loadingAfterTime:   &val500,
			expectPlaceholder:  true,
			expectLoading:      true,
			expectedPlacVal:    -10,
			expectedMinVal:     0,
			expectedAfterVal:   500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			deferOp := &ir.DeferOp{
				Xref:                   10,
				PlaceholderMinimumTime: tt.placeholderTime,
				LoadingMinimumTime:     tt.loadingMinTime,
				LoadingAfterTime:       tt.loadingAfterTime,
			}

			unit.GetCreate().Push(deferOp)

			ConfigureDeferInstructions(job)

			// 1. Placeholder config verification
			if tt.expectPlaceholder {
				pConfig, ok := deferOp.PlaceholderConfig.(*ir.ConstCollectedExpr)
				if !ok {
					t.Fatalf("Expected PlaceholderConfig to be ConstCollectedExpr, got %T", deferOp.PlaceholderConfig)
				}
				pArr, ok := pConfig.Expr.(*output.LiteralArrayExpr)
				if !ok {
					t.Fatalf("Expected collected placeholder expression to be LiteralArrayExpr, got %T", pConfig.Expr)
				}
				if len(pArr.Entries) != 1 {
					t.Fatalf("Expected 1 entry in placeholder config array, got %d", len(pArr.Entries))
				}
				pLit, ok := pArr.Entries[0].(*output.LiteralExpr)
				if !ok {
					t.Fatalf("Expected entry to be LiteralExpr, got %T", pArr.Entries[0])
				}
				if pLit.Value != tt.expectedPlacVal {
					t.Errorf("Expected placeholder value %v, got %v", tt.expectedPlacVal, pLit.Value)
				}
			} else {
				if deferOp.PlaceholderConfig != nil {
					t.Errorf("Expected PlaceholderConfig to be nil, got %v", deferOp.PlaceholderConfig)
				}
			}

			// 2. Loading config verification
			if tt.expectLoading {
				lConfig, ok := deferOp.LoadingConfig.(*ir.ConstCollectedExpr)
				if !ok {
					t.Fatalf("Expected LoadingConfig to be ConstCollectedExpr, got %T", deferOp.LoadingConfig)
				}
				lArr, ok := lConfig.Expr.(*output.LiteralArrayExpr)
				if !ok {
					t.Fatalf("Expected collected loading expression to be LiteralArrayExpr, got %T", lConfig.Expr)
				}
				if len(lArr.Entries) != 2 {
					t.Fatalf("Expected 2 entries in loading config array, got %d", len(lArr.Entries))
				}

				// Check min time entry
				lLitMin, ok := lArr.Entries[0].(*output.LiteralExpr)
				if !ok {
					t.Fatalf("Expected first entry to be LiteralExpr, got %T", lArr.Entries[0])
				}
				if lLitMin.Value != tt.expectedMinVal {
					t.Errorf("Expected loading min value %v, got %v", tt.expectedMinVal, lLitMin.Value)
				}

				// Check after time entry
				lLitAfter, ok := lArr.Entries[1].(*output.LiteralExpr)
				if !ok {
					t.Fatalf("Expected second entry to be LiteralExpr, got %T", lArr.Entries[1])
				}
				if lLitAfter.Value != tt.expectedAfterVal {
					t.Errorf("Expected loading after value %v, got %v", tt.expectedAfterVal, lLitAfter.Value)
				}
			} else {
				if deferOp.LoadingConfig != nil {
					t.Errorf("Expected LoadingConfig to be nil, got %v", deferOp.LoadingConfig)
				}
			}
		})
	}
}

// TestInsertIncrementalHydrationRuntime_MultipleDeferOps tests various combinations
// of multiple DeferOps (with and without hydration triggers) within a view.
func TestInsertIncrementalHydrationRuntime_MultipleDeferOps(t *testing.T) {
	tests := []struct {
		name              string
		deferFlags        []any // list of flags for consecutive DeferOps
		expectInsertion   bool  // whether we expect EnableIncrementalHydrationRuntimeOp to be inserted
		expectedInsertIdx int   // index before which it should be inserted (in terms of the original defer index)
	}{
		{
			name:            "No DeferOps in view",
			deferFlags:      []any{},
			expectInsertion: false,
		},
		{
			name:            "Multiple DeferOps, none hydrating",
			deferFlags:      []any{ir.TDeferDetailsFlagsDefault, ir.TDeferDetailsFlagsDefault},
			expectInsertion: false,
		},
		{
			name:            "First is hydrating, second is non-hydrating",
			deferFlags:      []any{ir.TDeferDetailsFlagsHasHydrateTriggers, ir.TDeferDetailsFlagsDefault},
			expectInsertion: true,
			expectedInsertIdx: 0,
		},
		{
			name:            "First is non-hydrating, second is hydrating, third is hydrating",
			deferFlags:      []any{ir.TDeferDetailsFlagsDefault, ir.TDeferDetailsFlagsHasHydrateTriggers, ir.TDeferDetailsFlagsHasHydrateTriggers},
			expectInsertion: true,
			expectedInsertIdx: 1,
		},
		{
			name:            "Untyped int flags are correctly resolved by hasHydrateTriggers",
			deferFlags:      []any{0, int(ir.TDeferDetailsFlagsHasHydrateTriggers)},
			expectInsertion: true,
			expectedInsertIdx: 1,
		},
		{
			name:            "Nil or unrecognized flags format are handled safely",
			deferFlags:      []any{nil, "some-string-flag", ir.TDeferDetailsFlagsHasHydrateTriggers},
			expectInsertion: true,
			expectedInsertIdx: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			var originalDeferOps []*ir.DeferOp
			for i, flag := range tt.deferFlags {
				op := &ir.DeferOp{
					Xref:  ir.XrefId(100 + i),
					Flags: flag,
				}
				originalDeferOps = append(originalDeferOps, op)
				unit.GetCreate().Push(op)
			}

			InsertIncrementalHydrationRuntime(job)

			ops := unit.GetCreate().Ops

			if !tt.expectInsertion {
				if len(ops) != len(tt.deferFlags) {
					t.Fatalf("Expected no insertions, so number of ops should be %d, but got %d", len(tt.deferFlags), len(ops))
				}
				// Ensure no hydration runtime op is present
				for _, op := range ops {
					if _, ok := op.(*ir.EnableIncrementalHydrationRuntimeOp); ok {
						t.Errorf("Unexpected EnableIncrementalHydrationRuntimeOp found in ops list")
					}
				}
			} else {
				if len(ops) != len(tt.deferFlags)+1 {
					t.Fatalf("Expected 1 insertion, so number of ops should be %d, but got %d", len(tt.deferFlags)+1, len(ops))
				}

				// Find index of the EnableIncrementalHydrationRuntimeOp
				runtimeIdx := -1
				for i, op := range ops {
					if _, ok := op.(*ir.EnableIncrementalHydrationRuntimeOp); ok {
						runtimeIdx = i
						break
					}
				}

				if runtimeIdx == -1 {
					t.Fatalf("Expected EnableIncrementalHydrationRuntimeOp to be inserted, but none found")
				}

				// Check that it was inserted immediately before the expected DeferOp
				expectedTargetDeferXref := originalDeferOps[tt.expectedInsertIdx].Xref
				if runtimeIdx == len(ops)-1 {
					t.Fatalf("EnableIncrementalHydrationRuntimeOp was inserted at the very end, expected it before DeferOp with Xref %d", expectedTargetDeferXref)
				}

				nextOp := ops[runtimeIdx+1]
				nextDefer, ok := nextOp.(*ir.DeferOp)
				if !ok {
					t.Fatalf("Expected the op after the hydration activator to be DeferOp, got %T", nextOp)
				}

				if nextDefer.Xref != expectedTargetDeferXref {
					t.Errorf("Expected hydration activator to be inserted before DeferOp Xref %d, but got before Xref %d", expectedTargetDeferXref, nextDefer.Xref)
				}
			}
		})
	}
}
