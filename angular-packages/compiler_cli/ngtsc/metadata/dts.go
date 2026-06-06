package metadata

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/core"
)

// DtsMetadataReader extracts Angular Ivy metadata from compiled declaration files (.d.ts).
type DtsMetadataReader struct {
	checker *checker.Checker
}

type dtsImportAlias struct {
	name         string
	owningModule string
}

func NewDtsMetadataReader(checker *checker.Checker) *DtsMetadataReader {
	return &DtsMetadataReader{checker: checker}
}

// Ensure it implements MetadataReader
var _ MetadataReader = (*DtsMetadataReader)(nil)
var _ SemanticMetadataReader = (*DtsMetadataReader)(nil)

func classNameOf(classDecl *ast.ClassDeclaration) string {
	if classDecl == nil || classDecl.Name() == nil {
		return ""
	}
	return classDecl.Name().AsIdentifier().Text
}

func sourcePathOf(node *ast.Node) string {
	sf := ast.GetSourceFileOfNode(node)
	if sf == nil {
		return ""
	}
	return sf.FileName()
}

func literalTypeString(node *ast.Node) string {
	if node == nil || node.Kind != ast.KindLiteralType {
		return ""
	}
	lit := node.AsLiteralTypeNode().Literal
	if lit.Kind == ast.KindStringLiteral {
		return lit.AsStringLiteral().Text
	}
	return ""
}

func literalTypeBool(node *ast.Node, defaultValue bool) bool {
	if node == nil || node.Kind != ast.KindLiteralType {
		return defaultValue
	}
	switch node.AsLiteralTypeNode().Literal.Kind {
	case ast.KindTrueKeyword:
		return true
	case ast.KindFalseKeyword:
		return false
	default:
		return defaultValue
	}
}

func parseStringTupleType(node *ast.Node) []string {
	if node == nil || node.Kind != ast.KindTupleType {
		return nil
	}
	var res []string
	for _, elem := range node.AsTupleTypeNode().Elements.Nodes {
		if value := literalTypeString(elem); value != "" {
			res = append(res, value)
		}
	}
	return res
}

func parseStringMapType(node *ast.Node) map[string]string {
	res := make(map[string]string)
	if node == nil || node.Kind != ast.KindTypeLiteral {
		return res
	}
	for _, member := range node.AsTypeLiteralNode().Members.Nodes {
		if member.Kind != ast.KindPropertySignature {
			continue
		}
		prop := member.AsPropertySignatureDeclaration()
		name := propertyNameText(prop.Name())
		if name == "" {
			continue
		}
		alias := name
		if prop.Type != nil {
			if value := literalTypeString(prop.Type); value != "" {
				alias = value
			} else if prop.Type.Kind == ast.KindTypeLiteral {
				if nested := parseStringMapType(prop.Type); nested["alias"] != "" {
					alias = nested["alias"]
				}
			}
		}
		res[name] = alias
	}
	return res
}

func propertyNameText(node *ast.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind {
	case ast.KindIdentifier:
		return node.AsIdentifier().Text
	case ast.KindStringLiteral:
		return node.AsStringLiteral().Text
	default:
		return ""
	}
}

func (r *DtsMetadataReader) GetDirectiveMetadata(classNode *ast.Node) *DirectiveMeta {
	if classNode == nil || classNode.Kind != ast.KindClassDeclaration {
		return nil
	}
	classDecl := classNode.AsClassDeclaration()
	for _, member := range classDecl.Members.Nodes {
		if member.Kind != ast.KindPropertyDeclaration {
			continue
		}
		prop := member.AsPropertyDeclaration()
		propName := prop.Name().AsIdentifier().Text
		if propName == "ɵcmp" || propName == "ɵdir" {
			isComponent := propName == "ɵcmp"
			typeNode := prop.Type
			if typeNode == nil || typeNode.Kind != ast.KindTypeReference {
				continue
			}
			typeRef := typeNode.AsTypeReferenceNode()
			if typeRef.TypeArguments == nil {
				continue
			}
			typeArgs := typeRef.TypeArguments.Nodes
			if len(typeArgs) < 2 {
				continue
			}

			selector := literalTypeString(typeArgs[1])
			var exportAs []string
			if len(typeArgs) > 2 {
				exportAs = parseStringTupleType(typeArgs[2])
			}
			inputs := make(map[string]string)
			if len(typeArgs) > 3 {
				inputs = parseStringMapType(typeArgs[3])
			}
			outputs := make(map[string]string)
			if len(typeArgs) > 4 {
				outputs = parseStringMapType(typeArgs[4])
			}

			// Parse Standalone
			standalone := false
			if len(typeArgs) > 7 {
				standalone = literalTypeBool(typeArgs[7], false)
			}

			className := classNameOf(classDecl)

			return &DirectiveMeta{
				Name:        className,
				Kind:        core.IfElse(isComponent, MetaKindComponent, MetaKindDirective),
				Ref:         Reference{Node: classNode, Name: className},
				Selector:    selector,
				Inputs:      inputs,
				Outputs:     outputs,
				ExportAs:    exportAs,
				Standalone:  standalone,
				IsComponent: isComponent,
			}
		}
	}
	return nil
}

