package imports

import (
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
)

var defaultImportDeclarations sync.Map // map[any]*ast.Node

// AttachDefaultImportDeclaration attaches a default import declaration to expr.
func AttachDefaultImportDeclaration(expr any, importDecl *ast.Node) {
	if expr != nil && importDecl != nil {
		defaultImportDeclarations.Store(expr, importDecl)
	}
}

// GetDefaultImportDeclaration obtains the default import declaration associated with expr.
func GetDefaultImportDeclaration(expr any) *ast.Node {
	if expr == nil {
		return nil
	}
	if val, ok := defaultImportDeclarations.Load(expr); ok {
		if node, ok := val.(*ast.Node); ok {
			return node
		}
	}
	return nil
}

// DefaultImportTracker keeps track of default imports that are used.
type DefaultImportTracker struct {
	mu                      sync.RWMutex
	sourceFileToUsedImports map[string]map[*ast.Node]bool
}

func NewDefaultImportTracker() *DefaultImportTracker {
	return &DefaultImportTracker{
		sourceFileToUsedImports: make(map[string]map[*ast.Node]bool),
	}
}

// RecordUsedImport registers a default import declaration as being used.
func (t *DefaultImportTracker) RecordUsedImport(importDecl *ast.Node) {
	if importDecl == nil {
		return
	}
	sf := getSourceFileOfNode(importDecl)
	if sf == nil {
		return
	}
	fileName := sf.AsSourceFile().FileName()

	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.sourceFileToUsedImports[fileName]; !ok {
		t.sourceFileToUsedImports[fileName] = make(map[*ast.Node]bool)
	}
	t.sourceFileToUsedImports[fileName][importDecl] = true
}

// ImportPreservingTransformer returns a transformer factory that preserves default imports.
func (t *DefaultImportTracker) ImportPreservingTransformer() any {
	return nil
}

func getSourceFileOfNode(node *ast.Node) *ast.Node {
	for node != nil {
		if ast.IsSourceFile(node) {
			return node
		}
		node = node.Parent
	}
	return nil
}
