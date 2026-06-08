package ir

import (
	"testing"
)

type BenchOp struct {
	id int
}

func (b *BenchOp) Kind() OpKind {
	return 0
}

func (b *BenchOp) GetKind() OpKind {
	return 0
}

func BenchmarkOpListPush(b *testing.B) {
	for i := 0; i < b.N; i++ {
		list := NewOpList()
		for j := 0; j < 1000; j++ {
			list.Push(&BenchOp{id: j})
		}
	}
}

func BenchmarkOpListInsertBefore(b *testing.B) {
	for i := 0; i < b.N; i++ {
		list := NewOpList()
		target := &BenchOp{id: -1}
		list.Push(target)
		for j := 0; j < 1000; j++ {
			OpListInsertBefore(list, &BenchOp{id: j}, target)
		}
	}
}

func BenchmarkOpListRemove(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		list := NewOpList()
		ops := make([]Op, 1000)
		for j := 0; j < 1000; j++ {
			ops[j] = &BenchOp{id: j}
		}
		list.PushSlice(ops)
		b.StartTimer()

		for j := 0; j < 1000; j++ {
			list.Remove(ops[j]) // Remove from beginning, worst case O(N)
		}
	}
}

// Representative compiler pipeline phase mutations

func BenchmarkPipelineReifyMutations(b *testing.B) {
	// Simulates the Reify phase where element start/end ops are transformed into statement ops.
	// This involves iterating over a snapshot and doing InsertBefore + Remove.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		list := NewOpList()
		ops := make([]Op, 100)
		for j := 0; j < 100; j++ {
			ops[j] = &BenchOp{id: j}
		}
		list.PushSlice(ops)
		b.StartTimer()

		// Snapshot iteration (safe slice mutation pattern)
		snapshot := make([]Op, len(list.Elements()))
		copy(snapshot, list.Elements())
		for _, op := range snapshot {
			newOp := &BenchOp{id: op.(*BenchOp).id + 1000}
			OpListInsertBefore(list, newOp, op)
			list.Remove(op)
		}
	}
}

func BenchmarkPipelineChainingMutations(b *testing.B) {
	// Simulates chaining phase where adjacent chainable instructions are merged, removing the chained ops.
	// This uses the dynamic index-based traversal loop.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		list := NewOpList()
		ops := make([]Op, 100)
		for j := 0; j < 100; j++ {
			ops[j] = &BenchOp{id: j}
		}
		list.PushSlice(ops)
		b.StartTimer()

		for idx := 0; idx < len(list.Elements()); {
			op := list.Elements()[idx]
			// Simulate chaining 3 elements at a time
			if idx > 0 && idx%3 != 0 {
				list.Remove(op)
				// Do not increment idx since elements shifted
			} else {
				idx++
			}
		}
	}
}

func BenchmarkPipelinePrependBlocks(b *testing.B) {
	// Simulates prepending small blocks of setup instructions to the beginning of the list.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list := NewOpList()
		for j := 0; j < 10; j++ {
			ops := []Op{
				&BenchOp{id: j * 2},
				&BenchOp{id: j*2 + 1},
			}
			list.Prepend(ops)
		}
	}
}

