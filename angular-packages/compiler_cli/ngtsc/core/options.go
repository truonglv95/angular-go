package core

import (
	"github.com/microsoft/typescript-go/internal/tsoptions"
)

type NgCompilerOptions struct {
	CompilationMode  string
	StrictTemplates  bool
	EnableHmr        bool
	InvalidatedFiles map[string]bool
}

type ParsedConfiguration struct {
	Options      NgCompilerOptions
	TsParsedOpts *tsoptions.ParsedCommandLine
}
