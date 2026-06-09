package indexer

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/internal/ast"
)

// GenerateAnalysis generates IndexedComponent entries from an IndexingContext,
// which has information about components discovered in the program registered in it.
// The context must be populated before GenerateAnalysis is called.
func GenerateAnalysis(context *IndexingContext, adapter NodeAdapter) map[*ast.Node]IndexedComponent {
	analysis := make(map[*ast.Node]IndexedComponent)

	for _, comp := range context.Components {
		name := adapter.GetName(comp.Declaration)
		fileName := adapter.GetFileName(comp.Declaration)

		// Get source files for the component and the template. If the template is inline, its source
		// file is the component's.
		componentFile := parse_util.NewParseSourceFile(adapter.GetContent(comp.Declaration), fileName)
		var templateFile *parse_util.ParseSourceFile
		if comp.TemplateMeta.IsInline {
			templateFile = componentFile
		} else {
			templateFile = comp.TemplateMeta.File
		}

		identifiers, errors := getTemplateIdentifiers(comp.BoundTemplate)

		analysis[comp.Declaration] = IndexedComponent{
			Name:     name,
			Selector: comp.Selector,
			File:     componentFile,
			Template: IndexedTemplate{
				Identifiers: identifiers,
				File:        templateFile,
			},
			Errors:   errors,
		}
	}

	return analysis
}
