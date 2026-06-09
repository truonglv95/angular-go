package program_driver

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/shims"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/tspath"
)

type shimTaggingHost struct {
	compiler.CompilerHost
	shimExtensionPrefixes []string
}

func (h *shimTaggingHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	sf := h.CompilerHost.GetSourceFile(opts)
	if sf != nil && len(h.shimExtensionPrefixes) > 0 {
		tagger := shims.NewShimReferenceTagger(h.shimExtensionPrefixes)
		tagger.Tag(sf)
	}
	return sf
}

type TsCreateProgramDriver struct {
	program          *compiler.Program
	host             compiler.CompilerHost
	options          any
	inliningMode     InliningMode
	LastUpdateReused bool
}

func NewTsCreateProgramDriver(
	originalProgram any,
	originalHost any,
	options any,
	shimExtensionPrefixes []string,
) *TsCreateProgramDriver {
	prog, _ := originalProgram.(*compiler.Program)
	host, _ := originalHost.(compiler.CompilerHost)
	if host != nil && len(shimExtensionPrefixes) > 0 {
		host = &shimTaggingHost{
			CompilerHost:          host,
			shimExtensionPrefixes: shimExtensionPrefixes,
		}
	}
	return &TsCreateProgramDriver{
		program:          prog,
		host:             host,
		options:          options,
		inliningMode:     InliningMode_InlineOps,
		LastUpdateReused: true,
	}
}

func (d *TsCreateProgramDriver) SetInliningMode(mode InliningMode) {
	d.inliningMode = mode
}

func (d *TsCreateProgramDriver) GetInliningMode() InliningMode {
	return d.inliningMode
}

func (d *TsCreateProgramDriver) GetSupportsInlineOperations() bool {
	return true
}

func (d *TsCreateProgramDriver) GetProgram() any {
	return d.program
}

func (d *TsCreateProgramDriver) UpdateFiles(contents map[file_system.AbsoluteFsPath]FileUpdate, updateMode UpdateMode) {
	d.LastUpdateReused = true
	for path, update := range contents {
		_ = d.host.FS().WriteFile(string(path), update.NewText)
		newProg, reused := d.program.UpdateProgram(tspath.Path(path), d.host, nil)
		d.program = newProg
		if !reused {
			d.LastUpdateReused = false
		}
	}
}

func (d *TsCreateProgramDriver) GetSourceFileVersion(sf ast.SourceFile) string {
	return fmt.Sprintf("%d", len(sf.Text()))
}
