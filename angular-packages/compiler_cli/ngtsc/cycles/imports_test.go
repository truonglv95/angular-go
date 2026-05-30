package cycles_test

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/cycles"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/perf"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
)

// makeProgramFromGraph constructs an in-memory TS program from a string DSL.
//
// The graph string consists of semicolon-separated files. Each file is specified as a name
// and (optionally) a comma-separated list of imports/exports. For example:
//
//	"a:b,c;b;c"
//
// means a.ts imports b.ts and c.ts. A * prefix indicates a re-export. A ! suffix indicates
// a type-only import.
func makeProgramFromGraph(t *testing.T, graph string) *ngtsctest.ProgramResult {
	t.Helper()
	var files []ngtsctest.ProgramFile
	for _, fileSegment := range strings.Split(graph, ";") {
		parts := strings.SplitN(fileSegment, ":", 2)
		name := parts[0]
		var importList []string
		if len(parts) == 2 && parts[1] != "" {
			importList = strings.Split(parts[1], ",")
		}

		var sb strings.Builder
		for _, i := range importList {
			if strings.HasPrefix(i, "*") {
				sym := i[1:]
				fmt.Fprintf(&sb, "export {%s} from './%s';\n", sym, sym)
			} else if strings.HasSuffix(i, "!") {
				sym := i[:len(i)-1]
				fmt.Fprintf(&sb, "import type {%s} from './%s';\n", sym, sym)
			} else {
				fmt.Fprintf(&sb, "import {%s} from './%s';\n", i, i)
			}
		}
		fmt.Fprintf(&sb, "export const %s = '%s';\n", name, name)

		files = append(files, ngtsctest.ProgramFile{
			Name:     "/" + name + ".ts",
			Contents: sb.String(),
		})
	}
	return ngtsctest.MakeProgram(t, files)
}

// importPath converts a slice of source files to a comma-separated string of basenames (without .ts).
func importPath(files []*ast.SourceFile) string {
	parts := make([]string, len(files))
	for i, sf := range files {
		base := filepath.Base(sf.FileName())
		parts[i] = strings.TrimSuffix(base, ".ts")
	}
	return strings.Join(parts, ",")
}

// importsToString converts a set of source files to a sorted comma-separated string of basenames.
func importsToString(imports map[*ast.SourceFile]struct{}) string {
	var parts []string
	for sf := range imports {
		base := filepath.Base(sf.FileName())
		parts = append(parts, strings.TrimSuffix(base, ".ts"))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func makeImportGraph(t *testing.T, graph string) (*ngtsctest.ProgramResult, *cycles.ImportGraph) {
	result := makeProgramFromGraph(t, graph)
	return result, cycles.NewImportGraph(result.Program, perf.NOOP_PERF_RECORDER)
}

func TestImportGraph_ImportsOf_SimpleProgram(t *testing.T) {
	result, graph := makeImportGraph(t, "a:b;b:c;c")
	defer result.Release()

	a := result.GetSourceFile("/a.ts")
	b := result.GetSourceFile("/b.ts")
	c := result.GetSourceFile("/c.ts")

	assert.NotNil(t, a, "a.ts should be in program")
	assert.NotNil(t, b, "b.ts should be in program")
	assert.NotNil(t, c, "c.ts should be in program")

	assert.Equal(t, "b", importsToString(graph.ImportsOf(a)), "a.ts should import b.ts")
	assert.Equal(t, "c", importsToString(graph.ImportsOf(b)), "b.ts should import c.ts")
}

func TestImportGraph_FindPath_CycleExists(t *testing.T) {
	result, graph := makeImportGraph(t, "a:*b,*c;b:*e,*f;c:*g,*h;e:f;f;g:e;h:g")
	defer result.Release()

	a := result.GetSourceFile("/a.ts")
	b := result.GetSourceFile("/b.ts")
	c := result.GetSourceFile("/c.ts")
	e := result.GetSourceFile("/e.ts")

	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotNil(t, c)
	assert.NotNil(t, e)

	path := graph.FindPath(a, a)
	assert.NotNil(t, path)
	assert.Equal(t, "a", importPath(path))

	path = graph.FindPath(a, b)
	assert.NotNil(t, path)
	assert.Equal(t, "a,b", importPath(path))

	path = graph.FindPath(c, e)
	assert.NotNil(t, path)
	assert.Equal(t, "c,g,e", importPath(path))

	assert.Nil(t, graph.FindPath(e, c), "e->c path should not exist")
	assert.Nil(t, graph.FindPath(b, c), "b->c path should not exist")
}

func TestImportGraph_FindPath_CircularDependencies(t *testing.T) {
	// a -> b -> c -> d
	// ^----/    |
	// ^---------/
	result, graph := makeImportGraph(t, "a:b;b:a,c;c:a,d;d")
	defer result.Release()

	a := result.GetSourceFile("/a.ts")
	d := result.GetSourceFile("/d.ts")

	assert.NotNil(t, a)
	assert.NotNil(t, d)

	path := graph.FindPath(a, d)
	assert.NotNil(t, path)
	assert.Equal(t, "a,b,c,d", importPath(path))
}
