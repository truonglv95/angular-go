package i18n

import (
	"reflect"
	"strings"
	"testing"
)

func TestI18nAstCloneAndRecurseVisitors(t *testing.T) {
	t.Run("CloneVisitor", func(t *testing.T) {
		t.Run("should clone an AST", func(t *testing.T) {
			messages := extractMessages(`<div i18n="m|d">b{count, plural, =0 {{sex, select, male {m}}}}a</div>`, nil, nil, true)
			nodes := messages[0].Nodes
			text := strings.Join(SerializeNodes(nodes), "")
			if text != `b<ph icu name="ICU">{count, plural, =0 {[{sex, select, male {[m]}}]}}</ph>a` {
				t.Fatalf("serialized nodes = %q", text)
			}

			visitor := &CloneVisitor{}
			cloneNodes := make([]Node, len(nodes))
			for i, n := range nodes {
				cloneNodes[i] = n.Visit(visitor, nil).(Node)
			}

			if !reflect.DeepEqual(SerializeNodes(nodes), SerializeNodes(cloneNodes)) {
				t.Fatalf("clone serialization mismatch")
			}
			for i, n := range nodes {
				if !reflect.DeepEqual(n, cloneNodes[i]) {
					t.Fatalf("node %d deep equal mismatch", i)
				}
				if reflect.ValueOf(n).Pointer() == reflect.ValueOf(cloneNodes[i]).Pointer() {
					t.Fatalf("node %d was not cloned", i)
				}
			}
		})
	})

	t.Run("RecurseVisitor", func(t *testing.T) {
		t.Run("should visit all nodes", func(t *testing.T) {
			visitor := &countingRecurseVisitor{}
			container := NewContainer([]Node{
				NewText("", nil),
				NewPlaceholder("", "", nil),
				NewIcuPlaceholder(nil, "", nil),
			}, nil)
			tag := NewTagPlaceholder("", map[string]string{}, "", "", []Node{container}, false, nil, nil, nil)
			icu := NewIcu("", "", map[string]Node{"tag": tag}, []string{"tag"}, nil, "")

			icu.Visit(visitor, nil)

			if visitor.textCount != 1 {
				t.Fatalf("textCount = %d", visitor.textCount)
			}
			if visitor.phCount != 1 {
				t.Fatalf("phCount = %d", visitor.phCount)
			}
			if visitor.icuPhCount != 1 {
				t.Fatalf("icuPhCount = %d", visitor.icuPhCount)
			}
		})
	})
}

type countingRecurseVisitor struct {
	textCount  int
	phCount    int
	icuPhCount int
}

func (v *countingRecurseVisitor) VisitText(text *Text, context any) any {
	v.textCount++
	return nil
}

func (v *countingRecurseVisitor) VisitContainer(container *Container, context any) any {
	for _, child := range container.Children {
		child.Visit(v, nil)
	}
	return nil
}

func (v *countingRecurseVisitor) VisitIcu(icu *Icu, context any) any {
	for _, name := range icu.CaseOrders {
		icu.Cases[name].Visit(v, nil)
	}
	return nil
}

func (v *countingRecurseVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	return nil
}

func (v *countingRecurseVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	v.phCount++
	return nil
}

func (v *countingRecurseVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	v.icuPhCount++
	return nil
}

func (v *countingRecurseVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	return nil
}
