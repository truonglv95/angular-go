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
			// Inserting before the first element (worst case for slice, O(N))
			list.Push(&BenchOp{id: j})
			// actually to test insertBefore we do:
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
