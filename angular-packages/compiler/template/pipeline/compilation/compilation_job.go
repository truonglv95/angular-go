package compilation

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"

	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

type CompilationJobKind int

const (
	CompilationJobKind_Tmpl CompilationJobKind = iota
	CompilationJobKind_Host
	CompilationJobKind_Both // A special value used to indicate that some logic applies to both compilation types
)

type TemplateCompilationMode int

const (
	TemplateCompilationMode_Full TemplateCompilationMode = iota
	TemplateCompilationMode_DomOnly
)

type CompilationJob interface {
	GetComponentName() string
	GetPool() any
	GetMode() TemplateCompilationMode
	GetLegacyOptionalChaining() bool

	GetKind() CompilationJobKind
	SetKind(kind CompilationJobKind)

	GetUnits() []CompilationUnit
	GetRoot() CompilationUnit
	GetFnSuffix() string

	AllocateXrefId() ir.XrefId

	GetConsts() []output.Expression
	GetConstsInitializers() []output.Statement
	AddConst(newConst output.Expression, initializers []output.Statement) ir.ConstIndex
}

type BaseCompilationJob struct {
	ComponentName          string
	Pool                   any
	Mode                   TemplateCompilationMode
	LegacyOptionalChaining bool

	Kind               CompilationJobKind
	NextXrefId         ir.XrefId
	Consts             []output.Expression
	ConstsInitializers []output.Statement
}

func (b *BaseCompilationJob) GetComponentName() string                  { return b.ComponentName }
func (b *BaseCompilationJob) GetPool() any                              { return b.Pool }
func (b *BaseCompilationJob) GetMode() TemplateCompilationMode          { return b.Mode }
func (b *BaseCompilationJob) GetLegacyOptionalChaining() bool           { return b.LegacyOptionalChaining }
func (b *BaseCompilationJob) GetKind() CompilationJobKind               { return b.Kind }
func (b *BaseCompilationJob) SetKind(kind CompilationJobKind)           { b.Kind = kind }
func (b *BaseCompilationJob) GetConsts() []output.Expression            { return b.Consts }
func (b *BaseCompilationJob) GetConstsInitializers() []output.Statement { return b.ConstsInitializers }

func (b *BaseCompilationJob) AllocateXrefId() ir.XrefId {
	id := b.NextXrefId
	b.NextXrefId++
	return id
}

func (b *BaseCompilationJob) AddConst(newConst output.Expression, initializers []output.Statement) ir.ConstIndex {
	for idx, c := range b.Consts {
		if c.IsEquivalent(newConst) {
			return ir.ConstIndex(idx)
		}
	}
	idx := len(b.Consts)
	b.Consts = append(b.Consts, newConst)
	if len(initializers) > 0 {
		b.ConstsInitializers = append(b.ConstsInitializers, initializers...)
	}
	return ir.ConstIndex(idx)
}

type ComponentCompilationJob struct {
	BaseCompilationJob

	RelativeContextFilePath string
	I18nUseExternalIds      bool
	DeferMeta               any
	AllDeferrableDepsFn     *output.ReadVarExpr
	RelativeTemplatePath    *string
	EnableDebugLocations    bool
	ForeignImports          []any

	Root             *ViewCompilationUnit
	Views            map[ir.XrefId]*ViewCompilationUnit
	ContentSelectors output.Expression
}

func NewComponentCompilationJob(
	componentName string,
	pool any,
	mode TemplateCompilationMode,
	relativeContextFilePath string,
	i18nUseExternalIds bool,
	deferMeta any,
	allDeferrableDepsFn *output.ReadVarExpr,
	relativeTemplatePath *string,
	enableDebugLocations bool,
	legacyOptionalChaining bool,
	foreignImports []any,
) *ComponentCompilationJob {
	job := &ComponentCompilationJob{
		BaseCompilationJob: BaseCompilationJob{
			ComponentName:          componentName,
			Pool:                   pool,
			Mode:                   mode,
			LegacyOptionalChaining: legacyOptionalChaining,
			Kind:                   CompilationJobKind_Tmpl,
			NextXrefId:             0,
		},
		RelativeContextFilePath: relativeContextFilePath,
		I18nUseExternalIds:      i18nUseExternalIds,
		DeferMeta:               deferMeta,
		AllDeferrableDepsFn:     allDeferrableDepsFn,
		RelativeTemplatePath:    relativeTemplatePath,
		EnableDebugLocations:    enableDebugLocations,
		ForeignImports:          foreignImports,
		Views:                   make(map[ir.XrefId]*ViewCompilationUnit),
	}
	job.Root = NewViewCompilationUnit(job, job.AllocateXrefId(), nil)
	job.Views[job.Root.GetXref()] = job.Root
	return job
}

func (j *ComponentCompilationJob) GetForeignComponent(element any) any {
	// element is t.Element, stubbing for now
	return nil
}

func (j *ComponentCompilationJob) GetUnits() []CompilationUnit {
	units := make([]CompilationUnit, 0, len(j.Views))
	for _, view := range j.Views {
		units = append(units, view)
	}
	return units
}

