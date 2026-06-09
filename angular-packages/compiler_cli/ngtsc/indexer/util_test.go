package indexer

import (
	"fmt"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/internal/ast"
)

type mockDirectiveMeta struct {
	name                  string
	refKey                string
	selector              *string
	isComponent           bool
	inputs                any
	outputs               any
	exportAs              []string
	isStructural          bool
	ngContentSelectors    []string
	preserveWhitespaces   bool
	animationTriggerNames *render3.LegacyAnimationTriggerNames
	matchSource           render3.MatchSource

	node *ast.Node
}

func (m *mockDirectiveMeta) GetName() string                                        { return m.name }
func (m *mockDirectiveMeta) GetRefKey() string                                      { return m.refKey }
func (m *mockDirectiveMeta) GetSelector() *string                                   { return m.selector }
func (m *mockDirectiveMeta) IsComponent() bool                                      { return m.isComponent }
func (m *mockDirectiveMeta) GetInputs() any                                         { return m.inputs }
func (m *mockDirectiveMeta) GetOutputs() any                                        { return m.outputs }
func (m *mockDirectiveMeta) GetExportAs() []string                                  { return m.exportAs }
func (m *mockDirectiveMeta) IsStructural() bool                                     { return m.isStructural }
func (m *mockDirectiveMeta) GetNgContentSelectors() []string                        { return m.ngContentSelectors }
func (m *mockDirectiveMeta) GetPreserveWhitespaces() bool                           { return m.preserveWhitespaces }
func (m *mockDirectiveMeta) GetAnimationTriggerNames() *render3.LegacyAnimationTriggerNames { return m.animationTriggerNames }
func (m *mockDirectiveMeta) GetMatchSource() render3.MatchSource                    { return m.matchSource }

type testReferenceTarget struct {
	node      any
	directive *ast.Node
}

func (r *testReferenceTarget) GetNode() any {
	return r.node
}

func (r *testReferenceTarget) GetDirectiveNode() *ast.Node {
	return r.directive
}

type testBoundTemplate struct {
	boundTarget render3.BoundTarget[render3.DirectiveMeta]
	templateAst []any
}

func (t *testBoundTemplate) GetDirectivesOfNode(node any) []DirectiveReference {
	var r3Node render3.DirectiveOwner
	switch n := node.(type) {
	case *render3.Element:
		r3Node = n
	case *render3.Template:
		r3Node = n
	case *render3.Component:
		r3Node = n
	case *render3.Directive:
		r3Node = n
	default:
		return nil
	}
	r3Dirs := t.boundTarget.GetDirectivesOfNode(r3Node)
	var dirs []DirectiveReference
	for _, dir := range r3Dirs {
		var declNode *ast.Node
		if mock, ok := dir.(*mockDirectiveMeta); ok {
			declNode = mock.node
		}
		selector := ""
		if dir.GetSelector() != nil {
			selector = *dir.GetSelector()
		}
		dirs = append(dirs, DirectiveReference{
			Node:     declNode,
			Selector: selector,
		})
	}
	return dirs
}

func (t *testBoundTemplate) GetReferenceTarget(node any) any {
	ref, ok := node.(*render3.Reference)
	if !ok {
		return nil
	}
	target := t.boundTarget.GetReferenceTarget(ref)
	fmt.Printf("DEBUG GetReferenceTarget: ref.Name=%s, target=%+v, target.Directive=%+v\n", ref.Name, target, target.Directive)
	if target == nil {
		return nil
	}
	if target.Directive != nil {
		var directiveNode *ast.Node
		if mock, ok := target.Directive.(*mockDirectiveMeta); ok {
			directiveNode = mock.node
		}
		var r3Node any
		if target.Element != nil {
			r3Node = target.Element
		} else if target.Template != nil {
			r3Node = target.Template
		} else {
			r3Node = target.Node
		}
		return &testReferenceTarget{
			node:      r3Node,
			directive: directiveNode,
		}
	}
	if target.Element != nil {
		return target.Element
	}
	if target.Template != nil {
		return target.Template
	}
	return target.Node
}

func (t *testBoundTemplate) GetExpressionTarget(astNode any) any {
	expr, ok := astNode.(expression_parser.AST)
	if !ok {
		return nil
	}
	return t.boundTarget.GetExpressionTarget(expr)
}

