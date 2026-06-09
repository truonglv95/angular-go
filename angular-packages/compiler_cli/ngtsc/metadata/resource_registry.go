package metadata

import (
	"sync"
	"github.com/microsoft/typescript-go/internal/ast"
)

type Resource struct {
	Path string // Empty if inline
	Node *ast.Node
}

type DirectiveResources struct {
	Template     *Resource
	Styles       []*Resource
	HostBindings []*Resource
}

type ResourceRegistry struct {
	mu                               sync.RWMutex
	externalTemplateToComponentsMap  map[string]map[*ast.Node]bool
	componentToTemplateMap           map[*ast.Node]*Resource
	componentToStylesMap             map[*ast.Node][]*Resource
	externalStyleToComponentsMap     map[string]map[*ast.Node]bool
	directiveToHostBindingsMap       map[*ast.Node][]*Resource
}

func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		externalTemplateToComponentsMap:  make(map[string]map[*ast.Node]bool),
		componentToTemplateMap:           make(map[*ast.Node]*Resource),
		componentToStylesMap:             make(map[*ast.Node][]*Resource),
		externalStyleToComponentsMap:     make(map[string]map[*ast.Node]bool),
		directiveToHostBindingsMap:       make(map[*ast.Node][]*Resource),
	}
}

func (r *ResourceRegistry) GetComponentsWithTemplate(templatePath string) []*ast.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	comps, ok := r.externalTemplateToComponentsMap[templatePath]
	if !ok {
		return nil
	}
	var res []*ast.Node
	for c := range comps {
		res = append(res, c)
	}
	return res
}

func (r *ResourceRegistry) RegisterResources(resources DirectiveResources, directive *ast.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if resources.Template != nil {
		path := resources.Template.Path
		if path != "" {
			if _, ok := r.externalTemplateToComponentsMap[path]; !ok {
				r.externalTemplateToComponentsMap[path] = make(map[*ast.Node]bool)
			}
			r.externalTemplateToComponentsMap[path][directive] = true
		}
		r.componentToTemplateMap[directive] = resources.Template
	}

	if resources.Styles != nil {
		r.componentToStylesMap[directive] = resources.Styles
		for _, style := range resources.Styles {
			path := style.Path
			if path != "" {
				if _, ok := r.externalStyleToComponentsMap[path]; !ok {
					r.externalStyleToComponentsMap[path] = make(map[*ast.Node]bool)
				}
				r.externalStyleToComponentsMap[path][directive] = true
			}
		}
	}

	if resources.HostBindings != nil {
		r.directiveToHostBindingsMap[directive] = resources.HostBindings
	}
}

func (r *ResourceRegistry) GetTemplate(component *ast.Node) *Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.componentToTemplateMap[component]
}

func (r *ResourceRegistry) GetStyles(component *ast.Node) []*Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.componentToStylesMap[component]
}

func (r *ResourceRegistry) GetComponentsWithStyle(stylePath string) []*ast.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	comps, ok := r.externalStyleToComponentsMap[stylePath]
	if !ok {
		return nil
	}
	var res []*ast.Node
	for c := range comps {
		res = append(res, c)
	}
	return res
}

func (r *ResourceRegistry) GetHostBindings(directive *ast.Node) []*Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.directiveToHostBindingsMap[directive]
}
