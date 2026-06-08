package compiler_cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/locale"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func getDiagnosticMessageText(d *ast.Diagnostic, loc locale.Locale) string {
	var text string
	func() {
		defer func() {
			if r := recover(); r != nil {
				if len(d.MessageArgs()) > 0 {
					text = d.MessageArgs()[0]
				} else {
					text = d.String()
				}
			}
		}()
		text = d.Localize(loc)
	}()
	return text
}

func FormatDiagnosticEsbuildStyle(output io.Writer, d *ast.Diagnostic, loc locale.Locale, compareOpts tspath.ComparePathsOptions) {
	categoryStr := ""
	categoryColor := ""
	icon := ""
	switch d.Category() {
	case diagnostics.CategoryWarning:
		categoryStr = "[WARNING]"
		categoryColor = "\u001b[33m"
		icon = "⚠"
	case diagnostics.CategoryError:
		categoryStr = "[ERROR]"
		categoryColor = "\u001b[31m"
		icon = "✘"
	default:
		categoryStr = "[INFO]"
		categoryColor = "\u001b[34m"
		icon = "ℹ"
	}
	fmt.Fprintf(output, "%s\u001b[1m%s %s \u001b[22m\u001b[39m", categoryColor, icon, categoryStr)
	fmt.Fprintf(output, "\u001b[1mTS%d: \u001b[22m", d.Code())

	// Flattened message
	msgText := getDiagnosticMessageText(d, loc)
	fmt.Fprint(output, msgText)
	// Flatten chain if any
	for _, chain := range d.MessageChain() {
		flattenDiagnosticMessageChainEsbuild(output, chain, loc, 1)
	}
	fmt.Fprintf(output, " \u001b[90m[plugin angular-compiler]\u001b[0m\n")

	if d.File() != nil && d.Code() != int32(diagnostics.File_appears_to_be_binary.Code()) {
		file := d.File()
		pos := d.Pos()
		firstLine, firstChar := scanner.GetECMALineAndUTF16CharacterOfPosition(file, pos)
		relativeFileName := tspath.ConvertToRelativePath(file.FileName(), compareOpts)

		fmt.Fprint(output, "\n")
		fmt.Fprintf(output, "    %s:%d:%d:\n", relativeFileName, firstLine+1, firstChar)
		writeCodeSnippetEsbuild(output, file, pos, d.Len(), d.Category(), "\n")
		fmt.Fprint(output, "\n")
	}

	if d.RelatedInformation() != nil && len(d.RelatedInformation()) > 0 {
		for _, related := range d.RelatedInformation() {
			file := related.File()
			fmt.Fprint(output, "\n")
			fmt.Fprint(output, "  ")

			// Flattened related message
			fmt.Fprint(output, getDiagnosticMessageText(related, loc))
			for _, chain := range related.MessageChain() {
				flattenDiagnosticMessageChainEsbuild(output, chain, loc, 1)
			}
			fmt.Fprint(output, "\n")

			if file != nil {
				fmt.Fprint(output, "\n")
				pos := related.Pos()
				firstLine, firstChar := scanner.GetECMALineAndUTF16CharacterOfPosition(file, pos)
				relativeFileName := tspath.ConvertToRelativePath(file.FileName(), compareOpts)
				fmt.Fprintf(output, "    %s:%d:%d:\n", relativeFileName, firstLine+1, firstChar)
				writeCodeSnippetEsbuild(output, file, pos, related.Len(), related.Category(), "\n")
			}
			fmt.Fprint(output, "\n")
		}
	}
}

func flattenDiagnosticMessageChainEsbuild(writer io.Writer, chain *ast.Diagnostic, loc locale.Locale, level int) {
	fmt.Fprint(writer, "\n")
	for range level {
		fmt.Fprint(writer, "  ")
	}
	fmt.Fprint(writer, getDiagnosticMessageText(chain, loc))
	for _, child := range chain.MessageChain() {
		flattenDiagnosticMessageChainEsbuild(writer, child, loc, level+1)
	}
}

