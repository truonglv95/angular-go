package typecheck

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/program_driver"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	internal_diagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
)

type TypeCheckingTarget struct {
	FileName          string
	Source            string
	Templates         map[string]string
	Declarations      []TestDeclaration
	ForeignComponents []string
}

type TestDeclaration struct {
	Type                string // "directive" or "pipe"
	File                string
	Selector            string
	Name                string
	Inputs              any // map[string]any or map[string]string
	Outputs             map[string]string
	IsStandalone        bool
	PipeName            string
	IsGeneric           bool
	Code                string
	HasNgFieldDirective bool
}

type TemplateTypeCheckerImpl struct {
	program         *compiler.Program
	programDriver   *program_driver.TsCreateProgramDriver
	targets         []TypeCheckingTarget
	inlining        bool
	inliningMode    program_driver.InliningMode
	absoluteOffset  int
	lastFileChecked string
}

func NewTemplateTypeCheckerImpl(
	program *compiler.Program,
	programDriver *program_driver.TsCreateProgramDriver,
	targets []TypeCheckingTarget,
	inlining bool,
	inliningMode program_driver.InliningMode,
) *TemplateTypeCheckerImpl {
	return &TemplateTypeCheckerImpl{
		program:         program,
		programDriver:   programDriver,
		targets:         targets,
		inlining:        inlining,
		inliningMode:    inliningMode,
		absoluteOffset:  0,
		lastFileChecked: "",
	}
}

func (c *TemplateTypeCheckerImpl) GetTemplate(classDecl *ast.Node) []render3.Node {
	if classDecl == nil || classDecl.Name() == nil {
		return nil
	}
	className := classDecl.Name().AsIdentifier().Text
	for _, target := range c.targets {
		if target.Templates != nil {
			if templateText, ok := target.Templates[className]; ok {
				parsed := render3.ParseTemplate(templateText, className+".html", nil)
				return parsed.Nodes
			}
		}
	}
	return nil
}

func (c *TemplateTypeCheckerImpl) getShimText(fileName string, sf *ast.SourceFile, tcbStr string) string {
	if c.inliningMode == program_driver.InliningMode_CopySourceToTcb {
		return sf.Text() + "\n\n" + tcbStr
	}

	var target *TypeCheckingTarget
	for i := range c.targets {
		if c.targets[i].FileName == fileName {
			target = &c.targets[i]
			break
		}
	}

	importStr := ""
	if target != nil {
		baseName := fileName
		if idx := strings.LastIndex(fileName, "/"); idx != -1 {
			baseName = fileName[idx+1:]
		}
		if strings.HasSuffix(baseName, ".ts") {
			baseName = baseName[:len(baseName)-3]
		}
		for className := range target.Templates {
			importStr += fmt.Sprintf("import { %s } from './%s';\n", className, baseName)
		}
	}

	return importStr + "\n" + tcbStr
}

func (c *TemplateTypeCheckerImpl) GetTypeCheckBlock(classDecl *ast.Node) *ast.Node {
	if classDecl == nil || classDecl.Name() == nil {
		return nil
	}
	className := classDecl.Name().AsIdentifier().Text
	sf := ast.GetSourceFileOfNode(classDecl)
	fileName := sf.FileName()

	// Invalidate shim of lastFileChecked if it changed
	if c.lastFileChecked != "" && c.lastFileChecked != fileName {
		lastShimPath := c.lastFileChecked + ".ngtypecheck.ts"
		if strings.HasSuffix(c.lastFileChecked, ".ts") {
			lastShimPath = c.lastFileChecked[:len(c.lastFileChecked)-3] + ".ngtypecheck.ts"
		}
		updates := map[file_system.AbsoluteFsPath]program_driver.FileUpdate{
			file_system.AbsoluteFsPath(lastShimPath): {
				NewText: "export const MODULE = true;",
			},
		}
		c.programDriver.UpdateFiles(updates, program_driver.UpdateMode_Complete)
		c.program = c.programDriver.GetProgram().(*compiler.Program)
	}
	c.lastFileChecked = fileName

	tcbStr := c.generateTcbForComponent(fileName, className)
	if tcbStr == "" {
		return nil
	}

	shimPath := fileName + ".ngtypecheck.ts"
	if strings.HasSuffix(fileName, ".ts") {
		shimPath = fileName[:len(fileName)-3] + ".ngtypecheck.ts"
	}

	shimText := c.getShimText(fileName, sf, tcbStr)

	currentShimSf := c.program.GetSourceFile(shimPath)
	if currentShimSf == nil || currentShimSf.Text() != shimText {
		updates := map[file_system.AbsoluteFsPath]program_driver.FileUpdate{
			file_system.AbsoluteFsPath(shimPath): {
				NewText: shimText,
			},
		}
		c.programDriver.UpdateFiles(updates, program_driver.UpdateMode_Complete)
		c.program = c.programDriver.GetProgram().(*compiler.Program)
	}

	shimSf := c.program.GetSourceFile(shimPath)
	if shimSf == nil {
		return nil
	}

	var tcbNode *ast.Node
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		if ast.IsFunctionDeclaration(node) {
			name := node.Name()
			if name != nil && ast.IsIdentifier(name) && name.AsIdentifier().Text == "_tcb_"+className {
				tcbNode = node
				return true
			}
		}
		return node.ForEachChild(walk)
	}
	shimSf.AsNode().ForEachChild(walk)
	return tcbNode
}

