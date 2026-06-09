package semantic_graph

import (
	"fmt"
	"sync"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/internal/ast"
)

type SemanticDependencyResult interface {
	GetAffectedFiles() []string
}

type semanticDependencyResultImpl struct {
	affectedFiles []string
}

func (r *semanticDependencyResultImpl) GetAffectedFiles() []string {
	return r.affectedFiles
}

type SemanticDepGraph struct {
	mu                sync.RWMutex
	SymbolByDecl      map[string]*SemanticSymbol
	symbols           map[string]*SemanticSymbol
	references        map[string][]string // source key -> target keys
	reverseReferences map[string][]string // target key -> source keys
}

func NewSemanticDepGraph() *SemanticDepGraph {
	return &SemanticDepGraph{
		SymbolByDecl:      make(map[string]*SemanticSymbol),
		symbols:           make(map[string]*SemanticSymbol),
		references:        make(map[string][]string),
		reverseReferences: make(map[string][]string),
	}
}

func (recv *SemanticDepGraph) ensureInitialized() {
	if recv.SymbolByDecl == nil {
		recv.SymbolByDecl = make(map[string]*SemanticSymbol)
	}
	if recv.symbols == nil {
		recv.symbols = make(map[string]*SemanticSymbol)
	}
	if recv.references == nil {
		recv.references = make(map[string][]string)
	}
	if recv.reverseReferences == nil {
		recv.reverseReferences = make(map[string][]string)
	}
}

func getDeclKey(decl any) string {
	if decl == nil {
		return ""
	}
	if s, ok := decl.(string); ok {
		return s
	}
	if sf, ok := decl.(interface{ FileName() string }); ok {
		return sf.FileName()
	}
	if node, ok := decl.(*ast.Node); ok {
		return nodeDeclKey(node)
	}
	if classDecl, ok := decl.(*ast.ClassDeclaration); ok {
		return nodeDeclKey(classDecl.AsNode())
	}
	return fmt.Sprintf("%v", decl)
}

func nodeDeclKey(node *ast.Node) string {
	if node == nil {
		return ""
	}
	sf := ast.GetSourceFileOfNode(node)
	if sf == nil {
		return fmt.Sprintf("%p", node)
	}
	name := ""
	if node.Kind == ast.KindClassDeclaration {
		classDecl := node.AsClassDeclaration()
		if classDecl.Name() != nil && ast.IsIdentifier(classDecl.Name()) {
			name = classDecl.Name().AsIdentifier().Text
		}
	}
	if name == "" {
		return sf.FileName()
	}
	return SymbolKey(sf.FileName(), name)
}

func SymbolKey(path string, identifier string) string {
	if path == "" {
		return identifier
	}
	if identifier == "" {
		return path
	}
	return path + "@" + identifier
}

func (recv *SemanticDepGraph) RegisterSymbol(symbol SemanticSymbol) any {
	recv.mu.Lock()
	defer recv.mu.Unlock()
	recv.ensureInitialized()

	key := SymbolKey(symbol.Path, symbol.Identifier)
	recv.symbols[key] = &symbol
	return nil
}

func (recv *SemanticDepGraph) GetEquivalentSymbol(symbol SemanticSymbol) SemanticSymbol {
	recv.mu.RLock()
	defer recv.mu.RUnlock()
	if recv.symbols == nil {
		return symbol
	}
	key := SymbolKey(symbol.Path, symbol.Identifier)
	if sym, ok := recv.symbols[key]; ok {
		return *sym
	}
	return symbol
}

func (recv *SemanticDepGraph) GetSymbolByDecl(decl any) SemanticSymbol {
	recv.mu.RLock()
	defer recv.mu.RUnlock()
	key := getDeclKey(decl)
	if recv.SymbolByDecl != nil {
		if sym, ok := recv.SymbolByDecl[key]; ok {
			return *sym
		}
	}
	return SemanticSymbol{Path: key}
}

func (recv *SemanticDepGraph) HasReferences() bool {
	recv.mu.RLock()
	defer recv.mu.RUnlock()
	return len(recv.references) > 0 || len(recv.reverseReferences) > 0
}

type SemanticDepGraphUpdater struct {
	mu               sync.Mutex
	graph            *SemanticDepGraph
	newSymbols       map[string]*SemanticSymbol
	symbolByDecl     map[string]*SemanticSymbol
	referencesByDecl map[string][]string
}

func NewSemanticDepGraphUpdater(graph *SemanticDepGraph) *SemanticDepGraphUpdater {
	if graph == nil {
		graph = NewSemanticDepGraph()
	}
	return &SemanticDepGraphUpdater{
		graph:            graph,
		newSymbols:       make(map[string]*SemanticSymbol),
		symbolByDecl:     make(map[string]*SemanticSymbol),
		referencesByDecl: make(map[string][]string),
	}
}