// FileLike interface mapping
type fileLikeWrapper struct {
	file *ast.SourceFile
}

func (w fileLikeWrapper) FileName() string { return w.file.FileName() }
func (w fileLikeWrapper) Text() string     { return w.file.Text() }
func (w fileLikeWrapper) ECMALineMap() []core.TextPos {
	return w.file.ECMALineMap()
}

func writeCodeSnippetEsbuild(writer io.Writer, file *ast.SourceFile, start int, length int, category diagnostics.Category, newLine string) {
	fw := fileLikeWrapper{file}
	firstLine, firstLineChar := scanner.GetECMALineAndUTF16CharacterOfPosition(fw, start)
	lastLine, lastLineChar := scanner.GetECMALineAndUTF16CharacterOfPosition(fw, start+length)
	if length == 0 {
		lastLineChar++
	}

	lastLineOfFile := scanner.GetECMALineOfPosition(fw, len(fw.Text()))

	hasMoreThanFiveLines := lastLine-firstLine >= 4
	gutterWidth := len(strconv.Itoa(lastLine + 1))
	if hasMoreThanFiveLines {
		gutterWidth = max(3, gutterWidth) // 3 is length of "..."
	}

	for i := firstLine; i <= lastLine; i++ {
		if i > firstLine {
			fmt.Fprint(writer, newLine)
		}

		if hasMoreThanFiveLines && firstLine+1 < i && i < lastLine-1 {
			fmt.Fprint(writer, "      ")
			fmt.Fprint(writer, "\u001b[90m")
			fmt.Fprintf(writer, "%*s", gutterWidth, "...")
			fmt.Fprint(writer, "\u001b[0m")
			fmt.Fprint(writer, " │ ")
			fmt.Fprint(writer, newLine)
			i = lastLine - 1
		}

		lineStart := scanner.GetECMAPositionOfLineAndByteOffset(fw, i, 0)
		var lineEnd int
		if i < lastLineOfFile {
			lineEnd = scanner.GetECMAPositionOfLineAndByteOffset(fw, i+1, 0)
		} else {
			lineEnd = len(fw.Text())
		}

		lineContent := strings.TrimRightFunc(fw.Text()[lineStart:lineEnd], unicode.IsSpace)
		lineContent = strings.ReplaceAll(lineContent, "\t", " ")

		// Output the gutter and the actual contents of the line.
		fmt.Fprint(writer, "      ")
		fmt.Fprint(writer, "\u001b[90m")
		fmt.Fprintf(writer, "%*d", gutterWidth, i+1)
		fmt.Fprint(writer, "\u001b[0m")
		fmt.Fprint(writer, " │ ")
		fmt.Fprint(writer, lineContent)
		fmt.Fprint(writer, newLine)

		// Output the gutter and the error span for the line using tildes.
		fmt.Fprint(writer, "      ")
		fmt.Fprint(writer, "\u001b[90m")
		fmt.Fprintf(writer, "%*s", gutterWidth, "")
		fmt.Fprint(writer, "\u001b[0m")
		fmt.Fprint(writer, " ╵ ")

		squiggleColor := "\u001b[31m"
		if category == diagnostics.CategoryWarning {
			squiggleColor = "\u001b[33m"
		} else if category != diagnostics.CategoryError {
			squiggleColor = "\u001b[34m"
		}
		fmt.Fprint(writer, squiggleColor)

		switch i {
		case firstLine:
			var lastCharForLine int
			if i == lastLine {
				lastCharForLine = int(lastLineChar)
			} else {
				lastCharForLine = int(core.UTF16Len(lineContent))
			}

			fmt.Fprint(writer, strings.Repeat(" ", int(firstLineChar)))
			fmt.Fprint(writer, strings.Repeat("~", lastCharForLine-int(firstLineChar)))
		case lastLine:
			fmt.Fprint(writer, strings.Repeat("~", int(lastLineChar)))
		default:
			fmt.Fprint(writer, strings.Repeat("~", int(core.UTF16Len(lineContent))))
		}

		fmt.Fprint(writer, "\u001b[0m")
	}
}
