package compiler

// Port of angular/packages/compiler/src/compiler.ts
//
// This file serves as the entry point for the Angular compiler, re-exporting
// all public APIs from the various sub-packages.
//
// In Go, this file documents the public surface area of the compiler package.
// Actual implementations are in the respective files.
//
// Public API surface of @angular/compiler - see compiler.ts for the TypeScript version.

// --- Core/Common Exports ---
// Version, core.EmitDistinctChangesOnlyDefaultValue: compiler_util.go, version.go, core.go

// --- Character Utilities ---
// IsWhitespace, IsDigit, etc.: chars.go

// --- CSS Selector / Directive Matching ---
// CssSelector, CssSelectorParse, SelectorMatcher, SelectorListContext, SelectorContext: go

// --- Property Mapping ---
// ClassPropertyMapping, InputOrOutput, ClassPropertyName, BindingPropertyName: go

// --- Resource Loader ---
// ResourceLoader: resource_loader.go

// --- Style URL Resolver ---
// IsStyleUrlResolvable: go

// --- Compiler Config ---
// CompilerConfig, PreserveWhitespacesDefault: config.go

// --- Compiler Facade Interface ---
// CompilerFacade, CompilerFacadeImpl, CoreEnvironment, R3DependencyMetadataFacade,
// R3InjectableMetadataFacade, R3DirectiveMetadataFacade, etc.: compiler_facade_interface.go

// --- JIT Compiler Facade ---
// CompilerFacadeImpl, PublishFacade, etc.: jit_compiler_facade.go

// --- Combined Visitor ---
// CombinedRecursiveAstVisitor: combined_visitor.go

// --- Injectable Compiler ---
// R3InjectableMetadata, CompileInjectable, CreateInjectableType, DelegateToFactory: injectable_compiler_2.go

// --- Service Compiler ---
// R3ServiceMetadata, CompileService: service_compiler.go

// --- Parse Utilities ---
// parse_util.ParseLocation, parse_util.ParseSourceSpan, parse_util.ParseSourceFile, parse_util.ParseError: parse_util.go

// --- Constant Pool ---
// ConstantPool: go

// --- Core Declarations ---
// core.ViewEncapsulation, core.ChangeDetectionStrategy, core.MissingTranslationStrategy, etc.: core.go

// --- Utilities ---
// DashCaseToCamelCase, SplitAtColon, Stringify, GetJitStandaloneDefaultForVersion: compiler_util.go

// --- Legacy Optional Chaining ---
// LegacyOptionalChainingDefault: legacy_optional_chaining_default.go
