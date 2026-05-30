package cycles_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/cycles"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/perf"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/stretchr/testify/assert"
)

func makeAnalyzer(t *testing.T, graph string) (*ngtsctest.ProgramResult, *cycles.CycleAnalyzer) {
	result := makeProgramFromGraph(t, graph)
	return result, cycles.NewCycleAnalyzer(cycles.NewImportGraph(result.Program, perf.NOOP_PERF_RECORDER))
}

func TestCycleAnalyzer_NoCycleWhenNoneExists(t *testing.T) {
	result, analyzer := makeAnalyzer(t, "a:b,c;b;c")
	defer result.Release()

	b := result.GetSourceFile("/b.ts")
	c := result.GetSourceFile("/c.ts")

	assert.NotNil(t, b)
	assert.NotNil(t, c)

	assert.Nil(t, analyzer.WouldCreateCycle(b, c))
	assert.Nil(t, analyzer.WouldCreateCycle(c, b))
}

func TestCycleAnalyzer_SimpleCycleBetweenTwoFiles(t *testing.T) {
	result, analyzer := makeAnalyzer(t, "a:b;b")
	defer result.Release()

	a := result.GetSourceFile("/a.ts")
	b := result.GetSourceFile("/b.ts")

	assert.NotNil(t, a)
	assert.NotNil(t, b)

	assert.Nil(t, analyzer.WouldCreateCycle(a, b))

	cycle := analyzer.WouldCreateCycle(b, a)
	assert.NotNil(t, cycle, "b->a should create a cycle")
	assert.Equal(t, "b,a,b", importPath(cycle.GetPath()))
}

func TestCycleAnalyzer_DealsWithExistingCycles(t *testing.T) {
	// a -> b -> c -> d
	//      ^---------/
	result, analyzer := makeAnalyzer(t, "a:b;b:c;c:d;d:b")
	defer result.Release()

	a := result.GetSourceFile("/a.ts")
	b := result.GetSourceFile("/b.ts")
	c := result.GetSourceFile("/c.ts")
	d := result.GetSourceFile("/d.ts")

	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotNil(t, c)
	assert.NotNil(t, d)

	assert.Nil(t, analyzer.WouldCreateCycle(a, b))
	assert.Nil(t, analyzer.WouldCreateCycle(a, c))
	assert.Nil(t, analyzer.WouldCreateCycle(a, d))

	assert.NotNil(t, analyzer.WouldCreateCycle(b, a))
	assert.NotNil(t, analyzer.WouldCreateCycle(b, c))
	assert.NotNil(t, analyzer.WouldCreateCycle(b, d))
}

func TestCycleAnalyzer_CycleWithReExport(t *testing.T) {
	result, analyzer := makeAnalyzer(t, "a:*b;b:c;c")
	defer result.Release()

	a := result.GetSourceFile("/a.ts")
	c := result.GetSourceFile("/c.ts")

	assert.NotNil(t, a)
	assert.NotNil(t, c)

	assert.Nil(t, analyzer.WouldCreateCycle(a, c))

	cycle := analyzer.WouldCreateCycle(c, a)
	assert.NotNil(t, cycle, "c->a should create a cycle")
	assert.Equal(t, "c,a,b,c", importPath(cycle.GetPath()))
}

func TestCycleAnalyzer_ComplexProgram(t *testing.T) {
	result, analyzer := makeAnalyzer(t, "a:*b,*c;b:*e,*f;c:*g,*h;e:f;f:c;g;h:g")
	defer result.Release()

	b := result.GetSourceFile("/b.ts")
	g := result.GetSourceFile("/g.ts")

	assert.NotNil(t, b)
	assert.NotNil(t, g)

	assert.Nil(t, analyzer.WouldCreateCycle(b, g))

	cycle := analyzer.WouldCreateCycle(g, b)
	assert.NotNil(t, cycle, "g->b should create a cycle")
	assert.Equal(t, "g,b,f,c,g", importPath(cycle.GetPath()))
}

func TestCycleAnalyzer_SyntheticEdge(t *testing.T) {
	result, analyzer := makeAnalyzer(t, "a:b,c;b;c")
	defer result.Release()

	b := result.GetSourceFile("/b.ts")
	c := result.GetSourceFile("/c.ts")

	assert.NotNil(t, b)
	assert.NotNil(t, c)

	assert.Nil(t, analyzer.WouldCreateCycle(b, c))

	// Record a synthetic import c -> b
	analyzer.RecordSyntheticImport(c, b)

	cycle := analyzer.WouldCreateCycle(b, c)
	assert.NotNil(t, cycle, "b->c should create a cycle after synthetic import c->b")
	assert.Equal(t, "b,c,b", importPath(cycle.GetPath()))
}

func TestCycleAnalyzer_TypeOnlyImportsIgnored(t *testing.T) {
	// a:b,c! means a imports b normally, and imports c with type-only
	result, analyzer := makeAnalyzer(t, "a:b,c!;b;c")
	defer result.Release()

	a := result.GetSourceFile("/a.ts")
	b := result.GetSourceFile("/b.ts")
	c := result.GetSourceFile("/c.ts")

	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotNil(t, c)

	// c -> a would normally create a cycle via c!->a->b, but since c! is type-only,
	// c is not considered a dependency of a.
	assert.Nil(t, analyzer.WouldCreateCycle(c, a))

	// b -> a creates a cycle since a -> b is a normal import
	cycle := analyzer.WouldCreateCycle(b, a)
	assert.NotNil(t, cycle, "b->a should create a cycle")
	assert.Equal(t, "b,a,b", importPath(cycle.GetPath()))
}