func (recv *SemanticDepGraphUpdater) RegisterSymbol(symbol SemanticSymbol) any {
	recv.mu.Lock()
	defer recv.mu.Unlock()
	key := SymbolKey(symbol.Path, symbol.Identifier)
	recv.newSymbols[key] = &symbol
	return nil
}

func (recv *SemanticDepGraphUpdater) RegisterSymbolForDecl(decl any, symbol SemanticSymbol) any {
	recv.mu.Lock()
	defer recv.mu.Unlock()
	key := SymbolKey(symbol.Path, symbol.Identifier)
	recv.newSymbols[key] = &symbol
	declKey := getDeclKey(decl)
	if declKey != "" {
		recv.symbolByDecl[declKey] = &symbol
	}
	return nil
}

func (recv *SemanticDepGraphUpdater) RegisterReference(fromDecl any, toSymbolKey string) {
	recv.mu.Lock()
	defer recv.mu.Unlock()

	srcKey := getDeclKey(fromDecl)
	if srcKey == "" || toSymbolKey == "" {
		return
	}
	recv.referencesByDecl[srcKey] = append(recv.referencesByDecl[srcKey], toSymbolKey)
}

func (recv *SemanticDepGraphUpdater) Finalize() SemanticDependencyResult {
	recv.mu.Lock()
	defer recv.mu.Unlock()

	recv.graph.mu.Lock()
	defer recv.graph.mu.Unlock()
	recv.graph.ensureInitialized()

	var affectedFiles []string
	affectedFilesMap := make(map[string]bool)

	// Set of symbol keys whose public API has changed
	changedPublicApis := make(map[string]bool)

	// Compare new symbols with old ones
	for key, newSym := range recv.newSymbols {
		oldSym, exists := recv.graph.symbols[key]
		if !exists || newSym.IsPublicApiAffected(*oldSym) {
			changedPublicApis[key] = true
			if !affectedFilesMap[newSym.Path] {
				affectedFilesMap[newSym.Path] = true
				affectedFiles = append(affectedFiles, newSym.Path)
			}
		}
		recv.graph.symbols[key] = newSym
	}

	// Transitive invalidation queue
	var queue []string
	for k := range changedPublicApis {
		queue = append(queue, k)
	}

	visited := make(map[string]bool)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if dependents, ok := recv.graph.reverseReferences[current]; ok {
			for _, depKey := range dependents {
				depSym, exists := recv.graph.symbols[depKey]
				if exists {
					if !affectedFilesMap[depSym.Path] {
						affectedFilesMap[depSym.Path] = true
						affectedFiles = append(affectedFiles, depSym.Path)
					}
					if !visited[depKey] {
						queue = append(queue, depKey)
					}
				}
			}
		}
	}

	// Clear old references for re-analyzed declarations
	for declKey := range recv.symbolByDecl {
		sym := recv.symbolByDecl[declKey]
		symKey := SymbolKey(sym.Path, sym.Identifier)
		delete(recv.graph.references, symKey)
	}

	// Update SymbolByDecl
	for declKey, sym := range recv.symbolByDecl {
		recv.graph.SymbolByDecl[declKey] = sym
	}

	// Record new references
	for declKey, targets := range recv.referencesByDecl {
		sym := recv.symbolByDecl[declKey]
		if sym == nil {
			continue
		}
		symKey := SymbolKey(sym.Path, sym.Identifier)
		recv.graph.references[symKey] = targets
	}

	// Recompute reverseReferences
	recv.graph.reverseReferences = make(map[string][]string)
	for srcKey, targets := range recv.graph.references {
		for _, tgtKey := range targets {
			recv.graph.reverseReferences[tgtKey] = append(recv.graph.reverseReferences[tgtKey], srcKey)
		}
	}

	return &semanticDependencyResultImpl{affectedFiles: affectedFiles}
}

type semanticReferenceImpl struct {
	source *SemanticSymbol
	target *SemanticSymbol
}

func (r *semanticReferenceImpl) GetSource() *SemanticSymbol {
	return r.source
}

func (r *semanticReferenceImpl) GetTarget() *SemanticSymbol {
	return r.target
}

func (recv *SemanticDepGraphUpdater) GetSemanticReference(decl any, expr output.Expression) SemanticReference {
	src := recv.GetSymbol(decl)
	tgt := SemanticSymbol{Path: getDeclKey(decl), Identifier: fmt.Sprintf("%v", expr)}
	return &semanticReferenceImpl{source: &src, target: &tgt}
}

func (recv *SemanticDepGraphUpdater) GetSymbol(decl any) SemanticSymbol {
	recv.mu.Lock()
	defer recv.mu.Unlock()

	key := getDeclKey(decl)
	if sym, ok := recv.symbolByDecl[key]; ok {
		return *sym
	}
	sym := SemanticSymbol{Path: key, Identifier: "Symbol"}
	recv.symbolByDecl[key] = &sym
	return sym
}

func (recv *SemanticDepGraphUpdater) GetGraph() *SemanticDepGraph {
	return recv.graph
}
