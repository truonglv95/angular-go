package core

import (
	"github.com/microsoft/typescript-go/internal/tsoptions"
)

type NgCompilerOptions struct {
	CompilationMode string
	StrictTemplates bool
	// ... add other options as needed
}

type ParsedConfiguration struct {
	Options      NgCompilerOptions
	TsParsedOpts *tsoptions.ParsedCommandLine
}
