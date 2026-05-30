package imports

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/internal/ast"
)

// OwningModule represents a module specifier that owns a reference.
type OwningModule struct {
	Specifier         string
	ResolutionContext string
}

// Reference is a pointer to a class/declaration node plus the context in which it was discovered.
type Reference struct {
	Node                  *ast.Node
	BestGuessOwningModule *OwningModule
	Identifiers           []*ast.Node
	Synthetic             bool
	IsAmbient             bool
	Key                   string
	_alias                output.Expression
}

func NewReference(node *ast.Node, bestGuessOwningModule *OwningModule) *Reference {
	ref := &Reference{
		Node:                  node,
		BestGuessOwningModule: bestGuessOwningModule,
	}
	if node != nil {
		if id := identifierOfNode(node); id != nil {
			ref.Identifiers = append(ref.Identifiers, id)
		}
		// Calculate key
		sf := getSourceFile(node)
		if sf != nil {
			ref.Key = fmt.Sprintf("%s#%d", sf.FileName(), node.Pos())
		} else if bestGuessOwningModule != nil {
			idText := ""
			if id := identifierOfNode(node); id != nil {
				idText = id.AsIdentifier().Text
			}
			ref.Key = fmt.Sprintf("%s#%s#%d", bestGuessOwningModule.Specifier, idText, node.Pos())
		}
	}
	return ref
}

func NewAmbientReference(node *ast.Node) *Reference {
	ref := NewReference(node, nil)
	ref.IsAmbient = true
	return ref
}

func (recv *Reference) OwnedByModuleGuess() string {
	if recv.BestGuessOwningModule != nil {
		return recv.BestGuessOwningModule.Specifier
	}
	return ""
}

func (recv *Reference) HasOwningModuleGuess() bool {
	return recv.BestGuessOwningModule != nil
}

func (recv *Reference) DebugName() string {
	if id := identifierOfNode(recv.Node); id != nil {
		return id.AsIdentifier().Text
	}
	return ""
}

func (recv *Reference) Alias() output.Expression {
	return recv._alias
}

func (recv *Reference) AddIdentifier(identifier *ast.Node) {
	recv.Identifiers = append(recv.Identifiers, identifier)
}

func (recv *Reference) GetIdentityIn(context *ast.SourceFile) *ast.Node {
	for _, id := range recv.Identifiers {
		if getSourceFile(id) == context {
			return id
		}
	}
	return nil
}

func (recv *Reference) GetIdentityInExpression(expr *ast.Node) *ast.Node {
	sf := getSourceFile(expr)
	for _, id := range recv.Identifiers {
		if getSourceFile(id) != sf {
			continue
		}
		if id.Pos() >= expr.Pos() && id.End() <= expr.End() {
			return id
		}
	}
	return nil
}

func (recv *Reference) GetOriginForDiagnostics(container *ast.Node, fallback *ast.Node) *ast.Node {
	id := recv.GetIdentityInExpression(container)
	if id != nil {
		return id
	}
	return fallback
}

func (recv *Reference) CloneWithAlias(alias output.Expression) *Reference {
	ref := &Reference{
		Node:                  recv.Node,
		BestGuessOwningModule: recv.BestGuessOwningModule,
		Identifiers:           append([]*ast.Node(nil), recv.Identifiers...),
		IsAmbient:             recv.IsAmbient,
		Synthetic:             recv.Synthetic,
		Key:                   recv.Key,
		_alias:                alias,
	}
	return ref
}

func (recv *Reference) CloneWithNoIdentifiers() *Reference {
	ref := &Reference{
		Node:                  recv.Node,
		BestGuessOwningModule: recv.BestGuessOwningModule,
		IsAmbient:             recv.IsAmbient,
		Synthetic:             recv.Synthetic,
		Key:                   recv.Key,
		_alias:                recv._alias,
	}
	return ref
}

// Helpers for ast nodes
func getSourceFile(node *ast.Node) *ast.SourceFile {
	return ast.GetSourceFileOfNode(node)
}

func identifierOfNode(decl *ast.Node) *ast.Node {
	if decl == nil {
		return nil
	}
	switch decl.Kind {
	case ast.KindClassDeclaration:
		if classDecl := decl.AsClassDeclaration(); classDecl != nil && classDecl.Name() != nil {
			return classDecl.Name()
		}
	case ast.KindFunctionDeclaration:
		if funcDecl := decl.AsFunctionDeclaration(); funcDecl != nil && funcDecl.Name() != nil {
			return funcDecl.Name()
		}
	case ast.KindVariableDeclaration:
		if varDecl := decl.AsVariableDeclaration(); varDecl != nil && varDecl.Name() != nil {
			if name := varDecl.Name(); name != nil && name.Kind == ast.KindIdentifier {
				return name
			}
		}
	case ast.KindInterfaceDeclaration:
		if ifaceDecl := decl.AsInterfaceDeclaration(); ifaceDecl != nil && ifaceDecl.Name() != nil {
			return ifaceDecl.Name()
		}
	case ast.KindEnumDeclaration:
		if enumDecl := decl.AsEnumDeclaration(); enumDecl != nil && enumDecl.Name() != nil {
			return enumDecl.Name()
		}
	case ast.KindTypeAliasDeclaration:
		if aliasDecl := decl.AsTypeAliasDeclaration(); aliasDecl != nil && aliasDecl.Name() != nil {
			return aliasDecl.Name()
		}
	case ast.KindIdentifier:
		return decl
	}
	return nil
}
