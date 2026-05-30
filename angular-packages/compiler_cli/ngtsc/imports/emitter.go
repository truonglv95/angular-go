package imports

import (
	"errors"
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

type ImportFlags int

const (
	None                    ImportFlags = 0x00
	ForceNewImport          ImportFlags = 0x01
	NoAliasing              ImportFlags = 0x02
	AllowTypeImports        ImportFlags = 0x04
	AllowRelativeDtsImports ImportFlags = 0x08
	AllowAmbientReferences  ImportFlags = 0x10
)

type ImportedFile = *ast.SourceFile

type ReferenceEmitKind int

const (
	Success ReferenceEmitKind = iota
	Failed
)

type EmittedReference struct {
	Kind         ReferenceEmitKind
	Expression   output.Expression
	ImportedFile ImportedFile
}

type FailedEmitResult struct {
	Kind    ReferenceEmitKind
	Ref     *Reference
	Context *ast.SourceFile
	Reason  string
}

type ReferenceEmitResult interface {
	GetKind() ReferenceEmitKind
}

func (r *EmittedReference) GetKind() ReferenceEmitKind { return r.Kind }
func (r *FailedEmitResult) GetKind() ReferenceEmitKind  { return r.Kind }

func AssertSuccessfulReferenceEmit(result ReferenceEmitResult, origin *ast.Node, typeKind string) (*EmittedReference, error) {
	if result.GetKind() == Success {
		return result.(*EmittedReference), nil
	}
	failed := result.(*FailedEmitResult)
	message := fmt.Sprintf("Unable to import %s %s. Reason: %s", typeKind, failed.Ref.DebugName(), failed.Reason)
	return nil, errors.New(message)
}

type ReferenceEmitStrategy interface {
	Emit(ref *Reference, context *ast.SourceFile, importFlags ImportFlags) ReferenceEmitResult
}

type ReferenceEmitter struct {
	strategies []ReferenceEmitStrategy
}

func NewReferenceEmitter(strategies []ReferenceEmitStrategy) *ReferenceEmitter {
	return &ReferenceEmitter{
		strategies: strategies,
	}
}

func (recv *ReferenceEmitter) Emit(ref *Reference, context *ast.SourceFile, importFlags ImportFlags) ReferenceEmitResult {
	for _, strategy := range recv.strategies {
		emitted := strategy.Emit(ref, context, importFlags)
		if emitted != nil {
			return emitted
		}
	}

	return &FailedEmitResult{
		Kind:    Failed,
		Ref:     ref,
		Context: context,
		Reason:  fmt.Sprintf("Unable to write a reference to %s.", ref.DebugName()),
	}
}

type LocalIdentifierStrategy struct {
}

func NewLocalIdentifierStrategy() *LocalIdentifierStrategy {
	return &LocalIdentifierStrategy{}
}

func (recv *LocalIdentifierStrategy) Emit(ref *Reference, context *ast.SourceFile, importFlags ImportFlags) ReferenceEmitResult {
	refSf := getSourceFile(ref.Node)

	if (importFlags&ForceNewImport) != 0 && refSf != context {
		return nil
	}

	isDecl := isValueDeclaration(ref.Node) || isTypeDeclaration(ref.Node)
	if !isDecl && refSf == context {
		return &EmittedReference{
			Kind:         Success,
			Expression:   output.NewWrappedNodeExpr(ref.Node, nil, nil, nil),
			ImportedFile: nil,
		}
	}

	if ref.IsAmbient && (importFlags&AllowAmbientReferences) != 0 {
		identifier := identifierOfNode(ref.Node)
		if identifier != nil {
			return &EmittedReference{
				Kind:         Success,
				Expression:   output.NewWrappedNodeExpr(identifier, nil, nil, nil),
				ImportedFile: nil,
			}
		}
		return nil
	}

	identifier := ref.GetIdentityIn(context)
	if identifier != nil {
		return &EmittedReference{
			Kind:         Success,
			Expression:   output.NewWrappedNodeExpr(identifier, nil, nil, nil),
			ImportedFile: nil,
		}
	}

	return nil
}

type ModuleExports struct {
	Module    *ast.SourceFile
	ExportMap map[*ast.Node]string
}

type AbsoluteModuleStrategy struct {
	moduleResolver     *ModuleResolver
	reflectionHost     reflection.ReflectionHost
	moduleExportsCache map[string]*ModuleExports
}

func NewAbsoluteModuleStrategy(
	moduleResolver *ModuleResolver,
	reflectionHost reflection.ReflectionHost,
) *AbsoluteModuleStrategy {
	return &AbsoluteModuleStrategy{
		moduleResolver:     moduleResolver,
		reflectionHost:     reflectionHost,
		moduleExportsCache: make(map[string]*ModuleExports),
	}
}

func (recv *AbsoluteModuleStrategy) Emit(ref *Reference, context *ast.SourceFile, importFlags ImportFlags) ReferenceEmitResult {
	if ref.BestGuessOwningModule == nil {
		return nil
	}

	isDecl := isValueDeclaration(ref.Node) || isTypeDeclaration(ref.Node)
	if !isDecl {
		panic(fmt.Sprintf("Debug assert: unable to import a Reference to non-declaration of type %v.", ref.Node.Kind))
	}

	if (importFlags&AllowTypeImports) == 0 && isTypeDeclaration(ref.Node) {
		panic(fmt.Sprintf("Importing a type-only declaration of type %v in a value position is not allowed.", ref.Node.Kind))
	}

	specifier := ref.BestGuessOwningModule.Specifier
	resolutionContext := ref.BestGuessOwningModule.ResolutionContext

	exports := recv.getExportsOfModule(specifier, resolutionContext)
	if exports.Module == nil {
		return &FailedEmitResult{
			Kind:    Failed,
			Ref:     ref,
			Context: context,
			Reason:  fmt.Sprintf("The module '%s' could not be found.", specifier),
		}
	}

	if exports.ExportMap == nil {
		return &FailedEmitResult{
			Kind:    Failed,
			Ref:     ref,
			Context: context,
			Reason:  fmt.Sprintf("The symbol is not exported from %s (module '%s').", exports.Module.FileName(), specifier),
		}
	}

	symbolName, ok := exports.ExportMap[ref.Node]
	if !ok {
		return &FailedEmitResult{
			Kind:    Failed,
			Ref:     ref,
			Context: context,
			Reason:  fmt.Sprintf("The symbol is not exported from %s (module '%s').", exports.Module.FileName(), specifier),
		}
	}

	refVal := output.ExternalReference{
		ModuleName: &specifier,
		Name:       &symbolName,
	}

	return &EmittedReference{
		Kind:         Success,
		Expression:   output.NewExternalExpr(refVal, nil, nil, nil, nil),
		ImportedFile: exports.Module,
	}
}

func (recv *AbsoluteModuleStrategy) getExportsOfModule(moduleName string, fromFile string) *ModuleExports {
	if _, ok := recv.moduleExportsCache[moduleName]; !ok {
		recv.moduleExportsCache[moduleName] = recv.enumerateExportsOfModule(moduleName, fromFile)
	}
	return recv.moduleExportsCache[moduleName]
}

func (recv *AbsoluteModuleStrategy) enumerateExportsOfModule(specifier string, fromFile string) *ModuleExports {
	entryPointFile := recv.moduleResolver.ResolveModule(specifier, fromFile)
	if entryPointFile == nil {
		return &ModuleExports{Module: nil, ExportMap: nil}
	}

	exports := recv.reflectionHost.GetExportsOfModule(entryPointFile.AsNode())
	if exports == nil {
		return &ModuleExports{Module: entryPointFile, ExportMap: nil}
	}

	exportMap := make(map[*ast.Node]string)
	for name, declaration := range exports {
		if existing, ok := exportMap[declaration.Node]; ok {
			if id := identifierOfNode(declaration.Node); id != nil && id.AsIdentifier().Text == existing {
				continue
			}
		}
		exportMap[declaration.Node] = name
	}

	return &ModuleExports{Module: entryPointFile, ExportMap: exportMap}
}

type LogicalProjectStrategy struct {
	reflector            reflection.ReflectionHost
	logicalFs            *file_system.LogicalFileSystem
	relativePathStrategy *RelativePathStrategy
}

func NewLogicalProjectStrategy(
	reflector reflection.ReflectionHost,
	logicalFs *file_system.LogicalFileSystem,
) *LogicalProjectStrategy {
	return &LogicalProjectStrategy{
		reflector:            reflector,
		logicalFs:            logicalFs,
		relativePathStrategy: NewRelativePathStrategy(reflector),
	}
}

func (recv *LogicalProjectStrategy) Emit(ref *Reference, context *ast.SourceFile, importFlags ImportFlags) ReferenceEmitResult {
	destSf := getSourceFile(ref.Node)

	destPath := recv.logicalFs.LogicalPathOfSf(*destSf)
	if destPath == nil {
		if destSf.IsDeclarationFile && (importFlags&AllowRelativeDtsImports) != 0 {
			return recv.relativePathStrategy.Emit(ref, context, importFlags)
		}

		return &FailedEmitResult{
			Kind:    Failed,
			Ref:     ref,
			Context: context,
			Reason:  fmt.Sprintf("The file %s is outside of the configured 'rootDir'.", destSf.FileName()),
		}
	}

	originPath := recv.logicalFs.LogicalPathOfSf(*context)
	if originPath == nil {
		panic(fmt.Sprintf("Debug assert: attempt to import from %s but it's outside the program?", context.FileName()))
	}

	if *destPath == *originPath {
		return nil
	}

	name := FindExportedNameOfNode(ref.Node, destSf, recv.reflector)
	if name == nil {
		return &FailedEmitResult{
			Kind:    Failed,
			Ref:     ref,
			Context: context,
			Reason:  fmt.Sprintf("The symbol is not exported from %s.", destSf.FileName()),
		}
	}

	moduleName := string(file_system.LogicalProjectPathRelativePathBetween(*originPath, *destPath))
	refVal := output.ExternalReference{
		ModuleName: &moduleName,
		Name:       name,
	}

	return &EmittedReference{
		Kind:         Success,
		Expression:   output.NewExternalExpr(refVal, nil, nil, nil, nil),
		ImportedFile: destSf,
	}
}

type RelativePathStrategy struct {
	reflector reflection.ReflectionHost
}

func NewRelativePathStrategy(reflector reflection.ReflectionHost) *RelativePathStrategy {
	return &RelativePathStrategy{
		reflector: reflector,
	}
}

func (recv *RelativePathStrategy) Emit(ref *Reference, context *ast.SourceFile, importFlags ImportFlags) ReferenceEmitResult {
	destSf := getSourceFile(ref.Node)
	relativePath := file_system.Relative(
		file_system.Dirname(string(file_system.AbsoluteFromSourceFile(context))),
		string(file_system.AbsoluteFromSourceFile(destSf)),
	)
	moduleName := file_system.ToRelativeImport(stripExtension(relativePath))

	name := FindExportedNameOfNode(ref.Node, destSf, recv.reflector)
	if name == nil {
		return &FailedEmitResult{
			Kind:    Failed,
			Ref:     ref,
			Context: context,
			Reason:  fmt.Sprintf("The symbol is not exported from %s.", destSf.FileName()),
		}
	}

	refVal := output.ExternalReference{
		ModuleName: &moduleName,
		Name:       name,
	}

	return &EmittedReference{
		Kind:         Success,
		Expression:   output.NewExternalExpr(refVal, nil, nil, nil, nil),
		ImportedFile: destSf,
	}
}

type UnifiedModulesHost interface {
	FileNameToModuleName(importedFilePath string, containingFilePath string) string
}

type UnifiedModulesStrategy struct {
	reflector          reflection.ReflectionHost
	unifiedModulesHost UnifiedModulesHost
}

func NewUnifiedModulesStrategy(
	reflector reflection.ReflectionHost,
	unifiedModulesHost UnifiedModulesHost,
) *UnifiedModulesStrategy {
	return &UnifiedModulesStrategy{
		reflector:          reflector,
		unifiedModulesHost: unifiedModulesHost,
	}
}

func (recv *UnifiedModulesStrategy) Emit(ref *Reference, context *ast.SourceFile, importFlags ImportFlags) ReferenceEmitResult {
	destSf := getSourceFile(ref.Node)
	name := FindExportedNameOfNode(ref.Node, destSf, recv.reflector)
	if name == nil {
		return nil
	}

	moduleName := recv.unifiedModulesHost.FileNameToModuleName(destSf.FileName(), context.FileName())
	refVal := output.ExternalReference{
		ModuleName: &moduleName,
		Name:       name,
	}

	return &EmittedReference{
		Kind:         Success,
		Expression:   output.NewExternalExpr(refVal, nil, nil, nil, nil),
		ImportedFile: destSf,
	}
}

func isValueDeclaration(node *ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case ast.KindClassDeclaration, ast.KindFunctionDeclaration, ast.KindVariableDeclaration:
		return true
	}
	return false
}

func isTypeDeclaration(node *ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case ast.KindInterfaceDeclaration, ast.KindTypeAliasDeclaration:
		return true
	}
	return false
}

func stripExtension(path string) string {
	if suffix := ".tsx"; len(path) > len(suffix) && path[len(path)-len(suffix):] == suffix {
		return path[:len(path)-len(suffix)]
	}
	if suffix := ".ts"; len(path) > len(suffix) && path[len(path)-len(suffix):] == suffix {
		return path[:len(path)-len(suffix)]
	}
	if suffix := ".d.ts"; len(path) > len(suffix) && path[len(path)-len(suffix):] == suffix {
		return path[:len(path)-len(suffix)]
	}
	return path
}
