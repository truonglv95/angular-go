package metadata

import (
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
)

// LocalMetadataRegistry stores analyzed metadata for the current compilation.
type LocalMetadataRegistry struct {
	mu         sync.RWMutex
	directives map[*ast.Node]*DirectiveMeta
	modules    map[*ast.Node]*NgModuleMeta
	pipes      map[*ast.Node]*PipeMeta
}

func NewLocalMetadataRegistry() *LocalMetadataRegistry {
	return &LocalMetadataRegistry{
		directives: make(map[*ast.Node]*DirectiveMeta),
		modules:    make(map[*ast.Node]*NgModuleMeta),
		pipes:      make(map[*ast.Node]*PipeMeta),
	}
}

// Ensure interface implementations
var _ MetadataReader = (*LocalMetadataRegistry)(nil)
var _ MetadataRegistry = (*LocalMetadataRegistry)(nil)

func (r *LocalMetadataRegistry) RegisterDirective(node *ast.Node, meta *DirectiveMeta) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.directives[node] = meta
}

func (r *LocalMetadataRegistry) RegisterNgModule(node *ast.Node, meta *NgModuleMeta) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[node] = meta
}

func (r *LocalMetadataRegistry) RegisterPipe(node *ast.Node, meta *PipeMeta) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pipes[node] = meta
}

func (r *LocalMetadataRegistry) GetDirectiveMetadata(node *ast.Node) *DirectiveMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.directives[node]
}

func (r *LocalMetadataRegistry) GetNgModuleMetadata(node *ast.Node) *NgModuleMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modules[node]
}

func (r *LocalMetadataRegistry) GetPipeMetadata(node *ast.Node) *PipeMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.pipes[node]
}
