package phases

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func TestAllocateSlots_Mutation(t *testing.T) {
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

	// Create ops that consume slots
	op1 := ir.CreateElementStartOp("div", job.AllocateXrefId(), nil, nil, nil, nil)
	op1.AddNumSlotsUsed(2) // Simulate consuming 2 slots

	op2 := ir.CreateElementStartOp("span", job.AllocateXrefId(), nil, nil, nil, nil)
	op2.AddNumSlotsUsed(1) // Simulate consuming 1 slot

	job.Root.GetCreate().Push(op1, op2)

	// Simulate mutation: insert op3 between op1 and op2
	op3 := ir.CreateElementStartOp("p", job.AllocateXrefId(), nil, nil, nil, nil)
	op3.AddNumSlotsUsed(3) // Consumes 3 slots

	// Insert op3 before op2
	ir.OpListInsertBefore(job.Root.GetCreate(), op3, op2)

	// Run slot allocation
	AllocateSlots(job)

	// Verify slots are allocated sequentially based on the mutated order: op1 -> op3 -> op2
	// op1 starts at 0, consumes 2 => next slot is 2
	// op3 starts at 2, consumes 3 => next slot is 5
	// op2 starts at 5, consumes 1 => next slot is 6

	if *op1.Handle().Slot != 0 {
		t.Errorf("expected op1 slot to be 0, got %d", *op1.Handle().Slot)
	}
	if *op3.Handle().Slot != 3 {
		t.Errorf("expected op3 slot to be 3, got %d", *op3.Handle().Slot)
	}
	if *op2.Handle().Slot != 7 {
		t.Errorf("expected op2 slot to be 7, got %d", *op2.Handle().Slot)
	}
	
	if *job.Root.Decls != 9 {
		t.Errorf("expected total slots (decls) to be 9, got %d", *job.Root.Decls)
	}
}
