package compiler

import "github.com/microsoft/typescript-go/angular-packages/compiler/core"

// CompilerConfig holds configuration for the Angular compiler.
type CompilerConfig struct {
	DefaultEncapsulation      *core.ViewEncapsulation
	PreserveWhitespaces       bool
	StrictInjectionParameters bool
}

// CompilerConfigOptions holds optional configuration parameters.
type CompilerConfigOptions struct {
	DefaultEncapsulation      *core.ViewEncapsulation
	PreserveWhitespaces       *bool
	StrictInjectionParameters *bool
}

// NewCompilerConfig creates a new CompilerConfig with the given options.
func NewCompilerConfig(opts CompilerConfigOptions) *CompilerConfig {
	enc := core.ViewEncapsulationEmulated
	defaultEncapsulation := &enc
	if opts.DefaultEncapsulation != nil {
		defaultEncapsulation = opts.DefaultEncapsulation
	}

	var preserveWhitespacesPtr *bool
	if opts.PreserveWhitespaces != nil {
		preserveWhitespacesPtr = opts.PreserveWhitespaces
	}

	strictInjection := false
	if opts.StrictInjectionParameters != nil {
		strictInjection = *opts.StrictInjectionParameters
	}

	return &CompilerConfig{
		DefaultEncapsulation:      defaultEncapsulation,
		PreserveWhitespaces:       PreserveWhitespacesDefault(NoUndefinedBool(preserveWhitespacesPtr)),
		StrictInjectionParameters: strictInjection,
	}
}

// PreserveWhitespacesDefault returns the effective preserve-whitespaces setting.
// If the option is nil (null), returns defaultSetting; otherwise returns the option value.
func PreserveWhitespacesDefault(preserveWhitespacesOption *bool, defaultSetting ...bool) bool {
	defSetting := false
	if len(defaultSetting) > 0 {
		defSetting = defaultSetting[0]
	}
	if preserveWhitespacesOption == nil {
		return defSetting
	}
	return *preserveWhitespacesOption
}

// NoUndefinedBool converts a *bool, treating nil (undefined) as nil (null).
// In the TS source, noUndefined converts undefined → null; here we keep *bool as-is.
func NoUndefinedBool(val *bool) *bool {
	return val
}
