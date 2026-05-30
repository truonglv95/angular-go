package incremental

type IncrementalState struct {
	Versions     map[string]string
	EmittedFiles map[string]bool
}
