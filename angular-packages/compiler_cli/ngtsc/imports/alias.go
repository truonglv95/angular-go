package imports

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/internal/ast"
)

type AliasingHost interface {
	AliasExportsInDts() bool
	MaybeAliasSymbolAs(ref *Reference, context ast.SourceFile, ngModuleName string, isReExport bool) string
	GetAliasIn(decl ast.Node, via ast.SourceFile, isReExport bool) output.Expression
}

type UnifiedModulesAliasingHost struct {
	AliasExportsInDts any
}

func (recv *UnifiedModulesAliasingHost) MaybeAliasSymbolAs(ref *Reference, context ast.SourceFile, ngModuleName string, isReExport bool) string {
	// TODO: stub
	panic("unimplemented")
}

func (recv *UnifiedModulesAliasingHost) GetAliasIn(decl ast.Node, via ast.SourceFile, isReExport bool) output.Expression {
	// TODO: stub
	panic("unimplemented")
}

type PrivateExportAliasingHost struct {
	AliasExportsInDts any
}

func (recv *PrivateExportAliasingHost) MaybeAliasSymbolAs(ref *Reference, context ast.SourceFile, ngModuleName string) string {
	// TODO: stub
	panic("unimplemented")
}

func (recv *PrivateExportAliasingHost) GetAliasIn() any {
	// TODO: stub
	panic("unimplemented")
}

type AliasStrategy struct {
}

func (recv *AliasStrategy) Emit(ref *Reference, context ast.SourceFile, importMode ImportFlags) EmittedReference {
	// TODO: stub
	panic("unimplemented")
}
