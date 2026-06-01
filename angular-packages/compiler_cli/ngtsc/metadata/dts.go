package metadata

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/core"
)

// DtsMetadataReader extracts Angular Ivy metadata from compiled declaration files (.d.ts).
type DtsMetadataReader struct {
	checker *checker.Checker
}

func NewDtsMetadataReader(checker *checker.Checker) *DtsMetadataReader {
	return &DtsMetadataReader{checker: checker}
}

// Ensure it implements MetadataReader
var _ MetadataReader = (*DtsMetadataReader)(nil)

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

			// Parse Selector
			selector := ""
			selectorArg := typeArgs[1]
			if selectorArg.Kind == ast.KindLiteralType {
				lit := selectorArg.AsLiteralTypeNode().Literal
				if lit.Kind == ast.KindStringLiteral {
					selector = lit.AsStringLiteral().Text
				}
			}

			// Parse Standalone
			standalone := false
			if len(typeArgs) > 7 {
				standaloneArg := typeArgs[7]
				if standaloneArg.Kind == ast.KindLiteralType {
					lit := standaloneArg.AsLiteralTypeNode().Literal
					if lit.Kind == ast.KindTrueKeyword {
						standalone = true
					}
				}
			}

			className := ""
			if classDecl.Name() != nil {
				className = classDecl.Name().AsIdentifier().Text
			}

			return &DirectiveMeta{
				Name:        className,
				Kind:        core.IfElse(isComponent, MetaKindComponent, MetaKindDirective),
				Ref:         Reference{Node: classNode, Name: className},
				Selector:    selector,
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
				Ref:          Reference{Node: classNode},
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

			// Parse Name
			name := ""
			nameArg := typeArgs[1]
			if nameArg.Kind == ast.KindLiteralType {
				lit := nameArg.AsLiteralTypeNode().Literal
				if lit.Kind == ast.KindStringLiteral {
					name = lit.AsStringLiteral().Text
				}
			}

			// Parse Pure & Standalone
			pure := true
			standalone := false
			if len(typeArgs) > 2 {
				pureArg := typeArgs[2]
				if pureArg.Kind == ast.KindLiteralType {
					lit := pureArg.AsLiteralTypeNode().Literal
					if lit.Kind == ast.KindFalseKeyword {
						pure = false
					}
				}
			}
			if len(typeArgs) > 3 {
				standaloneArg := typeArgs[3]
				if standaloneArg.Kind == ast.KindLiteralType {
					lit := standaloneArg.AsLiteralTypeNode().Literal
					if lit.Kind == ast.KindTrueKeyword {
						standalone = true
					}
				}
			}

			className := ""
			if classDecl.Name() != nil {
				className = classDecl.Name().AsIdentifier().Text
			}

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

func (r *DtsMetadataReader) resolveOwningModule(exprName *ast.Node) string {
	if exprName.Kind != ast.KindQualifiedName {
		return ""
	}
	left := exprName.AsQualifiedName().Left
	if left.Kind != ast.KindIdentifier {
		return ""
	}
	namespaceName := left.AsIdentifier().Text
	sourceFile := ast.GetSourceFileOfNode(exprName)
	if sourceFile == nil {
		return ""
	}

	// Scan top level statements for import declaration matching the namespace
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind == ast.KindImportDeclaration {
			importDecl := stmt.AsImportDeclaration()
			if importDecl.ImportClause != nil && importDecl.ImportClause.AsImportClause().NamedBindings != nil {
				if importDecl.ImportClause.AsImportClause().NamedBindings.Kind == ast.KindNamespaceImport {
					nsImport := importDecl.ImportClause.AsImportClause().NamedBindings.AsNamespaceImport()
					if nsImport.Name().AsIdentifier().Text == namespaceName {
						if importDecl.ModuleSpecifier.Kind == ast.KindStringLiteral {
							return importDecl.ModuleSpecifier.AsStringLiteral().Text
						}
					}
				}
			}
		}
	}
	return ""
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
				owningModule := r.resolveOwningModule(exprName)

				var targetNode *ast.Node = exprName
				if r.checker != nil {
					sym := r.checker.GetSymbolAtLocation(exprName)
					if sym != nil {
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

				refs = append(refs, Reference{
					Node:         targetNode,
					Name:         r.extractName(exprName),
					OwningModule: owningModule,
				})
			} else {
				refs = append(refs, Reference{Node: elem})
			}
		}
	} else if node.Kind == ast.KindTypeReference {
		refs = append(refs, Reference{Node: node})
	} else if node.Kind == ast.KindTypeQuery {
		exprName := node.AsTypeQueryNode().ExprName
		owningModule := r.resolveOwningModule(exprName)
		refs = append(refs, Reference{
			Node:         exprName,
			Name:         r.extractName(exprName),
			OwningModule: owningModule,
		})
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
