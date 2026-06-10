package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations_local"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/perf"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/typecheck"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/tspath"
)

type NgCompiler struct {
	tsProgram              *compiler.Program
	checker                *checker.Checker
	reflector              reflection.ReflectionHost
	handlers               []transform.DecoratorHandler
	traitCompiler          *transform.TraitCompiler
	incrementalCompilation *incremental.IncrementalCompilation
	affectedFiles          map[string]bool

	metaRegistry     *metadata.LocalMetadataRegistry
	scopeRegistry    *scope.LocalModuleScopeRegistry
	resourceRegistry *metadata.ResourceRegistry

	compilationMode string
	options         NgCompilerOptions

	analyzed      bool
	resolved      bool
	resolvedDiags []*ast.Diagnostic
	prepared      bool

	perfRecorder *perf.ActivePerfRecorder
}

func NewNgCompiler(tsProgram *compiler.Program, options NgCompilerOptions, oldCompiler *NgCompiler) (*NgCompiler, error) {
	ctx := context.Background()
	var chk *checker.Checker
	if options.CompilationMode != "local" {
		func() {
			chk, _ = tsProgram.GetTypeChecker(ctx)
		}()
	}
	refHost := reflection.NewTypeScriptReflectionHost(chk)

	localMetaRegistry := metadata.NewLocalMetadataRegistry()
	dtsMetaReader := metadata.NewDtsMetadataReader(chk)
	compoundMetaReader := metadata.NewCompoundMetadataReader([]metadata.MetadataReader{localMetaRegistry, dtsMetaReader})

	scopeRegistry := scope.NewLocalModuleScopeRegistry(compoundMetaReader)

	localRefHost := reflection.NewTypeScriptReflectionHost(nil)
	resourceRegistry := metadata.NewResourceRegistry()

	var handlers []transform.DecoratorHandler
	if options.CompilationMode == "local" {
		handlers = []transform.DecoratorHandler{
			annotations_local.NewComponentLocalDecoratorHandler(localRefHost, false, localMetaRegistry, scopeRegistry, options.EnableHmr, options.StyleIncludePaths),
			annotations_local.NewDirectiveLocalDecoratorHandler(localRefHost, localMetaRegistry),
			annotations_local.NewPipeLocalDecoratorHandler(localRefHost, localMetaRegistry), // LOCAL HANDLER
			annotations_local.NewInjectableLocalDecoratorHandler(localRefHost),
			annotations_local.NewNgModuleLocalDecoratorHandler(localRefHost, localMetaRegistry, scopeRegistry), // LOCAL HANDLER
		}
	} else {
		handlers = []transform.DecoratorHandler{
			annotations.NewComponentDecoratorHandler(refHost, false, localMetaRegistry, scopeRegistry, resourceRegistry, options.EnableHmr, options.StyleIncludePaths),
			annotations.NewDirectiveDecoratorHandler(refHost, localMetaRegistry),
			annotations.NewPipeDecoratorHandler(refHost, localMetaRegistry),
			annotations.NewInjectableDecoratorHandler(refHost), // GLOBAL HANDLER
			annotations.NewNgModuleDecoratorHandler(refHost, localMetaRegistry, scopeRegistry),
		}
	}

	var oldTc *transform.TraitCompiler
	var oldGraph *semantic_graph.SemanticDepGraph
	var oldIncrementalState *incremental.IncrementalState
	if oldCompiler != nil {
		oldTc = oldCompiler.traitCompiler
		if oldCompiler.incrementalCompilation != nil && oldCompiler.incrementalCompilation.State != nil {
			oldIncrementalState = oldCompiler.incrementalCompilation.State
			if sg, ok := oldIncrementalState.SemanticGraph.(*semantic_graph.SemanticDepGraph); ok {
				oldGraph = sg
			}
		}
	}
	traitCompiler := transform.NewTraitCompiler(handlers, refHost, localRefHost, oldTc)
	traitCompiler.SemanticUpdater = semantic_graph.NewSemanticDepGraphUpdater(oldGraph)

	newVersions := getFileVersions(tsProgram)
	var incrementalComp *incremental.IncrementalCompilation
	if oldIncrementalState != nil {
		incrementalComp = incremental.Incremental(
			tsProgram,
			newVersions,
			nil,
			oldIncrementalState,
			options.InvalidatedFiles,
			nil,
		)
	} else {
		incrementalComp = incremental.Fresh(newVersions)
	}
	traitCompiler.DepTracker = incrementalComp

	var affectedFiles map[string]bool
	if incrementalComp != nil && incrementalComp.AffectedFiles() != nil {
		affectedFiles = make(map[string]bool)
		for k, v := range incrementalComp.AffectedFiles() {
			affectedFiles[canonicalizePath(k)] = v
		}
	}

	return &NgCompiler{
		tsProgram:              tsProgram,
		checker:                chk,
		reflector:              refHost,
		handlers:               handlers,
		traitCompiler:          traitCompiler,
		incrementalCompilation: incrementalComp,
		affectedFiles:          affectedFiles,
		metaRegistry:           localMetaRegistry,
		scopeRegistry:          scopeRegistry,
		resourceRegistry:       resourceRegistry,
		compilationMode:        options.CompilationMode,
		options:                options,
	}, nil
}

