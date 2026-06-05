package ir

import "sync/atomic"

// OpListNextListId is the counter for assigning debug IDs to OpLists.
var OpListNextListId atomic.Int64

// NewOpListId allocates the next debug ID for an OpList.
func NewOpListId() int {
	return int(OpListNextListId.Add(1) - 1)
}

// OpList is a collection of Op nodes.
// It maintains a slice for O(1) indexed access and range iteration.
//
// This is the Go port of the TypeScript OpList<OpT> class in operations.ts.
// We dropped the linked list implementation for performance via slices.
//
// Go-specific Contract:
// - Operations are strictly ordered by their index in the slice.
// - Inserting, prepending, or removing operations will re-allocate or shift the underlying slice.
// - References to indexes (or iterators in TS) should NOT be held across mutations, as they will be invalidated.
// - Linked-list assumptions like op.next or op.prev do not exist and are not supported.
type OpList struct {
	// DebugListId is the unique debug identifier for this list.
	DebugListId int

	// Ops is a flat slice of all ops, for range iteration.
	Ops []Op
}

// NewOpList creates an empty OpList.
func NewOpList() *OpList {
	id := NewOpListId()
	return &OpList{
		DebugListId: id,
		Ops:         nil,
	}
}

// Elements returns all ops as a slice (excluding head/tail sentinels).
func (l *OpList) Elements() []Op {
	return l.Ops
}

// Push appends one or more ops to the end of the list.
func (l *OpList) Push(ops ...Op) {
	for _, op := range ops {
		l.Ops = append(l.Ops, op)
	}
}

// PushSlice appends a slice of ops to the end of the list.
func (l *OpList) PushSlice(ops []Op) {
	l.Ops = append(l.Ops, ops...)
}

// Prepend inserts ops at the beginning of the slice.
func (l *OpList) Prepend(ops []Op) {
	l.Ops = append(l.Ops, ops...)
	copy(l.Ops[len(ops):], l.Ops[:len(l.Ops)-len(ops)])
	copy(l.Ops[:len(ops)], ops)
}

// Remove removes an op from the list.
func (l *OpList) Remove(target Op) {
	for i, op := range l.Ops {
		if op == target {
			l.Ops = append(l.Ops[:i], l.Ops[i+1:]...)
			return
		}
	}
}

// OpListInsertBefore inserts op before target in the list.
func OpListInsertBefore(list *OpList, op Op, target Op) {
	for i, o := range list.Ops {
		if o == target {
			list.Ops = append(list.Ops, nil)
			copy(list.Ops[i+1:], list.Ops[i:len(list.Ops)-1])
			list.Ops[i] = op
			return
		}
	}
	list.Ops = append(list.Ops, op)
}

// HasConsumesSlotTrait returns true if op implements ConsumesSlotOpTrait.
func HasConsumesSlotTrait(op Op) bool {
	_, ok := op.(ConsumesSlotOpTrait)
	return ok
}
