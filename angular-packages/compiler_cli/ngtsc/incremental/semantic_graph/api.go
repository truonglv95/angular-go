package semantic_graph

type SemanticSymbol struct {
	Path           string
	Identifier     string
	Kind           string            // "component" | "directive" | "pipe"
	Selector       string            // Component/Directive selector
	PipeName       string            // Pipe name
	Inputs         map[string]string // property name -> binding alias
	Outputs        map[string]string // property name -> binding alias
	ExportAs       []string          // ExportAs aliases
	Pure           bool              // Pipe pure flag
	Standalone     bool              // Component/Directive standalone flag
	Imports        []string          // NgModule/Component imports
	Exports        []string          // NgModule/Component exports
	Declarations   []string          // NgModule declarations
	TypeParameters []string          // Generic type parameters
}

func (recv *SemanticSymbol) IsPublicApiAffected(previousSymbol SemanticSymbol) bool {
	if recv.Path != previousSymbol.Path || recv.Identifier != previousSymbol.Identifier || recv.Kind != previousSymbol.Kind {
		return true
	}
	if recv.Selector != previousSymbol.Selector || recv.PipeName != previousSymbol.PipeName {
		return true
	}
	if recv.Pure != previousSymbol.Pure || recv.Standalone != previousSymbol.Standalone {
		return true
	}
	if !isStringSliceEqual(recv.ExportAs, previousSymbol.ExportAs) {
		return true
	}
	if !isStringSliceEqual(recv.Imports, previousSymbol.Imports) {
		return true
	}
	if !isStringSliceEqual(recv.Exports, previousSymbol.Exports) {
		return true
	}
	if !isStringSliceEqual(recv.Declarations, previousSymbol.Declarations) {
		return true
	}
	if !isStringSliceEqual(recv.TypeParameters, previousSymbol.TypeParameters) {
		return true
	}
	if !isStringMapEqual(recv.Inputs, previousSymbol.Inputs) {
		return true
	}
	if !isStringMapEqual(recv.Outputs, previousSymbol.Outputs) {
		return true
	}
	return false
}

func isStringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isStringMapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (recv *SemanticSymbol) IsEmitAffected(previousSymbol SemanticSymbol, publicApiAffected map[string]bool) bool {
	return recv.IsPublicApiAffected(previousSymbol)
}

func (recv *SemanticSymbol) IsTypeCheckApiAffected(previousSymbol SemanticSymbol) bool {
	if recv.Path != previousSymbol.Path || recv.Identifier != previousSymbol.Identifier {
		return true
	}
	if recv.Selector != previousSymbol.Selector || recv.PipeName != previousSymbol.PipeName {
		return true
	}
	if !isStringMapEqual(recv.Inputs, previousSymbol.Inputs) || !isStringMapEqual(recv.Outputs, previousSymbol.Outputs) {
		return true
	}
	if !isStringSliceEqual(recv.TypeParameters, previousSymbol.TypeParameters) {
		return true
	}
	return false
}

func (recv *SemanticSymbol) IsTypeCheckBlockAffected(previousSymbol SemanticSymbol, typeCheckApiAffected map[string]bool) bool {
	return recv.IsPublicApiAffected(previousSymbol)
}

type SemanticReference interface {
	GetSource() *SemanticSymbol
	GetTarget() *SemanticSymbol
}