// AnalyzeSync scans all source files and runs decorator detection + analysis.
// B#1 FIX: Both global and local modes now run in parallel using a worker pool.
func (c *NgCompiler) AnalyzeSync() []*ast.Diagnostic {
	if c.perfRecorder != nil {
		c.perfRecorder.Phase(perf.PerfPhase_Analysis)
		defer c.perfRecorder.Phase(perf.PerfPhase_Unaccounted)
	}
	if c.analyzed {
		return nil
	}

	sourceFiles := c.tsProgram.SourceFiles()
	if len(sourceFiles) == 0 {
		c.analyzed = true
		return nil
	}

	// Feed files into a channel for workers to consume
	ch := make(chan *ast.SourceFile, len(sourceFiles))
	for _, sf := range sourceFiles {
		ch <- sf
	}
	close(ch)

	// Use min(NumCPU, 16, len(files)) workers to avoid excessive goroutine overhead
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}
	if numWorkers > len(sourceFiles) {
		numWorkers = len(sourceFiles)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	if c.compilationMode == "local" {
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for sf := range ch {
					c.traitCompiler.AnalyzeSyncLocal(sf)
				}
			}()
		}
	} else {
		// Global mode: AnalyzeSync calls Checker.GetSymbolAtLocation which writes
		// into LinkStore maps — the TypeScript Checker is NOT goroutine-safe.
		// Run all source files through a SINGLE worker to avoid concurrent map writes.
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sf := range ch {
				c.traitCompiler.AnalyzeSync(sf)
			}
		}()
	}
	wg.Wait()

	// Register all traits (newly analyzed or reused)
	for classDecl, traits := range c.traitCompiler.GetClasses() {
		for _, trait := range traits {
			if trait.State == transform.TraitStateAnalyzed || trait.State == transform.TraitStateResolved {
				trait.Handler.Register(classDecl, trait.Analysis)
			}
		}
	}

	if c.traitCompiler.SemanticUpdater != nil {
		res := c.traitCompiler.SemanticUpdater.Finalize()
		if c.incrementalCompilation != nil && c.incrementalCompilation.State != nil {
			c.incrementalCompilation.State.SemanticGraph = c.traitCompiler.SemanticUpdater.GetGraph()
		}
		c.mergeSemanticAffectedFiles(res)
	}

	if c.incrementalCompilation != nil {
		c.incrementalCompilation.RecordSuccessfulAnalysis(c.traitCompiler)
	}

	c.analyzed = true
	return nil
}

