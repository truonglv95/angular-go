package main

import (
	"fmt"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
)

func main() {
	src := "export class Cmp {}"
	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, src, core.ScriptKindTS)
	classDecl := sf.Statements.Nodes[0].AsClassDeclaration()
	fmt.Printf("Kind: %v\n", classDecl.Name().Kind)
	fmt.Printf("IsIdentifier: %v\n", ast.IsIdentifier(classDecl.Name()))
}
