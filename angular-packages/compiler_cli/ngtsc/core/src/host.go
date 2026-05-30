package src



import (
)

// DelegatingCompilerHost delegates to a TypeScript CompilerHost
type DelegatingCompilerHost struct {
	Delegate any
}

// NgCompilerHost wraps ts.CompilerHost
type NgCompilerHost struct {
	*DelegatingCompilerHost

	EntryPoint              *string // AbsoluteFsPath
	ConstructionDiagnostics []any

	InputFiles []string
	RootDirs   []string // AbsoluteFsPath

	// shimAdapter ShimAdapter
	// shimTagger  ShimReferenceTagger
}

func NewNgCompilerHost(
	delegate any,
	inputFiles []string,
	rootDirs []string,
	entryPoint *string,
	diagnostics []any,
) *NgCompilerHost {
	return &NgCompilerHost{
		DelegatingCompilerHost:  &DelegatingCompilerHost{Delegate: delegate},
		EntryPoint:              entryPoint,
		ConstructionDiagnostics: diagnostics,
		InputFiles:              inputFiles, // + shimAdapter.extraInputFiles
		RootDirs:                rootDirs,
	}
}

func (h *NgCompilerHost) IgnoreForEmit() map[any]struct{} {
	return nil // h.shimAdapter.ignoreForEmit
}

func (h *NgCompilerHost) ShimExtensionPrefixes() []string {
	return nil // h.shimAdapter.extensionPrefixes
}

func (h *NgCompilerHost) PostProgramCreationCleanup() {
	// h.shimTagger.finalize()
}

func NgCompilerHostWrap(
	delegate any,
	inputFiles []string,
	options any, // NgCompilerOptions
	oldProgram any,
) *NgCompilerHost {
	// dummy implementation
	return NewNgCompilerHost(delegate, inputFiles, nil, nil, nil)
}

func (h *NgCompilerHost) IsShim(sf any) bool {
	return false // isShim(sf)
}

func (h *NgCompilerHost) IsResource(sf any) bool {
	return false
}

func (h *NgCompilerHost) GetSourceFile(
	fileName string,
	languageVersionOrOptions any,
	onError func(message string),
	shouldCreateNewSourceFile bool,
) any {
	return nil
}

func (h *NgCompilerHost) FileExists(fileName string) bool {
	return false
}

func (h *NgCompilerHost) UnifiedModulesHost() any {
	return nil
}

func (h *NgCompilerHost) CreateCachedResolveModuleNamesFunction() any {
	return nil
}
