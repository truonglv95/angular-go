package scope

import (
	"sync"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/internal/ast"
)

// LocalModuleScopeRegistry computes compilation scopes for components declared in NgModules.
type LocalModuleScopeRegistry struct {
	mu                sync.RWMutex
	metaReader        metadata.MetadataReader
	componentToModule map[*ast.Node]*ast.Node // Component Node -> NgModule Node
}

func NewLocalModuleScopeRegistry(metaReader metadata.MetadataReader) *LocalModuleScopeRegistry {
	return &LocalModuleScopeRegistry{
		metaReader:        metaReader,
		componentToModule: make(map[*ast.Node]*ast.Node),
	}
}

// Ensure interface implementation
var _ ScopeReader = (*LocalModuleScopeRegistry)(nil)

// RegisterComponentDeclaration registers that a component is declared in a specific NgModule.
func (r *LocalModuleScopeRegistry) RegisterComponentDeclaration(component *ast.Node, module *ast.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.componentToModule[component] = module
}

// GetCompilationScope resolves the compilation scope (visible directives, components, pipes) for a component.

func (r *LocalModuleScopeRegistry) GetCompilationScope(component *ast.Node) *CompilationScope {
	r.mu.RLock()
	module, ok := r.componentToModule[component]
	r.mu.RUnlock()

	// If the component is standalone, its scope is defined by its own imports array
	if compMeta := r.metaReader.GetDirectiveMetadata(component); compMeta != nil && compMeta.Standalone {
		scope := &CompilationScope{}
		visited := make(map[*ast.Node]bool)
		for _, ref := range compMeta.Imports {
			if ref.Node == nil {
				continue
			}
			if dirMeta := r.metaReader.GetDirectiveMetadata(ref.Node); dirMeta != nil {
				dirMetaCopy := *dirMeta
				if ref.OwningModule != "" {
					dirMetaCopy.Ref.OwningModule = ref.OwningModule
				}
				scope.Directives = append(scope.Directives, dirMetaCopy)
			}
			if pipeMeta := r.metaReader.GetPipeMetadata(ref.Node); pipeMeta != nil {
				pipeMetaCopy := *pipeMeta
				if ref.OwningModule != "" {
					pipeMetaCopy.Ref.OwningModule = ref.OwningModule
				}
				scope.Pipes = append(scope.Pipes, pipeMetaCopy)
			}
			if r.metaReader.GetNgModuleMetadata(ref.Node) != nil {
				r.collectModuleExports(ref.Node, ref.OwningModule, &scope.Directives, &scope.Pipes, visited)
			}
		}
		return scope
	}

	if !ok || module == nil {
		return &CompilationScope{}
	}

	moduleMeta := r.metaReader.GetNgModuleMetadata(module)
	if moduleMeta == nil {
		return &CompilationScope{}
	}

	scope := &CompilationScope{}
	visited := make(map[*ast.Node]bool)

	// Direct declarations of this module
	for _, ref := range moduleMeta.Declarations {
		if ref.Node == nil {
			continue
		}
		if dirMeta := r.metaReader.GetDirectiveMetadata(ref.Node); dirMeta != nil {
			dirMetaCopy := *dirMeta
			if ref.OwningModule != "" {
				dirMetaCopy.Ref.OwningModule = ref.OwningModule
			}
			scope.Directives = append(scope.Directives, dirMetaCopy)
		}
		if pipeMeta := r.metaReader.GetPipeMetadata(ref.Node); pipeMeta != nil {
			pipeMetaCopy := *pipeMeta
			if ref.OwningModule != "" {
				pipeMetaCopy.Ref.OwningModule = ref.OwningModule
			}
			scope.Pipes = append(scope.Pipes, pipeMetaCopy)
		}
	}

	// For imports, we add the exported declarations of each imported module/directive/pipe recursively
	for _, ref := range moduleMeta.Imports {
		if ref.Node == nil {
			continue
		}
		// If it's directly a directive
		if dirMeta := r.metaReader.GetDirectiveMetadata(ref.Node); dirMeta != nil {
			dirMetaCopy := *dirMeta
			if ref.OwningModule != "" {
				dirMetaCopy.Ref.OwningModule = ref.OwningModule
			}
			scope.Directives = append(scope.Directives, dirMetaCopy)
		}
		// If it's directly a pipe
		if pipeMeta := r.metaReader.GetPipeMetadata(ref.Node); pipeMeta != nil {
			pipeMetaCopy := *pipeMeta
			if ref.OwningModule != "" {
				pipeMetaCopy.Ref.OwningModule = ref.OwningModule
			}
			scope.Pipes = append(scope.Pipes, pipeMetaCopy)
		}
		// If it's a module, collect all its exported declarations recursively
		if r.metaReader.GetNgModuleMetadata(ref.Node) != nil {
			r.collectModuleExports(ref.Node, ref.OwningModule, &scope.Directives, &scope.Pipes, visited)
		}
	}

	return scope
}

