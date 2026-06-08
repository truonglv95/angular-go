package ir

import (
	"testing"
)

type DummyOp struct {
	id int
}

func (d *DummyOp) Kind() OpKind {
	return 0
}

func (d *DummyOp) GetKind() OpKind {
	return 0
}

func TestOpListMutations(t *testing.T) {
	t.Run("Push and Elements", func(t *testing.T) {
		list := NewOpList()
		op1 := &DummyOp{id: 1}
		op2 := &DummyOp{id: 2}

		list.Push(op1, op2)

		elems := list.Elements()
		if len(elems) != 2 || elems[0] != op1 || elems[1] != op2 {
			t.Errorf("expected [op1, op2], got %v", elems)
		}
	})

	t.Run("Prepend", func(t *testing.T) {
		list := NewOpList()
		op1 := &DummyOp{id: 1}
		op2 := &DummyOp{id: 2}
		op3 := &DummyOp{id: 3}

		list.Push(op3)
		list.Prepend([]Op{op1, op2})

		elems := list.Elements()
		if len(elems) != 3 || elems[0] != op1 || elems[1] != op2 || elems[2] != op3 {
			t.Errorf("expected [op1, op2, op3], got %v", elems)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		list := NewOpList()
		op1 := &DummyOp{id: 1}
		op2 := &DummyOp{id: 2}
		op3 := &DummyOp{id: 3}

		list.Push(op1, op2, op3)
		list.Remove(op2)

		elems := list.Elements()
		if len(elems) != 2 || elems[0] != op1 || elems[1] != op3 {
			t.Errorf("expected [op1, op3], got %v", elems)
		}
	})

	t.Run("InsertBefore", func(t *testing.T) {
		list := NewOpList()
		op1 := &DummyOp{id: 1}
		op2 := &DummyOp{id: 2}
		op3 := &DummyOp{id: 3}

		list.Push(op1, op3)
		OpListInsertBefore(list, op2, op3)

		elems := list.Elements()
		if len(elems) != 3 || elems[0] != op1 || elems[1] != op2 || elems[2] != op3 {
			t.Errorf("expected [op1, op2, op3], got %v", elems)
		}
	})
	
	t.Run("InsertBefore target not found appends", func(t *testing.T) {
		list := NewOpList()
		op1 := &DummyOp{id: 1}
		op2 := &DummyOp{id: 2}
		op3 := &DummyOp{id: 3}

		list.Push(op1, op2)
		OpListInsertBefore(list, op3, &DummyOp{id: 99})

		elems := list.Elements()
		if len(elems) != 3 || elems[0] != op1 || elems[1] != op2 || elems[2] != op3 {
			t.Errorf("expected [op1, op2, op3], got %v", elems)
		}
	})
}

// Invariant tests for op mutations and slice safety

type DummySlotConsumingOp struct {
	xref       XrefId
	slotHandle *SlotHandle
	numSlots   int
}

func (d *DummySlotConsumingOp) Kind() OpKind {
	return OpKindElementStart
}

func (d *DummySlotConsumingOp) GetKind() OpKind {
	return OpKindElementStart
}

func (d *DummySlotConsumingOp) GetXref() XrefId {
	return d.xref
}

func (d *DummySlotConsumingOp) Handle() *SlotHandle {
	return d.slotHandle
}

func (d *DummySlotConsumingOp) AddNumSlotsUsed(n int) {
	d.numSlots += n
}

func (d *DummySlotConsumingOp) GetNumSlotsUsed() int {
	return d.numSlots
}

func TestOpListInvariant_SlotAllocationConsistency(t *testing.T) {
	list := NewOpList()

	op1 := &DummySlotConsumingOp{xref: 1, slotHandle: NewSlotHandle(), numSlots: 2}
	op2 := &DummySlotConsumingOp{xref: 2, slotHandle: NewSlotHandle(), numSlots: 1}
	op3 := &DummySlotConsumingOp{xref: 3, slotHandle: NewSlotHandle(), numSlots: 3}
	op4 := &DummySlotConsumingOp{xref: 4, slotHandle: NewSlotHandle(), numSlots: 4}

	// 1. Prepend and Push mutations
	list.Push(op2)
	list.Prepend([]Op{op1}) // list is now [op1, op2]
	list.Push(op4)          // list is now [op1, op2, op4]

	// 2. InsertBefore mutation
	OpListInsertBefore(list, op3, op4) // list is now [op1, op2, op3, op4]

	// 3. Remove mutation
	list.Remove(op2) // list is now [op1, op3, op4]

	// Run simulated slot allocation on the final mutated list
	slotCount := 0
	for _, op := range list.Elements() {
		if trait, ok := op.(ConsumesSlotTrait); ok {
			val := slotCount
			trait.Handle().Slot = &val
			slotCount += trait.GetNumSlotsUsed()
		}
	}

	// Verify order correctness and slot values:
	// op1: starts at 0, consumes 2 => next slot is 2
	// op3: starts at 2, consumes 3 => next slot is 5
	// op4: starts at 5, consumes 4 => next slot is 9
	elems := list.Elements()
	if len(elems) != 3 || elems[0] != op1 || elems[1] != op3 || elems[2] != op4 {
		t.Fatalf("unexpected list order: %v", elems)
	}

	if *op1.slotHandle.Slot != 0 {
		t.Errorf("expected op1 slot 0, got %d", *op1.slotHandle.Slot)
	}
	if *op3.slotHandle.Slot != 2 {
		t.Errorf("expected op3 slot 2, got %d", *op3.slotHandle.Slot)
	}
	if *op4.slotHandle.Slot != 5 {
		t.Errorf("expected op4 slot 5, got %d", *op4.slotHandle.Slot)
	}
	if slotCount != 9 {
		t.Errorf("expected total slot count 9, got %d", slotCount)
	}
}

func TestOpListInvariant_NestedEmbeddedViewOrdering(t *testing.T) {
	// Verify template/conditional child view ops can be inserted, removed and ordered
	list := NewOpList()

	t1 := &TemplateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:       10,
				SlotHandle: NewSlotHandle(),
			},
		},
	}
	t2 := &TemplateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:       20,
				SlotHandle: NewSlotHandle(),
			},
		},
	}
	t3 := &TemplateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:       30,
				SlotHandle: NewSlotHandle(),
			},
		},
	}

	// Push and mutate list of view ops
	list.Push(t1, t3)
	OpListInsertBefore(list, t2, t3) // order: t1, t2, t3

	elems := list.Elements()
	if len(elems) != 3 || elems[0] != t1 || elems[1] != t2 || elems[2] != t3 {
		t.Fatalf("expected template ops in order [t1, t2, t3], got %v", elems)
	}

	// Remove middle view
	list.Remove(t2) // order: t1, t3
	elems = list.Elements()
	if len(elems) != 2 || elems[0] != t1 || elems[1] != t3 {
		t.Fatalf("expected template ops in order [t1, t3], got %v", elems)
	}
}

