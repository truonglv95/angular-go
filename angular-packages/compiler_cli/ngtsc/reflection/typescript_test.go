package reflection

import (
	"context"
	"testing"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/typecheck"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/file_system"
	"github.com/microsoft/typescript-go/internal/program_driver"
)

func TestGetBaseClassExpression(t *testing.T) {
	fs := file_system.NewTestingFileSystem()
	fs.WriteFile(file_system.AbsoluteFsPath("/main.ts"), `
		import { Component } from '@angular/core';
		class BaseComponent {}
		class ChildComponent extends BaseComponent {}
	`)
	
	host := program_driver.NewInMemoryCompilerHost(fs, file_system.AbsoluteFsPath("/"))
	prog := compiler.NewProgram([]string{"/main.ts"}, compiler.CompilerOptions{}, host, nil)
	chk, _ := prog.GetTypeChecker(context.Background())
	
	refHost := NewTypeScriptReflectionHost(chk)
	
	sf := prog.GetSourceFile("/main.ts")
	var childClass ClassDeclaration
	sf.ForEachChild(func(n *ast.Node) bool {
		if n.Kind == ast.KindClassDeclaration && n.AsClassDeclaration().Name() != nil && n.AsClassDeclaration().Name().AsIdentifier().Text == "ChildComponent" {
			childClass = n
			return true
		}
		return false
	})
	
	if childClass == nil {
		t.Fatalf("ChildComponent not found")
	}
	
	baseExpr := refHost.GetBaseClassExpression(childClass)
	if baseExpr == nil {
		t.Fatalf("GetBaseClassExpression returned nil")
	}
	
	decl := refHost.GetDeclarationOfIdentifier(baseExpr)
	if decl == nil {
		t.Fatalf("GetDeclarationOfIdentifier returned nil for baseExpr")
	}
	
	if decl.Node.Kind != ast.KindClassDeclaration {
		t.Fatalf("Expected base class declaration, got %v", decl.Node.Kind)
	}
}