func (c *NgCompiler) Resolve() []*ast.Diagnostic {
	if c.perfRecorder != nil {
		c.perfRecorder.Phase(perf.PerfPhase_Resolve)
		defer c.perfRecorder.Phase(perf.PerfPhase_Unaccounted)
	}
	if c.resolved {
		return c.resolvedDiags
	}
	c.traitCompiler.Resolve()

	if c.traitCompiler.SemanticUpdater != nil {
		res := c.traitCompiler.SemanticUpdater.Finalize()
		if c.incrementalCompilation != nil && c.incrementalCompilation.State != nil {
			c.incrementalCompilation.State.SemanticGraph = c.traitCompiler.SemanticUpdater.GetGraph()
		}
		c.mergeSemanticAffectedFiles(res)
	}

	var finalDiags []*ast.Diagnostic
	var tcbDiags map[string][]*ast.Diagnostic
	if c.options.StrictTemplates {
		tcbDiags = c.runTemplateTypeChecking()
	}

	// Map of all diagnostics per file
	fileDiagsMap := make(map[string][]*ast.Diagnostic)
	traitDiags := c.traitCompiler.GetDiagnostics()

	for _, sf := range c.tsProgram.SourceFiles() {
		fileName := sf.FileName()
		if c.isFileAffected(fileName) {
			var dList []*ast.Diagnostic
			if fd, ok := traitDiags[fileName]; ok {
				for i := range fd {
					dList = append(dList, &fd[i])
				}
			}
			if tcbDiags != nil {
				if td, ok := tcbDiags[fileName]; ok {
					dList = append(dList, td...)
				}
			}
			fileDiagsMap[fileName] = dList
		} else {
			if c.incrementalCompilation != nil {
				prior := c.incrementalCompilation.PriorTypeCheckingResultsFor(sf)
				if prior != nil {
					if pd, ok := prior.([]ast.Diagnostic); ok {
						var dList []*ast.Diagnostic
						for i := range pd {
							dList = append(dList, &pd[i])
						}
						fileDiagsMap[fileName] = dList
					}
				}
			}
		}
	}

	// Flatten all diagnostics to finalDiags
	for _, diags := range fileDiagsMap {
		finalDiags = append(finalDiags, diags...)
	}

	if c.incrementalCompilation != nil {
		res := make(map[string]any)
		for file, diags := range fileDiagsMap {
			var astDiags []ast.Diagnostic
			for _, d := range diags {
				astDiags = append(astDiags, *d)
			}
			res[file] = astDiags
		}
		c.incrementalCompilation.RecordSuccessfulTypeCheck(res)
	}

	c.resolvedDiags = finalDiags
	c.resolved = true
	return finalDiags
}

func (c *NgCompiler) mergeSemanticAffectedFiles(res semantic_graph.SemanticDependencyResult) {
	if res == nil {
		return
	}
	if c.affectedFiles == nil {
		c.affectedFiles = make(map[string]bool)
	}
	for _, f := range res.GetAffectedFiles() {
		c.affectedFiles[canonicalizePath(f)] = true
	}
}

func (c *NgCompiler) PrepareEmit() []*ast.Diagnostic {
	if c.perfRecorder != nil {
		c.perfRecorder.Phase(perf.PerfPhase_Compile)
		defer c.perfRecorder.Phase(perf.PerfPhase_Unaccounted)
	}
	if c.prepared {
		return nil
	}

	sourceFiles := c.tsProgram.SourceFiles()
	if len(sourceFiles) == 0 {
		c.prepared = true
		return nil
	}

	ch := make(chan *ast.SourceFile, len(sourceFiles))
	for _, sf := range sourceFiles {
		ch <- sf
	}
	close(ch)

	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}
	if numWorkers > len(sourceFiles) {
		numWorkers = len(sourceFiles)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
			for sf := range ch {
				isAffected := false
				if c.affectedFiles != nil && c.affectedFiles[canonicalizePath(sf.FileName())] {
					isAffected = true
				}
				if !isAffected && c.incrementalCompilation != nil && c.incrementalCompilation.SafeToSkipEmit(sf) {
					c.incrementalCompilation.RecordSuccessfulEmit(sf)
					continue
				}
				c.traitCompiler.UpdateSourceFile(sf, factory)
				if c.incrementalCompilation != nil {
					c.incrementalCompilation.RecordSuccessfulEmit(sf)
				}
			}
		}()
	}
	wg.Wait()

	c.prepared = true
	return nil
}

func (c *NgCompiler) AffectedFiles() map[string]bool {
	return c.affectedFiles
}

func (c *NgCompiler) GetHmrUpdate(componentId string) string {
	if c.traitCompiler != nil {
		return c.traitCompiler.GetHmrUpdate(componentId)
	}
	return ""
}

func (c *NgCompiler) GetHmrComponentIds() []string {
	if c.traitCompiler != nil {
		return c.traitCompiler.GetHmrComponentIds()
	}
	return nil
}