func TestOpListInvariant_ListenerHandlerOrdering(t *testing.T) {
	// Verify listener handler inner ops are maintained, prepended, inserted and deleted correctly.
	listener := &ListenerOp{
		HandlerOps: NewOpList(),
	}

	ops := listener.GetHandlerOps()
	h1 := &DummyOp{id: 101}
	h2 := &DummyOp{id: 102}
	h3 := &DummyOp{id: 103}

	ops.Push(h2, h3)
	ops.Prepend([]Op{h1}) // order: h1, h2, h3

	elems := ops.Elements()
	if len(elems) != 3 || elems[0] != h1 || elems[1] != h2 || elems[2] != h3 {
		t.Fatalf("expected handler ops in order [h1, h2, h3], got %v", elems)
	}

	ops.Remove(h2) // order: h1, h3
	elems = ops.Elements()
	if len(elems) != 2 || elems[0] != h1 || elems[1] != h3 {
		t.Fatalf("expected handler ops in order [h1, h3], got %v", elems)
	}
}

func TestOpListInvariant_DeferChildViewOrdering(t *testing.T) {
	// Verify defer op correctly maps sub-views
	deferOp := &DeferOp{
		MainView:        40,
		LoadingView:     41,
		PlaceholderView: 42,
		ErrorView:       43,
	}

	if deferOp.MainView != 40 {
		t.Errorf("expected MainView 40, got %d", deferOp.MainView)
	}
	if deferOp.LoadingView != 41 {
		t.Errorf("expected LoadingView 41, got %d", deferOp.LoadingView)
	}
	if deferOp.PlaceholderView != 42 {
		t.Errorf("expected PlaceholderView 42, got %d", deferOp.PlaceholderView)
	}
	if deferOp.ErrorView != 43 {
		t.Errorf("expected ErrorView 43, got %d", deferOp.ErrorView)
	}
}

