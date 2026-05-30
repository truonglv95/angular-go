package shims

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type ShimAdapter struct {
	IgnoreForEmit     any
	ExtraInputFiles   any
	ExtensionPrefixes []string
}

func (recv *ShimAdapter) MaybeGenerate(fileName string) ast.SourceFile {
	// TODO: stub
	panic("unimplemented")
}