func getFileVersions(tsProgram *compiler.Program) map[string]string {
	versions := make(map[string]string)
	for _, sf := range tsProgram.SourceFiles() {
		hash := sha256.Sum256([]byte(sf.Text()))
		versions[sf.FileName()] = fmt.Sprintf("%x", hash)
	}
	return versions
}

type directiveMetaAdapter struct {
	meta *metadata.DirectiveMeta
	matchSource render3.MatchSource
}

func (d directiveMetaAdapter) GetName() string { return d.meta.Name }
func (d directiveMetaAdapter) GetRefKey() string {
	if d.meta.Ref.OwningModule != "" {
		return d.meta.Ref.OwningModule + "#" + d.meta.Ref.Name
	}
	return d.meta.Ref.Name
}
func (d directiveMetaAdapter) GetSelector() *string {
	if d.meta.Selector == "" {
		return nil
	}
	return &d.meta.Selector
}
func (d directiveMetaAdapter) IsComponent() bool { return d.meta.IsComponent }
func (d directiveMetaAdapter) GetInputs() any {
	obj := make(map[string]interface{}, len(d.meta.Inputs))
	for className, bindingName := range d.meta.Inputs {
		obj[className] = bindingName
	}
	return render3.ClassPropertyMappingFromMappedObject(obj)
}
func (d directiveMetaAdapter) GetOutputs() any {
	obj := make(map[string]interface{}, len(d.meta.Outputs))
	for className, bindingName := range d.meta.Outputs {
		obj[className] = bindingName
	}
	return render3.ClassPropertyMappingFromMappedObject(obj)
}
func (d directiveMetaAdapter) GetExportAs() []string { return d.meta.ExportAs }
func (d directiveMetaAdapter) IsStructural() bool    { return strings.HasPrefix(d.meta.Selector, "[") }
func (d directiveMetaAdapter) GetNgContentSelectors() []string {
	return nil
}
func (d directiveMetaAdapter) GetPreserveWhitespaces() bool { return false }
func (d directiveMetaAdapter) GetAnimationTriggerNames() *render3.LegacyAnimationTriggerNames {
	return nil
}
func (d directiveMetaAdapter) GetMatchSource() render3.MatchSource {
	return d.matchSource
}

func canonicalizePath(path string) string {
	if path == "" {
		return ""
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return tspath.GetCanonicalFileName(path, false)
	}
	return tspath.GetCanonicalFileName(path, true)
}

func (c *NgCompiler) isFileAffected(fileName string) bool {
	if c.incrementalCompilation == nil {
		return true
	}
	if c.affectedFiles == nil {
		return true
	}
	return c.affectedFiles[canonicalizePath(fileName)]
}

type overlayCompilerHost struct {
	compiler.CompilerHost
	virtualFiles map[tspath.Path]string
}

func (h *overlayCompilerHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if content, ok := h.virtualFiles[opts.Path]; ok {
		return parser.ParseSourceFile(opts, content, core.GetScriptKindFromFileName(opts.FileName))
	}
	return h.CompilerHost.GetSourceFile(opts)
}

type fileTcbData struct {
	tcbCodes []string
	blocks   []tcbBlockData
	sf       *ast.SourceFile
}

type tcbBlockData struct {
	className       string
	tcbCode         string
	lineSpans       map[int]parse_util.ParseSourceSpan
	analysis        *annotations.ComponentAnalysis
	classDecl       *ast.ClassDeclaration
	startCharOffset int
}