func (c *TemplateTypeCheckerImpl) GetDiagnosticsForFile(sf *ast.SourceFile, mode OptimizeFor) []*ast.Diagnostic {
	if mode == OptimizeFor_WholeProgram {
		c.lastFileChecked = ""
	}
	fileName := sf.FileName()

	var target *TypeCheckingTarget
	for i := range c.targets {
		if c.targets[i].FileName == fileName {
			target = &c.targets[i]
			break
		}
	}

	if target == nil || len(target.Templates) == 0 {
		return nil
	}

	if !c.inlining && c.inliningMode == program_driver.InliningMode_Error {
		for className := range target.Templates {
			decl := c.getClassDecl(sf, className)
			isExported := false
			if decl != nil && decl.Modifiers() != nil {
				if (decl.Modifiers().ModifierFlags & ast.ModifierFlagsExport) != 0 {
					isExported = true
				}
			}
			if !isExported {
				code := diagnostics.ErrorCode_INLINE_TCB_REQUIRED
				diag := diagnostics.MakeDiagnostic(
					code,
					decl,
					"Template type-checking block inline required",
					nil,
					internal_diagnostics.CategoryError,
				)
				return []*ast.Diagnostic{diag}
			}
		}
	}

	updates := make(map[file_system.AbsoluteFsPath]program_driver.FileUpdate)
	if mode == OptimizeFor_WholeProgram {
		for _, tgt := range c.targets {
			sfTgt := c.program.GetSourceFile(tgt.FileName)
			if sfTgt == nil {
				continue
			}
			var tcbParts []string
			for className := range tgt.Templates {
				tcbParts = append(tcbParts, c.generateTcbForComponent(tgt.FileName, className))
			}
			tcbStr := strings.Join(tcbParts, "\n\n")
			if tcbStr != "" {
				shimPath := tgt.FileName + ".ngtypecheck.ts"
				if strings.HasSuffix(tgt.FileName, ".ts") {
					shimPath = tgt.FileName[:len(tgt.FileName)-3] + ".ngtypecheck.ts"
				}
				shimText := c.getShimText(tgt.FileName, sfTgt, tcbStr)

				currentShimSf := c.program.GetSourceFile(shimPath)
				if currentShimSf == nil || currentShimSf.Text() != shimText {
					updates[file_system.AbsoluteFsPath(shimPath)] = program_driver.FileUpdate{
						NewText: shimText,
					}
				}
			}
		}
	} else {
		var tcbParts []string
		for className := range target.Templates {
			tcbParts = append(tcbParts, c.generateTcbForComponent(target.FileName, className))
		}
		tcbStr := strings.Join(tcbParts, "\n\n")
		if tcbStr != "" {
			shimPath := target.FileName + ".ngtypecheck.ts"
			if strings.HasSuffix(target.FileName, ".ts") {
				shimPath = target.FileName[:len(target.FileName)-3] + ".ngtypecheck.ts"
			}
			shimText := c.getShimText(target.FileName, sf, tcbStr)

			currentShimSf := c.program.GetSourceFile(shimPath)
			if currentShimSf == nil || currentShimSf.Text() != shimText {
				updates[file_system.AbsoluteFsPath(shimPath)] = program_driver.FileUpdate{
					NewText: shimText,
				}
			}
		}
	}

	if len(updates) > 0 {
		c.programDriver.UpdateFiles(updates, program_driver.UpdateMode_Complete)
		c.program = c.programDriver.GetProgram().(*compiler.Program)
	}

	for className := range target.Templates {
		_ = c.GetTypeCheckBlock(c.getClassDecl(sf, className))
	}

	chk, release := c.program.GetTypeChecker(context.Background())
	defer release()

	shimPath := fileName + ".ngtypecheck.ts"
	if strings.HasSuffix(fileName, ".ts") {
		shimPath = fileName[:len(fileName)-3] + ".ngtypecheck.ts"
	}
	shimSf := c.program.GetSourceFile(shimPath)
	if shimSf == nil {
		return nil
	}

	allDiags := chk.GetDiagnostics(context.Background(), shimSf)
	var filteredDiags []*ast.Diagnostic

	for _, diag := range allDiags {
		if c.inliningMode == program_driver.InliningMode_CopySourceToTcb {
			pos := diag.Pos()
			tcbIndex := strings.Index(shimSf.Text(), "function _tcb_")
			if tcbIndex != -1 && pos < tcbIndex {
				continue
			}
		}
		filteredDiags = append(filteredDiags, diag)
	}

	return filteredDiags
}