func (j *ComponentCompilationJob) GetRoot() CompilationUnit { return j.Root }
func (j *ComponentCompilationJob) GetFnSuffix() string      { return "Template" }

func (j *ComponentCompilationJob) AllocateView(parent ir.XrefId) *ViewCompilationUnit {
	parentPtr := &parent
	view := NewViewCompilationUnit(j, j.AllocateXrefId(), parentPtr)
	j.Views[view.GetXref()] = view
	return view
}

type CompilationUnit interface {
	GetXref() ir.XrefId
	GetCreate() *ir.OpList
	GetUpdate() *ir.OpList
	GetFunctions() []ir.Expression
	GetJob() CompilationJob
	GetFnName() *string
	SetFnName(name *string)
	GetVars() *int
	SetVars(vars *int)
	Ops() []ir.Op
}

type BaseCompilationUnit struct {
	Xref      ir.XrefId
	Create    *ir.OpList
	Update    *ir.OpList
	Functions []ir.Expression // Array of ir.ArrowFunctionExpr
	FnName    *string
	Vars      *int
}

func (b *BaseCompilationUnit) GetXref() ir.XrefId            { return b.Xref }
func (b *BaseCompilationUnit) GetCreate() *ir.OpList         { return b.Create }
func (b *BaseCompilationUnit) GetUpdate() *ir.OpList         { return b.Update }
func (b *BaseCompilationUnit) GetFunctions() []ir.Expression { return b.Functions }
func (b *BaseCompilationUnit) GetFnName() *string            { return b.FnName }
func (b *BaseCompilationUnit) SetFnName(name *string)        { b.FnName = name }
func (b *BaseCompilationUnit) GetVars() *int                 { return b.Vars }
func (b *BaseCompilationUnit) SetVars(vars *int)             { b.Vars = vars }

func (b *BaseCompilationUnit) Ops() []ir.Op {
	var ops []ir.Op
	for _, fn := range b.Functions {
		arrowFn := fn.(*ir.ArrowFunctionExpr)
		ops = append(ops, arrowFn.Ops.Elements()...)
	}
	for _, op := range b.Create.Elements() {
		ops = append(ops, op)
		if listener, ok := op.(ir.ListenerTrait); ok {
			ops = append(ops, listener.HandlerOps().Elements()...)
		}
	}
	for _, op := range b.Update.Elements() {
		ops = append(ops, op)
	}
	return ops
}

type ViewCompilationUnit struct {
	BaseCompilationUnit
	Job              *ComponentCompilationJob
	Parent           *ir.XrefId
	ContextVariables map[string]string
	Aliases          []ir.Expression // Set of ir.AliasVariable
	Decls            *int
}

func NewViewCompilationUnit(job *ComponentCompilationJob, xref ir.XrefId, parent *ir.XrefId) *ViewCompilationUnit {
	return &ViewCompilationUnit{
		BaseCompilationUnit: BaseCompilationUnit{
			Xref:   xref,
			Create: ir.NewOpList(),
			Update: ir.NewOpList(),
		},
		Job:              job,
		Parent:           parent,
		ContextVariables: make(map[string]string),
	}
}

func (v *ViewCompilationUnit) GetJob() CompilationJob { return v.Job }

type HostBindingCompilationJob struct {
	BaseCompilationJob
	Root *HostBindingCompilationUnit
}

func NewHostBindingCompilationJob(componentName string, pool any, mode TemplateCompilationMode, legacyOptionalChaining bool) *HostBindingCompilationJob {
	job := &HostBindingCompilationJob{
		BaseCompilationJob: BaseCompilationJob{
			ComponentName:          componentName,
			Pool:                   pool,
			Mode:                   mode,
			LegacyOptionalChaining: legacyOptionalChaining,
			Kind:                   CompilationJobKind_Host,
			NextXrefId:             0,
		},
	}
	job.Root = NewHostBindingCompilationUnit(job)
	return job
}

func (j *HostBindingCompilationJob) GetUnits() []CompilationUnit {
	return []CompilationUnit{j.Root}
}

func (j *HostBindingCompilationJob) GetRoot() CompilationUnit { return j.Root }
func (j *HostBindingCompilationJob) GetFnSuffix() string      { return "HostBindings" }

type HostBindingCompilationUnit struct {
	BaseCompilationUnit
	Job        *HostBindingCompilationJob
	Attributes *output.LiteralArrayExpr
}

func NewHostBindingCompilationUnit(job *HostBindingCompilationJob) *HostBindingCompilationUnit {
	return &HostBindingCompilationUnit{
		BaseCompilationUnit: BaseCompilationUnit{
			Xref:   0,
			Create: ir.NewOpList(),
			Update: ir.NewOpList(),
		},
		Job: job,
	}
}

func (h *HostBindingCompilationUnit) GetJob() CompilationJob { return h.Job }

var ParseSelectorToR3Selector func(selector *string) []any

