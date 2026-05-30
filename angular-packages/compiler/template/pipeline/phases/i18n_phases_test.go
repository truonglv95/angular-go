package phases

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func TestI18nPhases(t *testing.T) {
	// Initialize a mock ComponentCompilationJob
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

	rootUnit := job.Root

	// 1. Test WrapI18nIcus
	// Place an ICU outside of any i18n block in the root unit
	icuMsg := &i18n.Message{Id: "icu_msg"}
	icuStartXref := job.AllocateXrefId()
	icuStart := &ir.IcuStartOp{
		Xref:    icuStartXref,
		Message: icuMsg,
	}
	icuEnd := &ir.IcuEndOp{
		Xref: icuStartXref,
	}

	rootUnit.GetCreate().Push(icuStart, icuEnd)

	// Run WrapI18nIcus phase
	WrapI18nIcus(job)

	// Verify that I18nStartOp and I18nEndOp were inserted around the ICU ops
	ops := rootUnit.GetCreate().Ops
	if len(ops) != 4 {
		t.Fatalf("Expected 4 operations after WrapI18nIcus, got %d", len(ops))
	}

	i18nStart, ok := ops[0].(*ir.I18nStartOp)
	if !ok {
		t.Fatalf("Expected first op to be *ir.I18nStartOp, got %T", ops[0])
	}
	if i18nStart.Message != icuMsg {
		t.Errorf("Expected wrapped i18n message to match ICU message")
	}

	_, ok = ops[3].(*ir.I18nEndOp)
	if !ok {
		t.Fatalf("Expected last op to be *ir.I18nEndOp, got %T", ops[3])
	}

	// 2. Test PropagateI18nBlocks
	// Let's create a nested view compilation unit and a template creating op
	nestedView := job.AllocateView(rootUnit.GetXref())
	templateXref := nestedView.GetXref()

	templateOp := &ir.TemplateOp{
		ElementOpBase: ir.ElementOpBase{
			ElementOrContainerOpBase: ir.ElementOrContainerOpBase{
				Xref: templateXref,
			},
		},
		I18nPlaceholder: &struct{}{}, // non-nil to trigger propagation
	}

	// Add the TemplateOp inside the i18n block we wrapped
	rootUnit.GetCreate().Ops = []ir.Op{ops[0], ops[1], templateOp, ops[2], ops[3]}

	// Run PropagateI18nBlocks
	PropagateI18nBlocks(job)

	// Verify that the child/nested view was wrapped with i18n start and end ops
	nestedOps := nestedView.GetCreate().Ops
	if len(nestedOps) < 2 {
		t.Fatalf("Expected nested view to have been wrapped with i18n ops, got %d ops", len(nestedOps))
	}

	nestedI18nStart, ok := nestedOps[0].(*ir.I18nStartOp)
	if !ok {
		t.Fatalf("Expected first nested op to be *ir.I18nStartOp, got %T", nestedOps[0])
	}
	if nestedI18nStart.Message != icuMsg {
		t.Errorf("Expected nested i18n message to propagate from parent i18n block")
	}

	// 3. Test CreateI18nContexts
	// Run CreateI18nContexts
	CreateI18nContexts(job)

	// Verify that root i18n block got ir.I18nContextKindRootI18n context
	// And nested child view block inherited the same context
	if i18nStart.Context == 0 {
		t.Errorf("Expected root I18nStartOp to have a non-zero context link")
	}
	if nestedI18nStart.Context != i18nStart.Context {
		t.Errorf("Expected child/nested I18nStartOp to inherit the context from root, got %d, want %d", nestedI18nStart.Context, i18nStart.Context)
	}
}
