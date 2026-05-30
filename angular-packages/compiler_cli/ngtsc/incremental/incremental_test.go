package incremental_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncrementalReconciliation_ChangedVersion(t *testing.T) {
	fooPath := "/foo.ts"
	opts := ast.SourceFileParseOptions{
		FileName: fooPath,
	}

	sf := parser.ParseSourceFile(opts, "export const FOO = true;", core.ScriptKindTS)
	require.NotNil(t, sf)

	versionMapFirst := map[string]string{fooPath: "version.1"}
	firstCompilation := incremental.Fresh(versionMapFirst)
	firstCompilation.RecordSuccessfulAnalysis(nil)
	firstCompilation.RecordSuccessfulEmit(sf)

	versionMapSecond := map[string]string{fooPath: "version.2"}
	secondCompilation := incremental.Incremental(
		nil,
		versionMapSecond,
		nil,
		firstCompilation.State,
		make(map[string]bool),
		nil,
	)

	secondCompilation.RecordSuccessfulAnalysis(nil)
	assert.False(t, secondCompilation.SafeToSkipEmit(sf), "Should NOT be safe to skip emit because version has changed")
}