func (c *NgCompiler) runTemplateTypeChecking() map[string][]*ast.Diagnostic {
	res := make(map[string][]*ast.Diagnostic)
	filesData := make(map[string]*fileTcbData)

	classesMap := c.traitCompiler.GetClasses()
	for classDecl, traits := range classesMap {
		var compTrait *transform.Trait
		for _, trait := range traits {
			if trait.Handler.Name() == "ComponentDecoratorHandler" {
				compTrait = trait
				break
			}
		}
		if compTrait == nil {
			continue
		}

		sf := ast.GetSourceFileOfNode(classDecl.AsNode())
		if sf == nil {
			continue
		}
		if !c.isFileAffected(sf.FileName()) {
			continue
		}

		analysis, ok := compTrait.Analysis.(*annotations.ComponentAnalysis)
		if !ok || analysis == nil || analysis.ParsedTemplate == nil {
			continue
		}

		className := ""
		if classDecl.Name() != nil && classDecl.Name().AsIdentifier() != nil {
			className = classDecl.Name().AsIdentifier().Text
		}
		if className == "" {
			continue
		}

		scope := c.scopeRegistry.GetCompilationScope(classDecl.AsNode())

		pipes := make(map[string]string)
		var getDirectives func(node render3.Node) []typecheck.DirectiveInfo
		var getBindingConsumer func(node render3.Node, binding any) (string, string, bool)

		if scope != nil {
			for _, pipe := range scope.Pipes {
				pipes[pipe.Name] = pipe.Ref.Name
			}

			matcher := render3.NewSelectorMatcher[[]directiveMetaAdapter]()
			for i := range scope.Directives {
				dir := &scope.Directives[i]
				if dir.Selector == "" {
					continue
				}
				selectorParsed := render3.CssSelectorParse(dir.Selector)
				matcher.AddSelectables(selectorParsed, []directiveMetaAdapter{{meta: dir, matchSource: render3.MatchSourceSelector}})

				for _, hd := range dir.HostDirectives {
					hdMeta := c.metaRegistry.GetDirectiveMetadata(hd.Directive.Node)
					if hdMeta != nil {
						// Create a copy of the host directive meta with the mapped inputs/outputs
						clonedMeta := *hdMeta
						clonedMeta.Inputs = hd.Inputs
						clonedMeta.Outputs = hd.Outputs
						matcher.AddSelectables(selectorParsed, []directiveMetaAdapter{{meta: &clonedMeta, matchSource: render3.MatchSourceHostDirective}})
					}
				}
			}
			binderObj := render3.NewR3TargetBinder[directiveMetaAdapter](matcher, nil)
			bound := binderObj.Bind(render3.Target[directiveMetaAdapter]{Template: analysis.ParsedTemplate.Nodes})

			getDirectives = func(node render3.Node) []typecheck.DirectiveInfo {
				var result []typecheck.DirectiveInfo
				if dirOwner, ok := node.(render3.DirectiveOwner); ok {
					matched := bound.GetDirectivesOfNode(dirOwner)
					for _, m := range matched {
						result = append(result, typecheck.DirectiveInfo{
							ClassName:    m.meta.Ref.Name,
							OwningModule: m.meta.Ref.OwningModule,
						})
					}
				}
				return result
			}

			getBindingConsumer = func(node render3.Node, binding any) (string, string, bool) {
				if consumer := bound.GetConsumerOfBinding(binding); consumer != nil {
					if adapter, ok := consumer.(directiveMetaAdapter); ok {
						classPropName := ""
						var propName string
						switch b := binding.(type) {
						case *render3.BoundAttribute:
							propName = b.Name
						case *render3.TextAttribute:
							propName = b.Name
						case *render3.BoundEvent:
							propName = b.Name
						}
						if propName != "" {
							if inputs, ok := adapter.GetInputs().(*render3.ClassPropertyMappingGeneric); ok {
								if mapping := inputs.GetByBindingPropertyName(propName); len(mapping) > 0 {
									classPropName = mapping[0].ClassPropertyName
								}
							}
							if classPropName == "" {
								if outputs, ok := adapter.GetOutputs().(*render3.ClassPropertyMappingGeneric); ok {
									if mapping := outputs.GetByBindingPropertyName(propName); len(mapping) > 0 {
										classPropName = mapping[0].ClassPropertyName
									}
								}
							}
						}
						if classPropName == "" {
							classPropName = propName
						}
						return adapter.meta.Ref.Name, classPropName, true
					}
				}
				return "", "", false
			}
		} else {
			getDirectives = func(node render3.Node) []typecheck.DirectiveInfo {
				return nil
			}
			getBindingConsumer = func(node render3.Node, binding any) (string, string, bool) {
				return "", "", false
			}
		}

		if os.Getenv("DISABLE_GO_NGC_TYPECHECK") == "true" {
			continue
		}

		tcbCode, lineSpans := typecheck.GenerateTcb(className, analysis.ParsedTemplate, getDirectives, getBindingConsumer, pipes)
		if tcbCode == "" {
			continue
		}

		fileName := sf.FileName()
		data, ok := filesData[fileName]
		if !ok {
			data = &fileTcbData{sf: sf}
			filesData[fileName] = data
		}

		startOffset := len(sf.Text()) + 1
		for _, prevCode := range data.tcbCodes {
			startOffset += len(prevCode) + 1
		}

		data.tcbCodes = append(data.tcbCodes, tcbCode)
		data.blocks = append(data.blocks, tcbBlockData{
			className:       className,
			tcbCode:         tcbCode,
			lineSpans:       lineSpans,
			analysis:        analysis,
			classDecl:       classDecl,
			startCharOffset: startOffset,
		})
	}

	if len(filesData) == 0 {
		return res
	}

	virtualFiles := make(map[tspath.Path]string)
	for _, data := range filesData {
		content := data.sf.Text()
		for _, code := range data.tcbCodes {
			content += "\n" + code
		}
		virtualFiles[data.sf.Path()] = content
	}

	overlayHost := &overlayCompilerHost{
		CompilerHost: c.tsProgram.Host(),
		virtualFiles: virtualFiles,
	}

	typecheckProgram := c.tsProgram
	for path := range virtualFiles {
		var ok bool
		typecheckProgram, ok = typecheckProgram.UpdateProgram(path, overlayHost, nil)
		if !ok {
			continue
		}
	}

	freshChecker, _ := typecheckProgram.GetTypeChecker(context.TODO())

	for fileName, data := range filesData {
		sf := typecheckProgram.GetSourceFileByPath(data.sf.Path())
		if sf == nil {
			continue
		}

		checkerDiags := freshChecker.GetDiagnostics(context.TODO(), sf)

		for _, d := range checkerDiags {
			for _, block := range data.blocks {
				if d.Pos() >= block.startCharOffset && d.Pos() < block.startCharOffset+len(block.tcbCode) {
					tcbOffset := d.Pos() - block.startCharOffset
					tcbLines := strings.Split(block.tcbCode, "\n")
					charOffset := 0
					diagnosticLine := 1
					for idx, line := range tcbLines {
						lineLen := len(line) + 1
						if tcbOffset >= charOffset && tcbOffset < charOffset+lineLen {
							diagnosticLine = idx + 1
							break
						}
						charOffset += lineLen
					}

					if span, ok := block.lineSpans[diagnosticLine]; ok && span.Start != nil {
						startOffset := span.Start.Offset
						endOffset := span.End.Offset

						var targetFile *ast.SourceFile
						var targetLoc core.TextRange

						if block.analysis.TemplateUrl == "" {
							templateStart := strings.Index(data.sf.Text(), block.analysis.Template)
							if templateStart != -1 {
								targetFile = data.sf
								targetLoc = core.NewTextRange(templateStart+startOffset, templateStart+endOffset)
							} else {
								targetFile = data.sf
								targetLoc = core.NewTextRange(block.classDecl.Pos(), block.classDecl.End())
							}
						} else {
							templatePath := span.Start.File.Url
							templateContent := span.Start.File.Content
							templateSf := parser.ParseSourceFile(ast.SourceFileParseOptions{
								FileName: templatePath,
								Path:     tspath.Path(templatePath),
							}, templateContent, core.ScriptKindJS)
							if templateSf != nil {
								targetFile = templateSf
								targetLoc = core.NewTextRange(startOffset, endOffset)
							} else {
								targetFile = data.sf
								targetLoc = core.NewTextRange(block.classDecl.Pos(), block.classDecl.End())
							}
						}

						relatedNode := findTemplatePropertyNode(block.analysis.DecoratorNode)
						if relatedNode == nil {
							relatedNode = block.classDecl.AsNode()
						}
						var relatedFile *ast.SourceFile
						var relatedLoc core.TextRange
						if relatedNode != nil {
							relatedFile = ast.GetSourceFileOfNode(relatedNode)
							start := scanner.GetTokenPosOfNode(relatedNode, relatedFile, false)
							relatedLoc = core.NewTextRange(start, relatedNode.End())
						}
						relatedDiag := ast.NewDiagnosticFromSerialized(
							relatedFile,
							relatedLoc,
							0,
							3, // CategoryMessage
							"",
							[]string{"Error occurs in the template of component " + block.className + "."},
							nil,
							nil,
							false,
							false,
							false,
						)

						mappedDiag := ast.NewDiagnosticFromSerialized(
							targetFile,
							targetLoc,
							d.Code(),
							d.Category(),
							d.MessageKey(),
							d.MessageArgs(),
							d.MessageChain(),
							[]*ast.Diagnostic{relatedDiag},
							d.ReportsUnnecessary(),
							d.ReportsDeprecated(),
							d.SkippedOnNoEmit(),
						)
						res[fileName] = append(res[fileName], mappedDiag)
					}
					break
				}
			}
		}
	}

	return res
}

