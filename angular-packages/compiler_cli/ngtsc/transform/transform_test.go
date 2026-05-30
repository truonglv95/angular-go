package transform_test

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/printer"
)

type mockReflectionHost struct {
	decorators map[*ast.Node][]reflection.Decorator
}

func (m *mockReflectionHost) GetDecoratorsOfDeclaration(declaration *ast.Node) []reflection.Decorator {
	return m.decorators[declaration]
}
func (m *mockReflectionHost) GetMembersOfClass(clazz *ast.Node) []reflection.ClassMember { return nil }
func (m *mockReflectionHost) GetConstructorParameters(clazz *ast.Node) []reflection.CtorParameter {
	return nil
}
func (m *mockReflectionHost) GetDefinitionOfFunction(fn *ast.Node) *reflection.FunctionDefinition {
	return nil
}
func (m *mockReflectionHost) GetImportOfIdentifier(id *ast.Node) *reflection.Import { return nil }
func (m *mockReflectionHost) GetDeclarationOfIdentifier(id *ast.Node) *reflection.Declaration {
	return nil
}
func (m *mockReflectionHost) GetExportsOfModule(module *ast.Node) map[string]reflection.Declaration {
	return nil
}
func (m *mockReflectionHost) IsClass(node *ast.Node) bool                            { return true }
func (m *mockReflectionHost) HasBaseClass(clazz *ast.Node) bool                       { return false }
func (m *mockReflectionHost) GetBaseClassExpression(clazz *ast.Node) *ast.Node        { return nil }
func (m *mockReflectionHost) GetGenericArityOfClass(clazz *ast.Node) int              { return 0 }
func (m *mockReflectionHost) GetVariableValue(declaration *ast.Node) *ast.Node       { return nil }
func (m *mockReflectionHost) IsStaticallyExported(decl *ast.Node) bool                { return true }

func TestTransformSourceFile_Component(t *testing.T) {
	sourceText := `
@Component({
	selector: 'app-root',
	template: '<h1>Hello</h1>'
})
export class AppComponent {}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/app.component.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)
	if sourceFile == nil {
		t.Fatal("Failed to parse source file")
	}

	var classNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind == ast.KindClassDeclaration {
			classNode = stmt
			break
		}
	}

	if classNode == nil {
		t.Fatal("Could not find class in AST")
	}

	host := &mockReflectionHost{
		decorators: make(map[*ast.Node][]reflection.Decorator),
	}

	// We populate the decorator for the class node so our host returns it:
	decNode := classNode.Modifiers().Nodes[0]

	// Decorator Args
	var args []*ast.Node
	if decNode.Kind == ast.KindDecorator {
		decCall := decNode.AsDecorator().Expression
		if decCall.Kind == ast.KindCallExpression {
			args = decCall.AsCallExpression().Arguments.Nodes
		}
	}

	host.decorators[classNode] = []reflection.Decorator{
		{
			Name: "Component",
			Node: decNode,
			Args: args,
		},
	}

	// Run our transformer!
	transform.TransformSourceFile(sourceFile, host)

	// Now serialize the AST to string
	writer := printer.NewTextWriter("\n", 2)
	p := printer.NewPrinter(printer.PrinterOptions{RemoveComments: false}, printer.PrintHandlers{}, nil)
	p.Write(sourceFile.AsNode(), sourceFile, writer, nil)
	output := writer.String()

	t.Logf("Transformed Output:\n%s", output)

	// Assertions!
	// 1. Should prepend import * as i0 from '@angular/core';
	if !strings.Contains(output, "import * as i0 from \"@angular/core\";") && !strings.Contains(output, "import * as i0 from '@angular/core';") {
		t.Errorf("Expected import from @angular/core, got:\n%s", output)
	}

	// 2. Decorator @Component should be stripped from the class modifiers
	if strings.Contains(output, "@Component") {
		t.Errorf("Expected decorator to be stripped, but found @Component in:\n%s", output)
	}

	// 3. The class should have a static ɵcmp member
	if !strings.Contains(output, "static ɵcmp = i0.ɵɵdefineComponent") {
		t.Errorf("Expected static ɵcmp = i0.ɵɵdefineComponent, got:\n%s", output)
	}
}
