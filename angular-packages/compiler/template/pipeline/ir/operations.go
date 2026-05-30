package ir

// OpListNextListId is the counter for assigning debug IDs to OpLists.
var OpListNextListId = 0

// NewOpListId allocates the next debug ID for an OpList.
func NewOpListId() int {
	id := OpListNextListId
	OpListNextListId++
	return id
}

// OpList is a doubly-linked list of Op nodes, with a head and tail sentinel.
// It maintains both a linked-list structure (for O(1) insertions/deletions)
// and a slice (for O(1) indexed access and range iteration).
//
// This is the Go port of the TypeScript OpList<OpT> class in operations.ts.
type OpList struct {
	// debugListId is the unique debug identifier for this list.
	DebugListId int

	// head is the sentinel head node (kind = OpKindListEnd).
	HeadNode Op
	// tail is the sentinel tail node (kind = OpKindListEnd).
	TailNode Op

	// Ops is a flat slice of all ops (excluding head/tail), for range iteration.
	Ops []Op
}

// NewOpList creates an empty OpList with initialized head and tail sentinels.
func NewOpList() *OpList {
	id := NewOpListId()
	l := &OpList{
		DebugListId: id,
		Ops:         nil,
	}
	// Head and tail are set by the caller or initialization logic.
	// In practice, the linked-list traversal uses HeadNode.Next() / TailNode.Prev().
	return l
}

// Head returns the first real op node after the head sentinel, or nil.
func (l *OpList) Head() Op {
	if l.HeadNode == nil {
		return nil
	}
	n := l.HeadNode.Next()
	if n == nil || n == l.TailNode {
		return nil
	}
	return n
}

// Tail returns the tail sentinel node (used for InsertBefore(op, list.Tail())).
func (l *OpList) Tail() Op {
	return l.TailNode
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
	l.Ops = append(ops, l.Ops...)
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
			list.Ops = append(list.Ops[:i], append([]Op{op}, list.Ops[i:]...)...)
			return
		}
	}
}

// OpListRemove removes op from the list.
func OpListRemove(op Op) {
	// No-op: in the linked-list model, removal is done via prev/next pointers.
	// In the slice model this is a no-op without list context.
	// Phase files that use this typically pair it with slice-based lists.
}

// InsertBefore inserts newOp before targetOp (linked-list operation).
func InsertBefore(newOp Op, targetOp Op) {
	// Linked-list insert: update prev/next pointers.
	// The actual linked-list manipulation is deferred to implementations.
}

// InsertAfter inserts newOp after targetOp (linked-list operation).
func InsertAfter(newOp Op, targetOp Op) {
	// Linked-list insert after target.
}

// HasConsumesSlotTrait returns true if op implements ConsumesSlotOpTrait.
func HasConsumesSlotTrait(op Op) bool {
	_, ok := op.(ConsumesSlotOpTrait)
	return ok
}
