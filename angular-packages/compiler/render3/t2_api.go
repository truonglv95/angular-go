package render3

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

// ScopedNode Node that has a `Scope` associated with it.
type ScopedNode interface{} // Template | SwitchBlockCaseGroup | IfBlockBranch | ForLoopBlock | ForLoopBlockEmpty | DeferredBlock | DeferredBlockError | DeferredBlockLoading | DeferredBlockPlaceholder | Content | HostElement

// ReferenceTarget Possible values that a reference can be resolved to.
type ReferenceTarget[DirectiveT any] struct {
	Directive DirectiveT
	Node      DirectiveOwner
	Element   *Element
	Template  *Template
}

// TemplateEntity Entity that is local to the template and defined within the template.
type TemplateEntity interface {
	GetName() string
}

// DirectiveOwner Nodes that can have directives applied to them.
type DirectiveOwner interface{} // Element | Template | Component | Directive | HostElement

// ConflictingHostDirectiveBinding Information about a host directive binding that was exposed under conflicting aliases.
type ConflictingHostDirectiveBinding[DirectiveT any] struct {
	Directive          DirectiveT
	ClassPropertyName  string
	ConflictingAliases map[string]bool
	Kind               string // 'input' | 'output'
}

// Target A logical target for analysis, which could contain a template or other types of bindings.
type Target[DirectiveT any] struct {
	Template []Node
	Host     *TargetHost[DirectiveT]
}

type TargetHost[DirectiveT any] struct {
	Node       *HostElement
	Directives []DirectiveT
}

// LegacyAnimationTriggerNames A data structure which captures the animation trigger names that are statically resolvable
type LegacyAnimationTriggerNames struct {
	IncludesDynamicAnimations bool
	StaticTriggerNames        []string
}

type MatchSource int

const (
	MatchSourceSelector MatchSource = iota
	MatchSourceHostDirective
)

// DirectiveMeta Metadata regarding a directive that's needed to match it against template elements.
type DirectiveMeta interface {
	GetName() string
	GetRefKey() string
	GetSelector() *string
	IsComponent() bool
	GetInputs() any
	GetOutputs() any
	GetExportAs() []string
	IsStructural() bool
	GetNgContentSelectors() []string
	GetPreserveWhitespaces() bool
	GetAnimationTriggerNames() *LegacyAnimationTriggerNames
	GetMatchSource() MatchSource
}

// ForeignComponentMeta Metadata regarding a foreign component that's needed to match it against template elements.
type ForeignComponentMeta struct {
	Name string
	Ref  ForeignComponentMetaRef
}

type ForeignComponentMetaRef struct {
	Key string
}

// TargetBinder Interface to the binding API.
type TargetBinder[D DirectiveMeta] interface {
	Bind(target Target[D]) BoundTarget[D]
}

// BoundTarget Result of performing the binding operation against a `Target`.
type BoundTarget[DirectiveT DirectiveMeta] interface {
	GetTarget() Target[DirectiveT]
	GetDirectivesOfNode(node DirectiveOwner) []DirectiveT
	GetForeignComponent(element *Element) *ForeignComponentMeta
	GetReferenceTarget(ref *Reference) *ReferenceTarget[DirectiveT]
	GetConsumerOfBinding(binding any) any // DirectiveT | Element | Template
	GetExpressionTarget(expr expression_parser.AST) TemplateEntity
	GetDefinitionNodeOfSymbol(symbol TemplateEntity) ScopedNode
	GetNestingLevel(node ScopedNode) int
	GetEntitiesInScope(node ScopedNode) []TemplateEntity
	GetUsedDirectives() []DirectiveT
	GetEagerlyUsedDirectives() []DirectiveT
	GetUsedPipes() []string
	GetEagerlyUsedPipes() []string
	GetDeferBlocks() []*DeferredBlock
	GetDeferredTriggerTarget(block *DeferredBlock, trigger DeferredTrigger) *Element
	IsDeferred(node *Element) bool
	ReferencedDirectiveExists(name string) bool
	GetConflictingHostDirectiveBindings(node DirectiveOwner) []ConflictingHostDirectiveBinding[DirectiveT]
}
