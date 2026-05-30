package incremental

type FileDependencyGraph struct {
}

func (recv *FileDependencyGraph) AddDependency(from any, on any) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *FileDependencyGraph) AddResourceDependency(from any, resource string) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *FileDependencyGraph) RecordDependencyAnalysisFailure(file any) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *FileDependencyGraph) GetResourceDependencies(from any) []string {
	// TODO: stub
	panic("unimplemented")
}

func (recv *FileDependencyGraph) UpdateWithPhysicalChanges(previous FileDependencyGraph, changedTsPaths map[string]bool, deletedTsPaths map[string]bool, changedResources map[string]bool) map[string]bool {
	// TODO: stub
	panic("unimplemented")
}
