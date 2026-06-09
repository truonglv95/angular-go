package shims

import (
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
)

type CompilerHost interface {
	GetSourceFile(fileName string, languageVersion any, onError func(message string), shouldCreateNewSourceFile bool) *ast.SourceFile
	FileExists(fileName string) bool
}

type Program interface {
	GetSourceFiles() []*ast.SourceFile
}

type ShimAdapter struct {
	delegate          CompilerHost
	tsRootFiles       []string
	perFileGenerators []PerFileShimGenerator
	oldProgram        Program
	shims             map[string]*ast.SourceFile
	priorShims        map[string]*ast.SourceFile
	notShims          map[string]bool
	generators        []shimGeneratorData

	IgnoreForEmit     map[*ast.SourceFile]bool
	ExtraInputFiles   []string
	ExtensionPrefixes []string
}

type shimGeneratorData struct {
	generator PerFileShimGenerator
	test      *regexp.Regexp
	suffix    string
}

func NewShimAdapter(
	delegate CompilerHost,
	tsRootFiles []string,
	topLevelGenerators []TopLevelShimGenerator,
	perFileGenerators []PerFileShimGenerator,
	oldProgram Program,
) *ShimAdapter {
	adapter := &ShimAdapter{
		delegate:          delegate,
		tsRootFiles:       tsRootFiles,
		perFileGenerators: perFileGenerators,
		oldProgram:        oldProgram,
		shims:             make(map[string]*ast.SourceFile),
		priorShims:        make(map[string]*ast.SourceFile),
		notShims:          make(map[string]bool),
		IgnoreForEmit:     make(map[*ast.SourceFile]bool),
	}

	for _, gen := range perFileGenerators {
		pattern := `^(.*)\.` + regexp.QuoteMeta(gen.ExtensionPrefix()) + `\.ts$`
		regex := regexp.MustCompile(pattern)
		adapter.generators = append(adapter.generators, shimGeneratorData{
			generator: gen,
			test:      regex,
			suffix:    "." + gen.ExtensionPrefix() + ".ts",
		})
		adapter.ExtensionPrefixes = append(adapter.ExtensionPrefixes, gen.ExtensionPrefix())
	}

	for _, gen := range topLevelGenerators {
		sf := gen.MakeTopLevelShim()
		if sf != nil {
			if !gen.ShouldEmit() {
				adapter.IgnoreForEmit[sf] = true
			}
			fileName := sf.FileName()
			adapter.shims[fileName] = sf
			adapter.ExtraInputFiles = append(adapter.ExtraInputFiles, fileName)
		}
	}

	for _, rootFile := range tsRootFiles {
		for _, record := range adapter.generators {
			suffixName := rootFile + record.suffix
			adapter.ExtraInputFiles = append(adapter.ExtraInputFiles, suffixName)
		}
	}

	if oldProgram != nil {
		for _, oldSf := range oldProgram.GetSourceFiles() {
			if oldSf != nil && !oldSf.IsDeclarationFile {
				if adapter.isShim(oldSf.FileName()) {
					adapter.priorShims[oldSf.FileName()] = oldSf
				}
			}
		}
	}

	return adapter
}

func (recv *ShimAdapter) isShim(fileName string) bool {
	for _, record := range recv.generators {
		if record.test.MatchString(fileName) {
			return true
		}
	}
	return false
}

func (recv *ShimAdapter) MaybeGenerate(fileName string) *ast.SourceFile {
	if recv.notShims[fileName] {
		return nil
	}
	if sf, ok := recv.shims[fileName]; ok {
		return sf
	}

	if strings.HasSuffix(fileName, ".d.ts") {
		recv.notShims[fileName] = true
		return nil
	}

	for _, record := range recv.generators {
		match := record.test.FindStringSubmatch(fileName)
		if len(match) < 2 {
			continue
		}

		prefix := match[1]
		baseFileName := prefix + ".ts"
		inputFile := recv.delegate.GetSourceFile(baseFileName, nil, nil, false)
		if inputFile == nil {
			baseFileName = prefix + ".tsx"
			inputFile = recv.delegate.GetSourceFile(baseFileName, nil, nil, false)
		}

		if inputFile == nil {
			return nil
		}

		return recv.generateSpecific(fileName, record.generator, inputFile)
	}

	recv.notShims[fileName] = true
	return nil
}

func (recv *ShimAdapter) generateSpecific(
	fileName string,
	generator PerFileShimGenerator,
	inputFile *ast.SourceFile,
) *ast.SourceFile {
	var priorShimSf *ast.SourceFile = nil
	if sf, ok := recv.priorShims[fileName]; ok {
		priorShimSf = sf
		delete(recv.priorShims, fileName)
	}

	shimSf := generator.GenerateShimForFile(inputFile, fileName, priorShimSf)
	if !generator.ShouldEmit() {
		recv.IgnoreForEmit[shimSf] = true
	}

	recv.shims[fileName] = shimSf
	return shimSf
}