func (c *TemplateTypeCheckerImpl) getClassDecl(sf *ast.SourceFile, className string) *ast.Node {
	var decl *ast.Node
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		if ast.IsClassDeclaration(node) {
			name := node.Name()
			if name != nil && ast.IsIdentifier(name) && name.AsIdentifier().Text == className {
				decl = node
				return true
			}
		}
		return node.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)
	return decl
}

func matchesSelector(element *render3.Element, selector string) bool {
	if selector == "" {
		return false
	}
	if strings.HasPrefix(selector, "[") && strings.HasSuffix(selector, "]") {
		attrName := selector[1 : len(selector)-1]
		for _, attr := range element.Attributes {
			if attr.Name == attrName {
				return true
			}
		}
		for _, input := range element.Inputs {
			if input.Name == attrName {
				return true
			}
		}
		return false
	}
	return element.Name == selector
}

func (c *TemplateTypeCheckerImpl) generateTcbForComponent(fileName string, className string) string {
	var target *TypeCheckingTarget
	for i := range c.targets {
		if c.targets[i].FileName == fileName {
			target = &c.targets[i]
			break
		}
	}
	if target == nil {
		return ""
	}
	templateText, ok := target.Templates[className]
	if !ok {
		return ""
	}

	parsed := render3.ParseTemplate(templateText, className+".html", nil)

	pipes := make(map[string]string)
	for _, decl := range target.Declarations {
		if decl.Type == "pipe" {
			pipes[decl.PipeName] = decl.Name
		}
	}

	getDirectives := func(node render3.Node) []DirectiveInfo {
		var result []DirectiveInfo
		el, ok := node.(*render3.Element)
		if !ok {
			return nil
		}
		for _, decl := range target.Declarations {
			if decl.Type == "directive" {
				if matchesSelector(el, decl.Selector) {
					result = append(result, DirectiveInfo{
						ClassName:    decl.Name,
						OwningModule: decl.File,
					})
				}
			}
		}
		return result
	}

	getBindingConsumer := func(node render3.Node, binding any) (string, string, bool) {
		el, ok := node.(*render3.Element)
		if !ok {
			return "", "", false
		}
		boundAttr, isBoundAttr := binding.(*render3.BoundAttribute)
		if !isBoundAttr {
			return "", "", false
		}

		for _, decl := range target.Declarations {
			if decl.Type == "directive" && matchesSelector(el, decl.Selector) {
				if decl.Inputs != nil {
					switch inputs := decl.Inputs.(type) {
					case map[string]string:
						if propName, ok := inputs[boundAttr.Name]; ok {
							return decl.Name, propName, true
						}
					case map[string]any:
						if val, ok := inputs[boundAttr.Name]; ok {
							if strVal, ok := val.(string); ok {
								return decl.Name, strVal, true
							}
						}
					}
				}
			}
		}
		return "", "", false
	}

	tcbStr, _ := GenerateTcbWithOptions(
		className,
		&parsed,
		getDirectives,
		getBindingConsumer,
		pipes,
		true,
		c.absoluteOffset,
	)
	return tcbStr
}

func Setup(t *testing.T, targets []TypeCheckingTarget, overrides map[string]any) (*TemplateTypeCheckerImpl, *compiler.Program, *program_driver.TsCreateProgramDriver) {
	var files []ngtsctest.ProgramFile
	for _, target := range targets {
		contents := target.Source
		if contents == "" {
			contents = "// generated from templates\n\nexport const MODULE = true;\n\n"
			if target.Templates != nil {
				for className := range target.Templates {
					contents += fmt.Sprintf("export class %s {}\n", className)
				}
			}
		}
		files = append(files, ngtsctest.ProgramFile{
			Name:     target.FileName,
			Contents: contents,
		})

		shimName := target.FileName + ".ngtypecheck.ts"
		if strings.HasSuffix(target.FileName, ".ts") {
			shimName = target.FileName[:len(target.FileName)-3] + ".ngtypecheck.ts"
		}
		files = append(files, ngtsctest.ProgramFile{
			Name:     shimName,
			Contents: "export const MODULE = true;",
		})
	}

	progResult := ngtsctest.MakeProgram(t, files)
	prog := progResult.Program

	inlining := true
	if val, ok := overrides["inlining"]; ok {
		if b, ok := val.(bool); ok {
			inlining = b
		}
	}
	inliningMode := program_driver.InliningMode_InlineOps
	if !inlining {
		inliningMode = program_driver.InliningMode_Error
	}
	if val, ok := overrides["inliningMode"]; ok {
		if m, ok := val.(program_driver.InliningMode); ok {
			inliningMode = m
		}
	}

	programDriver := program_driver.NewTsCreateProgramDriver(prog, prog.Host(), nil, []string{"ngtypecheck"})
	programDriver.SetInliningMode(inliningMode)

	templateTypeChecker := NewTemplateTypeCheckerImpl(prog, programDriver, targets, inlining, inliningMode)

	return templateTypeChecker, prog, programDriver
}
