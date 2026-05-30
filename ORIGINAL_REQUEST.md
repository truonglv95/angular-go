# Original User Request

## Initial Request — 2026-05-29T11:01:23Z

# Teamwork Project Prompt — Draft

> Status: Ready for launch — awaiting user approval
> Goal: Craft prompt → get user approval → delegate to teamwork_preview

Port the remaining 69 TypeScript files from the original `@angular/compiler-cli` (`packages/compiler/src/template/pipeline/src/phases`) to Go (`angular-packages/compiler/template/pipeline/phases`). The Go implementation must maintain strict 1:1 parity with the TypeScript AST logic.

Working directory: `/Users/truong/Documents/angular-typescript-go/typescript-go/angular-packages`
Integrity mode: benchmark

## Requirements

### R1. Complete Translation
Port all remaining 69 TypeScript files from the `phases` pipeline step into Go. Ensure no file is skipped. 

### R2. Strict 1:1 Structural Parity
Do not use regex or arbitrary string manipulation. You must rely on the AST tree structure and Visitor interfaces provided in `compiler/template/pipeline/ir` and `compiler/template/pipeline/compilation`. The implementation must mirror the exact logic flow and variable assignments found in the TS source code.

### R3. Test Coverage
You must port or write corresponding unit test files (e.g., `_test.go`) for these phases to mathematically and objectively prove the AST logic behaves identically to TypeScript.

## Acceptance Criteria

### Correctness & Build
- [ ] Running `go build ./...` inside `compiler/template/pipeline/phases` completes without any syntax or type checking errors.
- [ ] Running `go test ./...` inside `compiler/template/pipeline/phases` completes with 100% success.

### Fidelity
- [ ] A review of the Go files confirms they use AST traversal logic mapping strictly to the TypeScript implementations, rather than ad-hoc parsing.
- [ ] External dependencies outside of the codebase and standard library were not added.

## Follow-up — 2026-05-29T11:19:59Z

The user has reviewed the first patch (Milestone 1) of the phases.
The go build, go vet, and gofmt checks passed with 100% success (0 errors, 0 warnings).
The user is very happy with the strict AST implementation.
You are approved to proceed to the next batch of files. Keep up the great work and ensure strict parity!

## Follow-up — 2026-05-29T19:45:50+07:00

# Teamwork Project Prompt — Draft

> Status: Launched
> Goal: Wait for teamwork_preview subagent to complete the task

Port the remaining 69 TypeScript files from the original `@angular/compiler-cli` (`packages/compiler/src/template/pipeline/src/phases`) to Go (`angular-packages/compiler/template/pipeline/phases`). The Go implementation must maintain strict 1:1 parity with the TypeScript AST logic.

Working directory: `/Users/truong/Documents/angular-typescript-go/typescript-go/angular-packages`
Integrity mode: benchmark

## Requirements

### R1. Complete Translation
Port all remaining 69 TypeScript files from the `phases` pipeline step into Go. Ensure no file is skipped. 

### R2. Strict 1:1 Structural Parity
Do not use regex or arbitrary string manipulation. You must rely on the AST tree structure and Visitor interfaces provided in `compiler/template/pipeline/ir` and `compiler/template/pipeline/compilation`. The implementation must mirror the exact logic flow and variable assignments found in the TS source code.

### R3. Test Coverage
You must port or write corresponding unit test files (e.g., `_test.go`) for these phases to mathematically and objectively prove the AST logic behaves identically to TypeScript.

## Acceptance Criteria

### Correctness & Build
- [ ] Running `go build ./...` inside `compiler/template/pipeline/phases` completes without any syntax or type checking errors.
- [ ] Running `go test ./...` inside `compiler/template/pipeline/phases` completes with 100% success.

### Fidelity
- [ ] A review of the Go files confirms they use AST traversal logic mapping strictly to the TypeScript implementations, rather than ad-hoc parsing.
- [ ] External dependencies outside of the codebase and standard library were not added.

SYSTEM NOTE: The previous orchestrator agent crashed due to a temporary network/DNS glitch. You must RESUME execution from where it left off (Milestone 2) based on the states in `.agents/sentinel/handoff.md` and any existing progress artifacts. Continue tracking progress and communicating back normally.