func (r *LocalModuleScopeRegistry) collectModuleExports(moduleNode *ast.Node, owningModule string, directives *[]metadata.DirectiveMeta, pipes *[]metadata.PipeMeta, visited map[*ast.Node]bool) {
	if moduleNode == nil || visited[moduleNode] {
		return
	}
	visited[moduleNode] = true

	moduleMeta := r.metaReader.GetNgModuleMetadata(moduleNode)
	if moduleMeta == nil {
		return
	}

	for _, expRef := range moduleMeta.Exports {
		if expRef.Node == nil {
			continue
		}
		
		targetOwningModule := owningModule
		if expRef.OwningModule != "" {
			targetOwningModule = expRef.OwningModule
		}

		// 1. Is it a directive?
		if dirMeta := r.metaReader.GetDirectiveMetadata(expRef.Node); dirMeta != nil {
			dirMetaCopy := *dirMeta
			if targetOwningModule != "" {
				dirMetaCopy.Ref.OwningModule = targetOwningModule
			}
			*directives = append(*directives, dirMetaCopy)
		}
		// 2. Is it a pipe?
		if pipeMeta := r.metaReader.GetPipeMetadata(expRef.Node); pipeMeta != nil {
			pipeMetaCopy := *pipeMeta
			if targetOwningModule != "" {
				pipeMetaCopy.Ref.OwningModule = targetOwningModule
			}
			*pipes = append(*pipes, pipeMetaCopy)
		}
		// 3. Is it another module?
		if r.metaReader.GetNgModuleMetadata(expRef.Node) != nil {
			r.collectModuleExports(expRef.Node, targetOwningModule, directives, pipes, visited)
		}
	}
}

func (r *LocalModuleScopeRegistry) GetStandaloneScope(node *ast.Node) *StandaloneScope {
	compMeta := r.metaReader.GetDirectiveMetadata(node)
	if compMeta == nil || !compMeta.Standalone {
		return nil
	}

	scope := &StandaloneScope{
		Component: node,
	}
	
	visited := make(map[*ast.Node]bool)
	var dirs []metadata.DirectiveMeta
	var pipes []metadata.PipeMeta

	for _, ref := range compMeta.Imports {
		if ref.Node == nil {
			continue
		}
		
		targetOwningModule := ref.OwningModule
		
		if dirMeta := r.metaReader.GetDirectiveMetadata(ref.Node); dirMeta != nil {
			dirMetaCopy := *dirMeta
			if targetOwningModule != "" {
				dirMetaCopy.Ref.OwningModule = targetOwningModule
			}
			dirs = append(dirs, dirMetaCopy)
		}
		if pipeMeta := r.metaReader.GetPipeMetadata(ref.Node); pipeMeta != nil {
			pipeMetaCopy := *pipeMeta
			if targetOwningModule != "" {
				pipeMetaCopy.Ref.OwningModule = targetOwningModule
			}
			pipes = append(pipes, pipeMetaCopy)
		}
		if r.metaReader.GetNgModuleMetadata(ref.Node) != nil {
			r.collectModuleExports(ref.Node, targetOwningModule, &dirs, &pipes, visited)
		}
	}

	for i := range dirs {
		scope.Dependencies = append(scope.Dependencies, ScopeDependency{
			Kind: metadata.MetaKindDirective,
			Directive: &dirs[i],
		})
	}
	for i := range pipes {
		scope.Dependencies = append(scope.Dependencies, ScopeDependency{
			Kind: metadata.MetaKindPipe,
			Pipe: &pipes[i],
		})
	}

	return scope
}
