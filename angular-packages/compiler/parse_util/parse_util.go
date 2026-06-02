package parse_util

import (
	"fmt"
	"strings"
)

type ParseLocation struct {
	File   *ParseSourceFile
	Offset int
	Line   int
	Col    int
}

func NewParseLocation(file *ParseSourceFile, offset int, line int, col int) *ParseLocation {
	return &ParseLocation{
		File:   file,
		Offset: offset,
		Line:   line,
		Col:    col,
	}
}

func (l *ParseLocation) ToString() string {
	if l.Offset != -1 {
		return fmt.Sprintf("%s@%d:%d", l.File.Url, l.Line, l.Col)
	}
	return l.File.Url
}

func (l *ParseLocation) MoveBy(delta int) *ParseLocation {
	source := l.File.Content
	length := len(source)
	offset := l.Offset
	line := l.Line
	col := l.Col

	if delta > 0 {
		end := offset + delta
		if end > length {
			end = length
		}
		chunk := source[offset:end]
		newlines := strings.Count(chunk, "\n")
		line += newlines
		if newlines > 0 {
			lastNewline := strings.LastIndexByte(chunk, '\n')
			col = len(chunk) - lastNewline - 1
		} else {
			col += delta
		}
		offset = end
	} else if delta < 0 {
		for offset > 0 && delta < 0 {
			offset--
			delta++
			ch := source[offset]
			if ch == '\n' {
				line--
				var priorLine int = -1
				if offset > 0 {
					priorLine = strings.LastIndexByte(source[:offset], '\n')
				}
				if priorLine > 0 {
					col = offset - priorLine
				} else {
					col = offset
				}
			} else {
				col--
			}
		}
	}
	return NewParseLocation(l.File, offset, line, col)
}

type ParseContext struct {
	Before string
	After  string
}

func (l *ParseLocation) GetContext(maxChars int, maxLines int) *ParseContext {
	content := l.File.Content
	startOffset := l.Offset

	if startOffset >= 0 {
		if startOffset > len(content)-1 {
			startOffset = len(content) - 1
		}
		if startOffset < 0 {
			startOffset = 0
		}
		endOffset := startOffset
		ctxChars := 0
		ctxLines := 0

		for ctxChars < maxChars && startOffset > 0 {
			startOffset--
			ctxChars++
			if content[startOffset] == '\n' {
				ctxLines++
				if ctxLines == maxLines {
					break
				}
			}
		}

		ctxChars = 0
		ctxLines = 0
		for ctxChars < maxChars && endOffset < len(content)-1 {
			endOffset++
			ctxChars++
			if content[endOffset] == '\n' {
				ctxLines++
				if ctxLines == maxLines {
					break
				}
			}
		}

		return &ParseContext{
			Before: content[startOffset:l.Offset],
			After:  content[l.Offset : endOffset+1],
		}
	}

	return nil
}

type ParseSourceFile struct {
	Content string
	Url     string
}

func NewParseSourceFile(content string, url string) *ParseSourceFile {
	return &ParseSourceFile{
		Content: content,
		Url:     url,
	}
}

type ParseSourceSpan struct {
	Start     *ParseLocation
	End       *ParseLocation
	FullStart *ParseLocation
	Details   *string
}

func NewParseSourceSpan(start *ParseLocation, end *ParseLocation, fullStart *ParseLocation, details *string) *ParseSourceSpan {
	if fullStart == nil {
		fullStart = start
	}
	return &ParseSourceSpan{
		Start:     start,
		End:       end,
		FullStart: fullStart,
		Details:   details,
	}
}

func (s *ParseSourceSpan) ToString() string {
	return s.Start.File.Content[s.Start.Offset:s.End.Offset]
}

type ParseErrorLevel int

const (
	ParseErrorLevelWarning ParseErrorLevel = 0
	ParseErrorLevelError   ParseErrorLevel = 1
)

type ParseError struct {
	Span         *ParseSourceSpan
	Msg          string
	Level        ParseErrorLevel
	RelatedError any
}

func NewParseError(span *ParseSourceSpan, msg string, level *ParseErrorLevel, relatedError any) *ParseError {
	l := ParseErrorLevelError
	if level != nil {
		l = *level
	}
	return &ParseError{
		Span:         span,
		Msg:          msg,
		Level:        l,
		RelatedError: relatedError,
	}
}

func (e *ParseError) Error() string {
	return e.ToString()
}

func (e *ParseError) ContextualMessage() string {
	ctx := e.Span.Start.GetContext(100, 3)
	if ctx != nil {
		levelStr := "ERROR"
		if e.Level == ParseErrorLevelWarning {
			levelStr = "WARNING"
		}
		return fmt.Sprintf("%s (\"%s[%s ->]%s\")", e.Msg, ctx.Before, levelStr, ctx.After)
	}
	return e.Msg
}

func (e *ParseError) ToString() string {
	details := ""
	if e.Span.Details != nil {
		details = fmt.Sprintf(", %s", *e.Span.Details)
	}
	return fmt.Sprintf("%s: %s%s", e.ContextualMessage(), e.Span.Start.ToString(), details)
}

func R3JitTypeSourceSpan(kind string, typeName string, sourceUrl string) *ParseSourceSpan {
	sourceFileName := fmt.Sprintf("in %s %s in %s", kind, typeName, sourceUrl)
	sourceFile := NewParseSourceFile("", sourceFileName)
	return NewParseSourceSpan(
		NewParseLocation(sourceFile, -1, -1, -1),
		NewParseLocation(sourceFile, -1, -1, -1),
		nil,
		nil,
	)
}

func (s *ParseSourceSpan) GetStart() *ParseLocation {
	return s.Start
}

func (l *ParseLocation) GetFile() *ParseSourceFile {
	return l.File
}

func (l *ParseLocation) GetLine() int {
	return l.Line
}

func (l *ParseLocation) GetCol() int {
	return l.Col
}

func (f *ParseSourceFile) GetUrl() string {
	return f.Url
}

func (f *ParseSourceFile) GetContent() string {
	return f.Content
}