func (r *DtsMetadataReader) GetNgModuleMetadata(classNode *ast.Node) *NgModuleMeta {
	if classNode == nil || classNode.Kind != ast.KindClassDeclaration {
		return nil
	}
	classDecl := classNode.AsClassDeclaration()
	for _, member := range classDecl.Members.Nodes {
		if member.Kind != ast.KindPropertyDeclaration {
			continue
		}
		prop := member.AsPropertyDeclaration()
		propName := prop.Name().AsIdentifier().Text
		if propName == "ɵmod" {
			typeNode := prop.Type
			if typeNode == nil || typeNode.Kind != ast.KindTypeReference {
				continue
			}
			typeRef := typeNode.AsTypeReferenceNode()
			if typeRef.TypeArguments == nil {
				continue
			}
			typeArgs := typeRef.TypeArguments.Nodes
			if len(typeArgs) < 4 {
				continue
			}

			declarations := r.parseReferencesList(typeArgs[1])
			imports := r.parseReferencesList(typeArgs[2])
			exports := r.parseReferencesList(typeArgs[3])

			return &NgModuleMeta{
				Name:         classNameOf(classDecl),
				Ref:          Reference{Node: classNode, Name: classNameOf(classDecl)},
				Declarations: declarations,
				Imports:      imports,
				Exports:      exports,
			}
		}
	}
	return nil
}

func (r *DtsMetadataReader) GetPipeMetadata(classNode *ast.Node) *PipeMeta {
	if classNode == nil || classNode.Kind != ast.KindClassDeclaration {
		return nil
	}
	classDecl := classNode.AsClassDeclaration()
	for _, member := range classDecl.Members.Nodes {
		if member.Kind != ast.KindPropertyDeclaration {
			continue
		}
		prop := member.AsPropertyDeclaration()
		propName := prop.Name().AsIdentifier().Text
		if propName == "ɵpipe" {
			typeNode := prop.Type
			if typeNode == nil || typeNode.Kind != ast.KindTypeReference {
				continue
			}
			typeRef := typeNode.AsTypeReferenceNode()
			if typeRef.TypeArguments == nil {
				continue
			}
			typeArgs := typeRef.TypeArguments.Nodes
			if len(typeArgs) < 2 {
				continue
			}

			name := literalTypeString(typeArgs[1])

			// Parse Pure & Standalone
			pure := true
			standalone := false
			if len(typeArgs) > 2 {
				pure = literalTypeBool(typeArgs[2], true)
			}
			if len(typeArgs) > 3 {
				standalone = literalTypeBool(typeArgs[3], false)
			}

			className := classNameOf(classDecl)

			return &PipeMeta{
				Ref:        Reference{Node: classNode, Name: className},
				Name:       name,
				Pure:       pure,
				Standalone: standalone,
			}
		}
	}
	return nil
}

