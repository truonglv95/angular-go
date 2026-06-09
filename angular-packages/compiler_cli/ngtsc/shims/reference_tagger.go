package shims

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
)

type ShimReferenceTagger struct {
	suffixes []string
	tagged   map[string]bool
	enabled  bool
}

func NewShimReferenceTagger(shimExtensions []string) *ShimReferenceTagger {
	var suffixes []string
	for _, ext := range shimExtensions {
		suffixes = append(suffixes, "."+ext+".ts")
	}
	return &ShimReferenceTagger{
		suffixes: suffixes,
		tagged:   make(map[string]bool),
		enabled:  true,
	}
}

func (recv *ShimReferenceTagger) Tag(sf *ast.SourceFile) {
	if sf == nil || !recv.enabled || sf.IsDeclarationFile || IsShim(sf) || recv.tagged[sf.FileName()] {
		return
	}

	// Only tag .ts and .tsx files
	if !strings.HasSuffix(sf.FileName(), ".ts") && !strings.HasSuffix(sf.FileName(), ".tsx") {
		return
	}
	// Do not tag declaration files
	if strings.HasSuffix(sf.FileName(), ".d.ts") {
		return
	}

	// Capture original referenced files if not already done
	if _, ok := originalReferencedFilesMap[sf.FileName()]; !ok {
		originalReferencedFilesMap[sf.FileName()] = sf.ReferencedFiles
	}

	referencedFiles := append([]*ast.FileReference(nil), originalReferencedFilesMap[sf.FileName()]...)

	sfPath := sf.FileName()
	if strings.HasSuffix(sfPath, ".tsx") {
		sfPath = sfPath[:len(sfPath)-4]
	} else if strings.HasSuffix(sfPath, ".ts") {
		sfPath = sfPath[:len(sfPath)-3]
	}

	for _, suffix := range recv.suffixes {
		referencedFiles = append(referencedFiles, &ast.FileReference{
			FileName: sfPath + suffix,
		})
	}

	sf.ReferencedFiles = referencedFiles
	recv.tagged[sf.FileName()] = true
}

func (recv *ShimReferenceTagger) Finalize() {
	recv.enabled = false
	recv.tagged = make(map[string]bool)
}
