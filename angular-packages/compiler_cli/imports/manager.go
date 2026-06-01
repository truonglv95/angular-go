package imports

import (
	"sort"
	"strconv"

	"github.com/microsoft/typescript-go/internal/ast"
)

type ImportRewriter interface {
	RewriteSymbol(symbol string, specifier string) string
	RewriteSpecifier(specifier string, inContextOfFile string) string
	RewriteNamespaceImportIdentifier(specifier string, moduleName string) string
}

type ImportManagerConfig struct {
	GenerateUniqueIdentifier             func(file *ast.Node, baseName string) *ast.Node
	ShouldUseSingleQuotes                func(file *ast.Node) bool
	Rewriter                             ImportRewriter
	NamespaceImportPrefix                string
	DisableOriginalSourceFileReuse       bool
	ForceGenerateNamespacesForNewImports bool
}

var PresetImportManagerForceNamespaceImports = ImportManagerConfig{
	DisableOriginalSourceFileReuse:       true,
	ForceGenerateNamespacesForNewImports: true,
}

type ModuleName string

type ImportRequest struct {
	ExportModuleSpecifier string
	ExportSymbolName      *string
	RequestedFile         *ast.Node // actually SourceFile
	UnsafeAliasOverride   *string
	AsTypeReference       bool
}

type ImportGenerator interface {
	AddImport(request ImportRequest) *ast.Node
}

type fileImports struct {
	NamespaceImports  map[ModuleName]*ast.Node   // NamespaceImport
	NamedImports      map[ModuleName][]*ast.Node // ImportSpecifier
	SideEffectImports map[ModuleName]bool
}

type ReuseExistingSourceFileImportsTracker struct {
	GenerateUniqueIdentifier func(file *ast.Node, baseName string) *ast.Node
	ReusedAliasDeclarations  map[*ast.Node]bool        // Set of AliasImportDeclaration equivalent
	UpdatedImports           map[*ast.Node][]*ast.Node // ImportDeclaration -> UpdatedImportExpression equivalent
}

type ReuseGeneratedImportsTracker struct {
	DirectReuseCache          map[string]*ast.Node
	NamespaceImportReuseCache map[string]*ast.Node
}

type ImportManager struct {
	newImports                    map[*ast.Node]*fileImports
	removedImports                map[*ast.Node]map[ModuleName]map[string]bool
	nextUniqueIndex               int
	config                        ImportManagerConfig
	reuseSourceFileImportsTracker *ReuseExistingSourceFileImportsTracker
	reuseGeneratedImportsTracker  *ReuseGeneratedImportsTracker
	factory                       *ast.NodeFactory
}

func NewImportManager(config *ImportManagerConfig, factory *ast.NodeFactory) *ImportManager {
	c := ImportManagerConfig{
		ShouldUseSingleQuotes: func(file *ast.Node) bool { return false },
		NamespaceImportPrefix: "i",
	}
	if config != nil {
		if config.ShouldUseSingleQuotes != nil {
			c.ShouldUseSingleQuotes = config.ShouldUseSingleQuotes
		}
		c.Rewriter = config.Rewriter
		c.DisableOriginalSourceFileReuse = config.DisableOriginalSourceFileReuse
		c.ForceGenerateNamespacesForNewImports = config.ForceGenerateNamespacesForNewImports
		if config.NamespaceImportPrefix != "" {
			c.NamespaceImportPrefix = config.NamespaceImportPrefix
		}
		if config.GenerateUniqueIdentifier != nil {
			c.GenerateUniqueIdentifier = config.GenerateUniqueIdentifier
		}
	}

	return &ImportManager{
		newImports:      make(map[*ast.Node]*fileImports),
		removedImports:  make(map[*ast.Node]map[ModuleName]map[string]bool),
		nextUniqueIndex: 0,
		config:          c,
		reuseSourceFileImportsTracker: &ReuseExistingSourceFileImportsTracker{
			GenerateUniqueIdentifier: c.GenerateUniqueIdentifier,
			ReusedAliasDeclarations:  make(map[*ast.Node]bool),
			UpdatedImports:           make(map[*ast.Node][]*ast.Node),
		},
		reuseGeneratedImportsTracker: &ReuseGeneratedImportsTracker{
			DirectReuseCache:          make(map[string]*ast.Node),
			NamespaceImportReuseCache: make(map[string]*ast.Node),
		},
		factory: factory,
	}
}

