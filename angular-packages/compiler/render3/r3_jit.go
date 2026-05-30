package render3

// Port of angular/packages/compiler/src/render3/r3_jit.ts

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// ExternalReferenceResolver is an interface for resolving external references.
// Equivalent to ExternalReferenceResolver from output/output_jit.ts.
type ExternalReferenceResolver interface {
	ResolveExternalReference(ref output.ExternalReference) (interface{}, error)
}

// R3JitReflector implements ExternalReferenceResolver which resolves references to @angular/core
// symbols at runtime, according to a consumer-provided mapping.
//
// Only supports ResolveExternalReference, all other methods throw.
type R3JitReflector struct {
	context map[string]interface{}
}

// NewR3JitReflector creates a new R3JitReflector with the given context map.
func NewR3JitReflector(context map[string]interface{}) *R3JitReflector {
	return &R3JitReflector{context: context}
}

// ResolveExternalReference resolves an external reference against @angular/core context.
func (r *R3JitReflector) ResolveExternalReference(ref output.ExternalReference) (interface{}, error) {
	// This reflector only handles @angular/core imports.
	moduleName := ""
	if ref.ModuleName != nil {
		moduleName = *ref.ModuleName
	}
	if moduleName != "@angular/core" {
		return nil, fmt.Errorf(
			"Cannot resolve external reference to %s, only references to @angular/core are supported.",
			moduleName,
		)
	}
	name := ""
	if ref.Name != nil {
		name = *ref.Name
	}
	val, ok := r.context[name]
	if !ok {
		return nil, fmt.Errorf("No value provided for @angular/core symbol '%s'.", name)
	}
	return val, nil
}