func (c *NgCompiler) SetPerfRecorder(rec *perf.ActivePerfRecorder) {
	c.perfRecorder = rec
}

func (c *NgCompiler) PerfRecorder() *perf.ActivePerfRecorder {
	return c.perfRecorder
}

func (c *NgCompiler) GetPerfResults() map[string]int64 {
	if c.perfRecorder != nil {
		res := c.perfRecorder.Finalize()
		return res.Phases
	}
	return nil
}

func findTemplatePropertyNode(decoratorNode *ast.Node) *ast.Node {
	if decoratorNode == nil || decoratorNode.AsDecorator() == nil || decoratorNode.AsDecorator().Expression == nil {
		return nil
	}
	callExpr := decoratorNode.AsDecorator().Expression.AsCallExpression()
	if callExpr == nil || callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) == 0 {
		return nil
	}
	obj := callExpr.Arguments.Nodes[0].AsObjectLiteralExpression()
	if obj == nil || obj.Properties == nil {
		return nil
	}
	for _, prop := range obj.Properties.Nodes {
		if !ast.IsPropertyAssignment(prop) {
			continue
		}
		name := prop.Name()
		if name == nil {
			continue
		}
		if name.Text() == "templateUrl" || name.Text() == "template" {
			return prop.AsPropertyAssignment().Initializer
		}
	}
	return nil
}