func (r *DtsMetadataReader) GetSemanticSymbol(node *ast.Node) *semantic_graph.SemanticSymbol {
	if node == nil || node.Kind != ast.KindClassDeclaration {
		return nil
	}
	path := sourcePathOf(node)
	if path == "" {
		return nil
	}
	if dirMeta := r.GetDirectiveMetadata(node); dirMeta != nil {
		return &semantic_graph.SemanticSymbol{
			Path:       path,
			Identifier: dirMeta.Ref.Name,
			Kind:       core.IfElse(dirMeta.IsComponent, "component", "directive"),
			Selector:   dirMeta.Selector,
			Inputs:     dirMeta.Inputs,
			Outputs:    dirMeta.Outputs,
			ExportAs:   dirMeta.ExportAs,
			Standalone: dirMeta.Standalone,
		}
	}
	if pipeMeta := r.GetPipeMetadata(node); pipeMeta != nil {
		return &semantic_graph.SemanticSymbol{
			Path:       path,
			Identifier: pipeMeta.Ref.Name,
			Kind:       "pipe",
			PipeName:   pipeMeta.Name,
			Pure:       pipeMeta.Pure,
			Standalone: pipeMeta.Standalone,
		}
	}
	if moduleMeta := r.GetNgModuleMetadata(node); moduleMeta != nil {
		names := func(refs []Reference) []string {
			var res []string
			for _, ref := range refs {
				if ref.Name != "" {
					res = append(res, ref.Name)
				}
			}
			return res
		}
		return &semantic_graph.SemanticSymbol{
			Path:         path,
			Identifier:   moduleMeta.Ref.Name,
			Kind:         "ngmodule",
			Declarations: names(moduleMeta.Declarations),
			Imports:      names(moduleMeta.Imports),
			Exports:      names(moduleMeta.Exports),
		}
	}
	return nil
}

func (r *DtsMetadataReader) resolveOwningModule(exprName *ast.Node) string {
	ref := r.resolveReference(exprName)
	return ref.OwningModule
}

func (r *DtsMetadataReader) resolveReference(exprName *ast.Node) Reference {
	if exprName == nil {
		return Reference{}
	}
	name := r.extractName(exprName)
	owningModule := ""
	if alias, ok := r.resolveImportAlias(exprName); ok {
		if alias.name != "" {
			name = alias.name
		}
		owningModule = alias.owningModule
	}

	targetNode := exprName
	if r.checker != nil {
		if sym := r.checker.GetSymbolAtLocation(exprName); sym != nil {
			for sym.Flags&ast.SymbolFlagsAlias != 0 {
				sym = r.checker.GetAliasedSymbol(sym)
			}
			if sym.ValueDeclaration != nil {
				targetNode = sym.ValueDeclaration
			} else if len(sym.Declarations) > 0 {
				targetNode = sym.Declarations[0]
			}
		}
	}

	if targetNode != nil && targetNode.Kind == ast.KindClassDeclaration {
		if className := classNameOf(targetNode.AsClassDeclaration()); className != "" {
			name = className
		}
	}

	return Reference{
		Node:         targetNode,
		Name:         name,
		OwningModule: owningModule,
	}
}

func (r *DtsMetadataReader) resolveImportAlias(exprName *ast.Node) (dtsImportAlias, bool) {
	if exprName == nil {
		return dtsImportAlias{}, false
	}
	sourceFile := ast.GetSourceFileOfNode(exprName)
	if sourceFile == nil {
		return dtsImportAlias{}, false
	}

	if exprName.Kind == ast.KindQualifiedName {
		left := exprName.AsQualifiedName().Left
		if left.Kind != ast.KindIdentifier {
			return dtsImportAlias{}, false
		}
		namespaceName := left.AsIdentifier().Text
		rightName := exprName.AsQualifiedName().Right.AsIdentifier().Text
		for _, stmt := range sourceFile.Statements.Nodes {
			if stmt.Kind != ast.KindImportDeclaration {
				continue
			}
			importDecl := stmt.AsImportDeclaration()
			if importDecl.ModuleSpecifier == nil || !ast.IsStringLiteral(importDecl.ModuleSpecifier) {
				continue
			}
			if importDecl.ImportClause == nil || importDecl.ImportClause.AsImportClause().NamedBindings == nil {
				continue
			}
			namedBindings := importDecl.ImportClause.AsImportClause().NamedBindings
			if namedBindings.Kind != ast.KindNamespaceImport {
				continue
			}
			nsImport := namedBindings.AsNamespaceImport()
			if nsImport.Name().AsIdentifier().Text == namespaceName {
				return dtsImportAlias{
					name:         rightName,
					owningModule: importDecl.ModuleSpecifier.AsStringLiteral().Text,
				}, true
			}
		}
		return dtsImportAlias{}, false
	}

	if exprName.Kind != ast.KindIdentifier {
		return dtsImportAlias{}, false
	}
	localName := exprName.AsIdentifier().Text
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind != ast.KindImportDeclaration {
			continue
		}
		importDecl := stmt.AsImportDeclaration()
		if importDecl.ModuleSpecifier == nil || !ast.IsStringLiteral(importDecl.ModuleSpecifier) {
			continue
		}
		if importDecl.ImportClause == nil {
			continue
		}
		importClause := importDecl.ImportClause.AsImportClause()
		if importClause.Name() != nil && importClause.Name().AsIdentifier().Text == localName {
			return dtsImportAlias{
				name:         "default",
				owningModule: importDecl.ModuleSpecifier.AsStringLiteral().Text,
			}, true
		}
		if importClause.NamedBindings == nil || importClause.NamedBindings.Kind != ast.KindNamedImports {
			continue
		}
		for _, element := range importClause.NamedBindings.AsNamedImports().Elements.Nodes {
			if element.Kind != ast.KindImportSpecifier {
				continue
			}
			spec := element.AsImportSpecifier()
			if spec.Name().AsIdentifier().Text != localName {
				continue
			}
			importedName := localName
			if spec.PropertyName != nil {
				importedName = spec.PropertyName.Text()
			}
			return dtsImportAlias{
				name:         importedName,
				owningModule: importDecl.ModuleSpecifier.AsStringLiteral().Text,
			}, true
		}
	}
	return dtsImportAlias{}, false
}

