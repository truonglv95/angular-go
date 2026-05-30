package output

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type sourceSpanNode struct {
	span ParseSourceSpan
}

func (n sourceSpanNode) GetSourceSpan() interface{} {
	return n.span
}

func TestEmitterVisitorContext(t *testing.T) {
	fileA := parse_util.NewParseSourceFile("a0a1a2a3a4a5a6a7a8a9", "a.js")
	fileB := parse_util.NewParseSourceFile("b0b1b2b3b4b5b6b7b8b9", "b.js")

	newCtx := func() *EmitterVisitorContext {
		return EmitterVisitorContextCreateRoot()
	}

	t.Run("should add source files to the source map", func(t *testing.T) {
		ctx := newCtx()
		ctx.Print(sourceNode(fileA, 0), "o0", false)
		ctx.Print(sourceNode(fileA, 1), "o1", false)
		ctx.Print(sourceNode(fileB, 0), "o2", false)
		ctx.Print(sourceNode(fileB, 1), "o3", false)

		sm := ctx.ToSourceMapGenerator("o.ts", 0).ToJSON()
		if sm == nil {
			t.Fatal("expected source map")
		}
		if got := strings.Join(sm.Sources, ","); got != "a.js,b.js" {
			t.Fatalf("sources = %q", got)
		}
		if len(sm.SourcesContent) != 2 || sm.SourcesContent[0] == nil || sm.SourcesContent[1] == nil {
			t.Fatalf("sourcesContent missing")
		}
		if *sm.SourcesContent[0] != fileA.Content || *sm.SourcesContent[1] != fileB.Content {
			t.Fatalf("sourcesContent mismatch")
		}
	})

	t.Run("should use the default source file for the first character", func(t *testing.T) {
		ctx := newCtx()
		ctx.Print(nil, "fileA-0", false)

		sm := ctx.ToSourceMapGenerator("o.ts", 0).ToJSON()
		if sm == nil {
			t.Fatal("expected source map")
		}
		if got := strings.Join(sm.Sources, ","); got != "o.ts" {
			t.Fatalf("sources = %q", got)
		}
		if got := segmentsPerLine(ctx); len(got) != 1 || got[0] != 1 {
			t.Fatalf("segmentsPerLine = %#v", got)
		}
	})

	t.Run("should use an explicit mapping for the first character", func(t *testing.T) {
		ctx := newCtx()
		ctx.Print(sourceNode(fileA, 0), "fileA-0", false)

		sm := ctx.ToSourceMapGenerator("o.ts", 0).ToJSON()
		if sm == nil {
			t.Fatal("expected source map")
		}
		if got := strings.Join(sm.Sources, ","); got != "a.js" {
			t.Fatalf("sources = %q", got)
		}
		if got := segmentsPerLine(ctx); len(got) != 1 || got[0] != 1 {
			t.Fatalf("segmentsPerLine = %#v", got)
		}
	})

	t.Run("should map leading segment without span", func(t *testing.T) {
		ctx := newCtx()
		ctx.Print(nil, "....", false)
		ctx.Print(sourceNode(fileA, 0), "fileA-0", false)

		if got := segmentsPerLine(ctx); len(got) != 1 || got[0] != 2 {
			t.Fatalf("segmentsPerLine = %#v", got)
		}
	})

	t.Run("should handle indent", func(t *testing.T) {
		ctx := newCtx()
		ctx.IncIndent()
		ctx.Println(sourceNode(fileA, 0), "fileA-0")
		ctx.IncIndent()
		ctx.Println(sourceNode(fileA, 1), "fileA-1")
		ctx.DecIndent()
		ctx.Println(sourceNode(fileA, 2), "fileA-2")

		if got := ctx.ToSource(); got != "  fileA-0\n    fileA-1\n  fileA-2" {
			t.Fatalf("source = %q", got)
		}
		if got := segmentsPerLine(ctx); len(got) != 3 || got[0] != 2 || got[1] != 1 || got[2] != 1 {
			t.Fatalf("segmentsPerLine = %#v", got)
		}
	})

	t.Run("should coalesce identical span", func(t *testing.T) {
		ctx := newCtx()
		span := sourceNode(fileA, 0)
		ctx.Print(span, "fileA-0", false)
		ctx.Print(nil, "...", false)
		ctx.Print(span, "fileA-0", false)
		ctx.Print(sourceNode(fileB, 0), "fileB-0", false)

		if got := segmentsPerLine(ctx); len(got) != 1 || got[0] != 2 {
			t.Fatalf("segmentsPerLine = %#v", got)
		}
	})

	t.Run("should be able to shift the content", func(t *testing.T) {
		ctx := newCtx()
		ctx.Print(sourceNode(fileA, 0), "fileA-0", false)

		sm := ctx.ToSourceMapGenerator("o.ts", 10).ToJSON()
		if sm == nil {
			t.Fatal("expected source map")
		}
		if lines := strings.Count(sm.Mappings, ";") + 1; lines != 11 {
			t.Fatalf("mapping lines = %d", lines)
		}
	})
}

func sourceNode(file *parse_util.ParseSourceFile, idx int) sourceSpanNode {
	col := 2 * idx
	start := parse_util.NewParseLocation(file, col, 0, col)
	end := parse_util.NewParseLocation(file, col+2, 0, col+2)
	span := parse_util.NewParseSourceSpan(start, end, start, nil)
	return sourceSpanNode{span: NewSourceSpanAdapter(span)}
}

func segmentsPerLine(ctx *EmitterVisitorContext) []int {
	sm := ctx.ToSourceMapGenerator("o.ts", 0).ToJSON()
	if sm == nil {
		return nil
	}
	lines := strings.Split(sm.Mappings, ";")
	counts := make([]int, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			counts = append(counts, 1)
			continue
		}
		counts = append(counts, strings.Count(line, ",")+1)
	}
	return counts
}
