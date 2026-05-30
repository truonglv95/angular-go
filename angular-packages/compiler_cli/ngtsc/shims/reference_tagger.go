package shims

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type ShimReferenceTagger struct {
}

func (recv *ShimReferenceTagger) Tag(sf ast.SourceFile) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *ShimReferenceTagger) Finalize() any {
	// TODO: stub
	panic("unimplemented")
}
