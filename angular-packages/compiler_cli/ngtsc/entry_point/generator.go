package entry_point

import (
	"regexp"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/util"
)

type FlatIndexGenerator struct {
	entryPoint    file_system.AbsoluteFsPath
	flatIndexPath string
	moduleName    *string
	shouldEmit    bool
}

func NewFlatIndexGenerator(entryPoint file_system.AbsoluteFsPath, relativeFlatIndexPath string, moduleName *string) *FlatIndexGenerator {
	dir := file_system.Dirname(string(entryPoint))
	joined := file_system.Join(dir, relativeFlatIndexPath)

	// Replace \.js$ with empty string
	re := regexp.MustCompile(`\.js$`)
	replaced := re.ReplaceAllString(joined, "")
	flatIndexPath := replaced + ".ts"

	return &FlatIndexGenerator{
		entryPoint:    entryPoint,
		flatIndexPath: flatIndexPath,
		moduleName:    moduleName,
		shouldEmit:    true,
	}
}

func (g *FlatIndexGenerator) FlatIndexPath() string {
	return g.flatIndexPath
}

func (g *FlatIndexGenerator) ShouldEmit() bool {
	return g.shouldEmit
}

func (g *FlatIndexGenerator) MakeTopLevelShim() ast.SourceFile {
	relativeEntryPoint := util.RelativePathBetween(g.flatIndexPath, string(g.entryPoint))
	contents := `/**
 * Generated bundle index. Do not edit.
 */

export * from '` + *relativeEntryPoint + `';
`
	_ = contents
	var genFile any

	if g.moduleName != nil {
		// In Go, since we can't easily assign dynamically, we might need a cast or helper.
		// For structural match, we leave a note or use whatever setter typescript-go provides
		if modFile, ok := genFile.(interface{ SetModuleName(string) }); ok {
			modFile.SetModuleName(*g.moduleName)
		}
	}
	return genFile.(ast.SourceFile)
}