func (t *testBoundTemplate) GetUsedDirectives() []DirectiveReference {
	r3Dirs := t.boundTarget.GetUsedDirectives()
	var dirs []DirectiveReference
	for _, dir := range r3Dirs {
		var declNode *ast.Node
		if mock, ok := dir.(*mockDirectiveMeta); ok {
			declNode = mock.node
		}
		selector := ""
		if dir.GetSelector() != nil {
			selector = *dir.GetSelector()
		}
		dirs = append(dirs, DirectiveReference{
			Node:     declNode,
			Selector: selector,
		})
	}
	return dirs
}

func (t *testBoundTemplate) GetTemplateAst() []any {
	return t.templateAst
}

type bindComponent struct {
	selector    string
	declaration *ast.Node
}

func getComponentDeclaration(t *testing.T, componentStr string, className string) (*ast.Node, func()) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name:     "/TEST_FILE.ts",
			Contents: componentStr,
		},
	})
	sf := ngtsctest.RequireSourceFile(t, result, "/TEST_FILE.ts")
	node := ngtsctest.FindNamedClassDeclaration(sf, className)
	if node == nil {
		t.Fatalf("Class declaration %q not found in test source", className)
	}
	return node, result.Release
}

func getBoundTemplate(t *testing.T, template string, options *render3.ParseTemplateOptions, components []bindComponent) AbstractBoundTemplate {
	parsed := render3.ParseTemplate(template, "/TEST_FILE.ts", options)

	var matcher any
	var componentsMeta []render3.DirectiveMeta
	for _, comp := range components {
		name := ""
		if classDecl := comp.declaration.AsClassDeclaration(); classDecl != nil && classDecl.Name() != nil {
			name = classDecl.Name().AsIdentifier().Text
		}
		meta := &mockDirectiveMeta{
			name:        name,
			selector:    &comp.selector,
			isComponent: true,
			inputs:      render3.ClassPropertyMappingFromMappedObject(nil),
			outputs:     render3.ClassPropertyMappingFromMappedObject(nil),
			matchSource: render3.MatchSourceSelector,
			node:        comp.declaration,
		}
		componentsMeta = append(componentsMeta, meta)
	}

	if options != nil && options.EnableSelectorless != nil && *options.EnableSelectorless {
		registry := make(map[string][]render3.DirectiveMeta)
		for _, meta := range componentsMeta {
			registry[meta.GetName()] = []render3.DirectiveMeta{meta}
		}
		matcher = render3.NewSelectorlessMatcher[render3.DirectiveMeta](registry)
	} else {
		sm := render3.NewSelectorMatcher[[]render3.DirectiveMeta]()
		for _, meta := range componentsMeta {
			if meta.GetSelector() != nil {
				sm.AddSelectables(render3.CssSelectorParse(*meta.GetSelector()), []render3.DirectiveMeta{meta})
			}
		}
		matcher = sm
	}

	nodes := parsed.Nodes
	if nodes == nil {
		nodes = []render3.Node{}
	}

	binder := render3.NewR3TargetBinder[render3.DirectiveMeta](matcher, nil)
	bound := binder.Bind(render3.Target[render3.DirectiveMeta]{Template: nodes})

	var templateAst []any
	for _, n := range nodes {
		templateAst = append(templateAst, n)
	}

	return &testBoundTemplate{
		boundTarget: bound,
		templateAst: templateAst,
	}
}

func findSourceFileNode(node *ast.Node) *ast.SourceFile {
	parent := node
	for parent != nil {
		if ast.IsSourceFile(parent) {
			return parent.AsSourceFile()
		}
		parent = parent.Parent
	}
	return nil
}

type nodeAdapter struct{}

func (nodeAdapter) GetName(node *ast.Node) string {
	if classDecl := node.AsClassDeclaration(); classDecl != nil && classDecl.Name() != nil {
		return classDecl.Name().AsIdentifier().Text
	}
	return ""
}

func (nodeAdapter) GetFileName(node *ast.Node) string {
	sf := findSourceFileNode(node)
	if sf != nil {
		return sf.FileName()
	}
	return ""
}

func (nodeAdapter) GetContent(node *ast.Node) string {
	sf := findSourceFileNode(node)
	if sf != nil {
		return sf.Text()
	}
	return ""
}