func (m *ImportManager) AddSideEffectImport(requestedFile *ast.Node, moduleSpecifier string) {
	if m.config.Rewriter != nil {
		// Note: for filename we assume empty string since we don't have direct access here without casting
		moduleSpecifier = m.config.Rewriter.RewriteSpecifier(moduleSpecifier, "")
	}
	m.getNewImportsTrackerForFile(requestedFile).SideEffectImports[ModuleName(moduleSpecifier)] = true
}

func (m *ImportManager) AddImport(request ImportRequest) *ast.Node {
	if m.config.Rewriter != nil {
		if request.ExportSymbolName != nil {
			rewrittenSymbol := m.config.Rewriter.RewriteSymbol(*request.ExportSymbolName, request.ExportModuleSpecifier)
			request.ExportSymbolName = &rewrittenSymbol
		}
		request.ExportModuleSpecifier = m.config.Rewriter.RewriteSpecifier(request.ExportModuleSpecifier, "")
	}

	if request.ExportSymbolName != nil && !request.AsTypeReference {
		if fileMap, ok := m.removedImports[request.RequestedFile]; ok {
			if symbolMap, ok := fileMap[ModuleName(request.ExportModuleSpecifier)]; ok {
				delete(symbolMap, *request.ExportSymbolName)
			}
		}
	}

	// previousGeneratedImportRef := attemptToReuseGeneratedImports...
	var previousGeneratedImportRef *ast.Node = nil
	if previousGeneratedImportRef != nil {
		return createImportReference(request.AsTypeReference, previousGeneratedImportRef, m.factory)
	}

	resultImportRef := m.generateNewImport(request)
	return createImportReference(request.AsTypeReference, resultImportRef, m.factory)
}

func (m *ImportManager) RemoveImport(requestedFile *ast.Node, exportSymbolName string, moduleSpecifier string) {
	if m.removedImports == nil {
		m.removedImports = make(map[*ast.Node]map[ModuleName]map[string]bool)
	}
	moduleMap, ok := m.removedImports[requestedFile]
	if !ok {
		moduleMap = make(map[ModuleName]map[string]bool)
		m.removedImports[requestedFile] = moduleMap
	}

	removedSymbols, ok := moduleMap[ModuleName(moduleSpecifier)]
	if !ok {
		removedSymbols = make(map[string]bool)
		moduleMap[ModuleName(moduleSpecifier)] = removedSymbols
	}

	removedSymbols[exportSymbolName] = true
}

func (m *ImportManager) IsRemoved(requestedFile *ast.Node, exportSymbolName string, moduleSpecifier string) bool {
	if m.removedImports != nil {
		if moduleMap, ok := m.removedImports[requestedFile]; ok {
			if removedSymbols, ok := moduleMap[ModuleName(moduleSpecifier)]; ok {
				return removedSymbols[exportSymbolName]
			}
		}
	}
	return false
}

