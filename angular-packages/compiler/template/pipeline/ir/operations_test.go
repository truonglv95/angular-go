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