func (c *NgCompiler) GetComponentsWithTemplateFile(templatePath string) []*ast.Node {
	if c.resourceRegistry != nil {
		return c.resourceRegistry.GetComponentsWithTemplate(templatePath)
	}
	return nil
}

func (c *NgCompiler) GetComponentsWithStyleFile(stylePath string) []*ast.Node {
	if c.resourceRegistry != nil {
		return c.resourceRegistry.GetComponentsWithStyle(stylePath)
	}
	return nil
}

func (c *NgCompiler) GetDirectiveResources(directive *ast.Node) *metadata.DirectiveResources {
	if c.resourceRegistry != nil {
		template := c.resourceRegistry.GetTemplate(directive)
		styles := c.resourceRegistry.GetStyles(directive)
		hostBindings := c.resourceRegistry.GetHostBindings(directive)
		if template != nil || len(styles) > 0 || len(hostBindings) > 0 {
			return &metadata.DirectiveResources{
				Template:     template,
				Styles:       styles,
				HostBindings: hostBindings,
			}
		}
	}
	return nil
}

func (c *NgCompiler) GetDiagnosticsForFile(file *ast.SourceFile, optimizeFor any) []*ast.Diagnostic {
	c.AnalyzeSync()
	diags := c.Resolve()

	var result []*ast.Diagnostic
	for _, d := range diags {
		if d.File() != nil && d.File().FileName() == file.FileName() {
			result = append(result, d)
			continue
		}
		if d.RelatedInformation() != nil {
			for _, rel := range d.RelatedInformation() {
				if rel.File() != nil && rel.File().FileName() == file.FileName() {
					result = append(result, d)
					break
				}
			}
		}
	}
	return result
}

func (c *NgCompiler) GetResourceDependencies(file *ast.SourceFile) []string {
	c.AnalyzeSync()
	if c.incrementalCompilation != nil && c.incrementalCompilation.State != nil && c.incrementalCompilation.State.DepGraph != nil {
		return c.incrementalCompilation.State.DepGraph.GetResourceDependencies(file)
	}
	return nil
}

