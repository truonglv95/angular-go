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
// Go-specific Contract & Developer Guide:
//
// 1. Performance and Design Decision:
//    Unlike the original Angular TypeScript compiler which uses a doubly-linked list for the template pipeline IR,
//    the Go compiler implements OpList using flat slices. Slices provide O(1) random-access indexing, high cache-locality
//    for range loops, and lower memory overhead, making compiler passes significantly faster.
//
// 2. Index & Slice Reference Invalidation:
//    - Slice backing arrays may be reallocated during insertions (Push, Prepend, InsertBefore).
//    - Removing or inserting elements shifts all subsequent elements, changing their indices.
//    - Consequently, do NOT hold indices or slice headers across list mutations.
//
// 3. Safe Iteration and Mutation Patterns (CRITICAL):
//    - Pattern A: Snapshot / Copy Loop (Use this when removing or replacing elements during range loops)
//      Because `range list.Elements()` copies the slice header only once at loop entry, mutating list.Ops directly
//      inside the loop can cause out-of-sync iteration (skipping elements, accessing out-of-bound indices, or reading nils).
//      Always snapshot the elements into a new slice before looping:
//
//          opsSnapshot := make([]ir.Op, len(list.Elements()))
//          copy(opsSnapshot, list.Elements())
//          for _, op := range opsSnapshot {
//              if shouldRemove(op) {
//                  list.Remove(op)
//              }
//          }
//
//    - Pattern B: Dynamic Index-based Loop (Use this for chaining/merging adjacent elements)
//      Evaluate the dynamic length of the slice in the loop condition:
//
//          for i := 0; i < len(list.Elements()); {
//              op := list.Elements()[i]
//              if canChainWithNext(op) {
//                  list.Remove(nextOp)
//                  // Do not increment i, as the elements shifted left
//              } else {
//                  i++
//              }
//          }
//
//    - Linked-list properties/assumptions (like `op.next` or `op.prev` fields on node objects) are NOT supported.
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