func (r *DtsMetadataReader) extractName(exprName *ast.Node) string {
	if exprName.Kind == ast.KindIdentifier {
		return exprName.AsIdentifier().Text
	} else if exprName.Kind == ast.KindQualifiedName {
		return exprName.AsQualifiedName().Right.AsIdentifier().Text
	}
	return ""
}

func (r *DtsMetadataReader) parseReferencesList(node *ast.Node) []Reference {
	if node == nil {
		return nil
	}
	var refs []Reference
	if node.Kind == ast.KindTupleType {
		tuple := node.AsTupleTypeNode()
		for _, elem := range tuple.Elements.Nodes {
			if elem.Kind == ast.KindTypeQuery {
				exprName := elem.AsTypeQueryNode().ExprName
				refs = append(refs, r.resolveReference(exprName))
			} else {
				refs = append(refs, Reference{Node: elem})
			}
		}
	} else if node.Kind == ast.KindTypeReference {
		refs = append(refs, Reference{Node: node})
	} else if node.Kind == ast.KindTypeQuery {
		exprName := node.AsTypeQueryNode().ExprName
		refs = append(refs, r.resolveReference(exprName))
	}
	return refs
}

// CompoundMetadataReader queries multiple readers in sequence.
type CompoundMetadataReader struct {
	readers []MetadataReader
}

func NewCompoundMetadataReader(readers []MetadataReader) *CompoundMetadataReader {
	return &CompoundMetadataReader{readers: readers}
}

var _ MetadataReader = (*CompoundMetadataReader)(nil)
var _ SemanticMetadataReader = (*CompoundMetadataReader)(nil)

func (c *CompoundMetadataReader) GetDirectiveMetadata(node *ast.Node) *DirectiveMeta {
	for _, reader := range c.readers {
		if meta := reader.GetDirectiveMetadata(node); meta != nil {
			return meta
		}
	}
	return nil
}

func (c *CompoundMetadataReader) GetNgModuleMetadata(node *ast.Node) *NgModuleMeta {
	for _, reader := range c.readers {
		if meta := reader.GetNgModuleMetadata(node); meta != nil {
			return meta
		}
	}
	return nil
}

func (c *CompoundMetadataReader) GetPipeMetadata(node *ast.Node) *PipeMeta {
	for _, reader := range c.readers {
		if meta := reader.GetPipeMetadata(node); meta != nil {
			return meta
		}
	}
	return nil
}

func (c *CompoundMetadataReader) GetSemanticSymbol(node *ast.Node) *semantic_graph.SemanticSymbol {
	for _, reader := range c.readers {
		if semanticReader, ok := reader.(SemanticMetadataReader); ok {
			if symbol := semanticReader.GetSemanticSymbol(node); symbol != nil {
				return symbol
			}
		}
	}
	return nil
}
