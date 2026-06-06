package semantic_graph

import (
	"reflect"
)

func IsSymbolEqual(a SemanticSymbol, b SemanticSymbol) bool {
	return a.Path == b.Path && a.Identifier == b.Identifier
}

func IsReferenceEqual(a SemanticReference, b SemanticReference) bool {
	if a == nil || b == nil {
		return a == b
	}
	srcA, tgtA := a.GetSource(), a.GetTarget()
	srcB, tgtB := b.GetSource(), b.GetTarget()
	if srcA == nil || srcB == nil || tgtA == nil || tgtB == nil {
		return false
	}
	return IsSymbolEqual(*srcA, *srcB) && IsSymbolEqual(*tgtA, *tgtB)
}

func ReferenceEquality(a any, b any) bool {
	refA, okA := a.(SemanticReference)
	refB, okB := b.(SemanticReference)
	if !okA || !okB {
		return a == b
	}
	return IsReferenceEqual(refA, refB)
}

func IsArrayEqual(a any, b any, equalityTester any) bool {
	if a == nil || b == nil {
		return a == b
	}
	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)
	if valA.Kind() != reflect.Slice && valA.Kind() != reflect.Array {
		return false
	}
	if valB.Kind() != reflect.Slice && valB.Kind() != reflect.Array {
		return false
	}
	if valA.Len() != valB.Len() {
		return false
	}
	tester, ok := equalityTester.(func(any, any) bool)
	if !ok {
		for i := 0; i < valA.Len(); i++ {
			if valA.Index(i).Interface() != valB.Index(i).Interface() {
				return false
			}
		}
		return true
	}
	for i := 0; i < valA.Len(); i++ {
		if !tester(valA.Index(i).Interface(), valB.Index(i).Interface()) {
			return false
		}
	}
	return true
}

func IsSetEqual(a any, b any, equalityTester any) bool {
	if a == nil || b == nil {
		return a == b
	}
	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)
	if valA.Kind() != reflect.Map || valB.Kind() != reflect.Map {
		return false
	}
	if valA.Len() != valB.Len() {
		return false
	}
	tester, ok := equalityTester.(func(any, any) bool)
	for _, key := range valA.MapKeys() {
		valBElement := valB.MapIndex(key)
		if !valBElement.IsValid() {
			return false
		}
		if ok {
			if !tester(valA.MapIndex(key).Interface(), valBElement.Interface()) {
				return false
			}
		} else {
			if valA.MapIndex(key).Interface() != valBElement.Interface() {
				return false
			}
		}
	}
	return true
}
