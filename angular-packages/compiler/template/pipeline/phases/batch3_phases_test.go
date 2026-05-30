package phases

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func TestGeneratePureLiteralStructures(t *testing.T) {
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

	// Create a non-constant expression
	readVar := output.NewReadVarExpr("myVar", nil, nil, nil)

	// A LiteralArrayExpr: [1, myVar]
	literalArr := output.NewLiteralArrayExpr([]output.Expression{
		output.NewLiteralExpr(1, nil, nil, nil),
		readVar,
	}, nil, nil, nil)

	// An update op that carries the expression
	updateOp := &ir.PropertyOp{
		Target:      10,
		Name:        "prop",
		Expression:  literalArr,
		SecurityContext: 0,
	}

	unit.GetUpdate().Push(updateOp)

	GeneratePureLiteralStructures(job)

	// Check that the expression in updateOp is transformed into a PureFunctionExpr
	propOp, ok := unit.GetUpdate().Ops[0].(*ir.PropertyOp)
	if !ok {
		t.Fatalf("Expected PropertyOp, got %T", unit.GetUpdate().Ops[0])
	}

	pureFn, ok := propOp.Expression.(*ir.PureFunctionExpr)
	if !ok {
		t.Fatalf("Expected PropertyOp Expression to be a PureFunctionExpr, got %T", propOp.Expression)
	}

	if len(pureFn.Args) != 1 {
		t.Errorf("Expected 1 argument in PureFunctionExpr, got %d", len(pureFn.Args))
	} else {
		if readVarExpr, ok := pureFn.Args[0].(*output.ReadVarExpr); !ok || readVarExpr.Name != "myVar" {
			t.Errorf("Expected argument to be ReadVarExpr 'myVar', got %T", pureFn.Args[0])
		}
	}

	arrBody, ok := pureFn.Body.(*output.LiteralArrayExpr)
	if !ok {
		t.Fatalf("Expected PureFunctionExpr Body to be LiteralArrayExpr, got %T", pureFn.Body)
	}

	if len(arrBody.Entries) != 2 {
		t.Fatalf("Expected 2 entries in transformed LiteralArrayExpr, got %d", len(arrBody.Entries))
	}

	// First entry (constant 1) should remain constant
	constEntry, ok := arrBody.Entries[0].(*output.LiteralExpr)
	if !ok || constEntry.Value != 1 {
		t.Errorf("Expected first entry to be constant 1, got %T", arrBody.Entries[0])
	}

	// Second entry (non-constant myVar) should become a PureFunctionParameterExpr
	paramEntry, ok := arrBody.Entries[1].(*ir.PureFunctionParameterExpr)
	if !ok || paramEntry.Index != 0 {
		t.Errorf("Expected second entry to be PureFunctionParameterExpr at index 0, got %T", arrBody.Entries[1])
	}
}

type mockConstPool struct {
	literals []output.Expression
}

func (m *mockConstPool) GetSharedConstant(key any, expr output.Expression) output.Expression {
	return expr
}

func (m *mockConstPool) UniqueName(name string) string {
	return name
}

func (m *mockConstPool) GetConstLiteral(literal output.Expression, share bool) output.Expression {
	m.literals = append(m.literals, literal)
	return literal
}

func TestGenerateProjectionDefs(t *testing.T) {
	pool := &mockConstPool{}
	job := compilation.NewComponentCompilationJob(
		"TestComponent",
		pool,
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

	selector := "div"
	projectionOp := &ir.ProjectionOp{
		Selector: &selector,
	}

	unit.GetCreate().Push(projectionOp)

	// Wire up ParseSelectorToR3Selector stub for testing
	compilation.ParseSelectorToR3Selector = func(s *string) []any {
		return []any{*s}
	}

	GenerateProjectionDefs(job)

	// Verify projection Slot index is assigned
	if projectionOp.ProjectionSlotIndex == nil || *projectionOp.ProjectionSlotIndex != 0 {
		t.Errorf("Expected ProjectionSlotIndex to be 0, got %v", projectionOp.ProjectionSlotIndex)
	}

	// Verify that a ProjectionDefOp was prepended to root view creation sequence
	createOps := unit.GetCreate().Ops
	if len(createOps) != 2 {
		t.Fatalf("Expected 2 create ops, got %d", len(createOps))
	}

	_, ok := createOps[0].(*ir.ProjectionDefOp)
	if !ok {
		t.Errorf("Expected first create op to be ProjectionDefOp, got %T", createOps[0])
	}
}

func TestGenerateLocalLetReferences(t *testing.T) {
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

	storeLetOp := &ir.StoreLetOp{
		DeclaredName: "myLet",
		Target:       100,
		Value:        output.NewLiteralExpr("val", nil, nil, nil),
	}

	unit.GetUpdate().Push(storeLetOp)

	GenerateLocalLetReferences(job)

	// Verify that StoreLetOp is replaced by a VariableOp
	updateOps := unit.GetUpdate().Ops
	if len(updateOps) != 1 {
		t.Fatalf("Expected 1 update op, got %d", len(updateOps))
	}

	varOp, ok := updateOps[0].(*ir.VariableOp)
	if !ok {
		t.Fatalf("Expected update op to be VariableOp, got %T", updateOps[0])
	}

	if varOp.Variable.Kind != ir.SemanticVariableKindIdentifier || varOp.Variable.Identifier != "myLet" || !varOp.Variable.Local {
		t.Errorf("Expected VariableOp to define local identifier 'myLet', got %+v", varOp.Variable)
	}

	storeLetExpr, ok := varOp.Initializer.(*ir.StoreLetExpr)
	if !ok {
		t.Errorf("Expected VariableOp initializer to be StoreLetExpr, got %T", varOp.Initializer)
	} else if storeLetExpr.Target != 100 {
		t.Errorf("Expected StoreLetExpr target to be 100, got %d", storeLetExpr.Target)
	}
}

func TestGenerateVariables(t *testing.T) {
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

	// Setup context variables on the unit
	unit.ContextVariables["myVar"] = "myVarProp"

	// Create an ElementStartOp with a local ref
	localRefs := []ir.LocalRef{
		{Name: "myRef", Target: ""},
	}
	elementOp := &ir.ElementStartOp{
		ElementOpBase: ir.ElementOpBase{
			ElementOrContainerOpBase: ir.ElementOrContainerOpBase{
				Xref:           10,
				LocalRefsField: localRefs,
			},
		},
	}

	unit.GetCreate().Push(elementOp)

	GenerateVariables(job)

	// Verify that variables are prepended to the update block
	updateOps := unit.GetUpdate().Ops
	if len(updateOps) < 2 {
		t.Fatalf("Expected at least 2 variable operations prepended, got %d", len(updateOps))
	}

	// Verify the prepended context variable read
	var1, ok := updateOps[0].(*ir.VariableOp)
	if !ok {
		t.Fatalf("Expected first op to be VariableOp, got %T", updateOps[0])
	}
	if var1.Variable.Kind != ir.SemanticVariableKindIdentifier || var1.Variable.Identifier != "myVar" {
		t.Errorf("Expected first variable to read 'myVar', got %+v", var1.Variable)
	}

	// Verify the prepended reference variable read
	var2, ok := updateOps[1].(*ir.VariableOp)
	if !ok {
		t.Fatalf("Expected second op to be VariableOp, got %T", updateOps[1])
	}
	if var2.Variable.Kind != ir.SemanticVariableKindIdentifier || var2.Variable.Identifier != "myRef" {
		t.Errorf("Expected second variable to read 'myRef', got %+v", var2.Variable)
	}
}