func (m *ImportManager) generateNewImport(request ImportRequest) *ast.Node {
	sourceFile := request.RequestedFile

	tracker := m.getNewImportsTrackerForFile(sourceFile)
	namedImports := tracker.NamedImports
	namespaceImports := tracker.NamespaceImports

	if request.ExportSymbolName == nil || m.config.ForceGenerateNamespacesForNewImports {
		var namespaceImportName string
		if existingImport, ok := namespaceImports[ModuleName(request.ExportModuleSpecifier)]; ok {
			namespaceImportName = existingImport.AsNamespaceImport().Name().AsIdentifier().Text
		} else {
			importStr := "i"
			if m.config.NamespaceImportPrefix != "" {
				importStr = m.config.NamespaceImportPrefix
			}
			importStr += strconv.Itoa(m.nextUniqueIndex)
			m.nextUniqueIndex++
			namespaceImportName = importStr

			namespaceImport := m.factory.NewNamespaceImport(m.factory.NewIdentifier(namespaceImportName))
			namespaceImports[ModuleName(request.ExportModuleSpecifier)] = namespaceImport
		}

		if request.ExportSymbolName != nil {
			// Returns i0.symbolName
			return m.factory.NewPropertyAccessExpression(
				(*ast.Expression)(m.factory.NewIdentifier(namespaceImportName)),
				nil,
				(*ast.MemberName)(m.factory.NewIdentifier(*request.ExportSymbolName)),
				0,
			)
		}
		return (*ast.Node)(m.factory.NewIdentifier(namespaceImportName))
	}

	if _, ok := namedImports[ModuleName(request.ExportModuleSpecifier)]; !ok {
		namedImports[ModuleName(request.ExportModuleSpecifier)] = []*ast.Node{}
	}

	var fileUniqueName *ast.Node
	if request.UnsafeAliasOverride == nil && m.config.GenerateUniqueIdentifier != nil {
		fileUniqueName = m.config.GenerateUniqueIdentifier(sourceFile, *request.ExportSymbolName)
	}

	var needsAlias bool
	var specifierName *ast.Node
	var exportSymbolNameIdent = m.factory.NewIdentifier(*request.ExportSymbolName)

	if request.UnsafeAliasOverride != nil {
		needsAlias = true
		specifierName = m.factory.NewIdentifier(*request.UnsafeAliasOverride)
	} else if fileUniqueName != nil {
		needsAlias = true
		specifierName = fileUniqueName
	} else {
		needsAlias = false
		specifierName = exportSymbolNameIdent
	}

	var propertyName *ast.Node
	if needsAlias {
		propertyName = exportSymbolNameIdent
	}

	specifier := m.factory.NewImportSpecifier(false, propertyName, specifierName)
	namedImports[ModuleName(request.ExportModuleSpecifier)] = append(namedImports[ModuleName(request.ExportModuleSpecifier)], specifier)

	return specifierName
}

func (m *ImportManager) getNewImportsTrackerForFile(file *ast.Node) *fileImports {
	if tracker, ok := m.newImports[file]; ok {
		return tracker
	}
	tracker := &fileImports{
		NamespaceImports:  make(map[ModuleName]*ast.Node),
		NamedImports:      make(map[ModuleName][]*ast.Node),
		SideEffectImports: make(map[ModuleName]bool),
	}
	m.newImports[file] = tracker
	return tracker
}

func createImportReference(asTypeReference bool, ref *ast.Node, factory *ast.NodeFactory) *ast.Node {
	return ref
}

func (m *ImportManager) GetAllImports(file *ast.Node) []*ast.Node {
	// We don't need to panic here, it might just mean that it's a generated file
	if file != nil && file.Kind == ast.KindSourceFile {
		// fmt.Printf("DEBUG: ImportManager.generateNewImport: Could not find node %s in file %s\n", request.ExportSymbolName, file.AsSourceFile().FileName())
	}
	tracker, ok := m.newImports[file]
	if !ok {
		return nil
	}
	var moduleNames []string
	for moduleName := range tracker.NamespaceImports {
		moduleNames = append(moduleNames, string(moduleName))
	}
	sort.Strings(moduleNames)

	var declarations []*ast.Node
	for _, moduleNameStr := range moduleNames {
		moduleName := ModuleName(moduleNameStr)
		namespaceImport := tracker.NamespaceImports[moduleName]
		importClause := m.factory.NewImportClause(ast.KindUnknown, nil, namespaceImport)
		importClause.Flags |= ast.NodeFlagsSynthesized
		namespaceImport.AsNode().Flags |= ast.NodeFlagsSynthesized
		moduleSpecifier := m.factory.NewStringLiteral(string(moduleName), 0)
		importDecl := m.factory.NewImportDeclaration(nil, importClause, moduleSpecifier, nil)
		importDecl.Flags |= ast.NodeFlagsSynthesized
		declarations = append(declarations, importDecl.AsNode())
	}
	// TODO: Handle NamedImports and SideEffectImports if needed
	return declarations
}
