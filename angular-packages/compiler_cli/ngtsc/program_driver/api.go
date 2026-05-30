package program_driver

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/internal/ast"
)

type FileUpdate struct {
	// The source file text.
	NewText string

	// Represents the source file from the original program that is being updated. If the file update
	// targets a shim file then this is null, as shim files do not have an associated original file.
	OriginalFile ast.SourceFile
}

const NgOriginalFile = "NgOriginalFile"

type MaybeSourceFileWithOriginalFile interface {
	ast.SourceFile
	GetNgOriginalFile() ast.SourceFile
	SetNgOriginalFile(file ast.SourceFile)
}

// How a type-checking context should handle operations which would require inlining.
type InliningMode int

const (
	InliningMode_InlineOps InliningMode = iota
	InliningMode_Error
	InliningMode_CopySourceToTcb
)

type ProgramDriver interface {
	// How this strategy handles operations that require inlining.
	GetInliningMode() InliningMode

	// Whether this strategy supports modifying user files (inline modifications) in addition to
	// modifying type-checking shims.
	GetSupportsInlineOperations() bool

	// Retrieve the latest version of the program, containing all the updates made thus far.
	GetProgram() any

	// Incorporate a set of changes to either augment or completely replace the type-checking code
	// included in the type-checking program.
	UpdateFiles(contents map[file_system.AbsoluteFsPath]FileUpdate, updateMode UpdateMode)

	// Retrieve a string version for a given `ts.SourceFile`, which much change when the contents of
	// the file have changed.
	//
	// If this method is present, the compiler will use these versions in addition to object identity
	// for `ts.SourceFile`s to determine what's changed between two incremental programs. This is
	// valuable for some clients (such as the Language Service) that treat `ts.SourceFile`s as mutable
	// objects.
	GetSourceFileVersion(sf ast.SourceFile) string
}

type UpdateMode int

const (
	// A complete update creates a completely new overlay of type-checking code on top of the user's
	// original program, which doesn't include type-checking code from previous calls to
	// `updateFiles`.
	UpdateMode_Complete UpdateMode = iota

	// An incremental update changes the contents of some files in the type-checking program without
	// reverting any prior changes.
	UpdateMode_Incremental
)
