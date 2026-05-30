package shims

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

var NgExtension any

type NgExtensionData interface {
}

type NgExtendedSourceFile interface {
}

func IsExtended(sf ast.SourceFile) any {
	// TODO: stub
	panic("unimplemented")
}

func SfExtensionData(sf ast.SourceFile) NgExtensionData {
	// TODO: stub
	panic("unimplemented")
}

type NgFileShimData interface {
}

type NgFileShimSourceFile interface {
}

func IsFileShimSourceFile(sf ast.SourceFile) any {
	// TODO: stub
	panic("unimplemented")
}

func IsShim(sf ast.SourceFile) bool {
	// TODO: stub
	panic("unimplemented")
}

func CopyFileShimData(from ast.SourceFile, to ast.SourceFile) any {
	// TODO: stub
	panic("unimplemented")
}

func UntagAllTsFiles(program any) any {
	// TODO: stub
	panic("unimplemented")
}

func RetagAllTsFiles(program any) any {
	// TODO: stub
	panic("unimplemented")
}

func UntagTsFile(sf ast.SourceFile) any {
	// TODO: stub
	panic("unimplemented")
}

func RetagTsFile(sf ast.SourceFile) any {
	// TODO: stub
	panic("unimplemented")
}
