package incremental

type IncrementalBuildStrategy interface {
	GetIncrementalState(program any) IncrementalState
	SetIncrementalState(driver IncrementalState, program any) any
	ToNextBuildStrategy() IncrementalBuildStrategy
}

type NoopIncrementalBuildStrategy struct {
}

func (recv *NoopIncrementalBuildStrategy) GetIncrementalState() any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *NoopIncrementalBuildStrategy) SetIncrementalState() any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *NoopIncrementalBuildStrategy) ToNextBuildStrategy() IncrementalBuildStrategy {
	// TODO: stub
	panic("unimplemented")
}

type TrackedIncrementalBuildStrategy struct {
}

func (recv *TrackedIncrementalBuildStrategy) GetIncrementalState() IncrementalState {
	// TODO: stub
	panic("unimplemented")
}

func (recv *TrackedIncrementalBuildStrategy) SetIncrementalState(state IncrementalState) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *TrackedIncrementalBuildStrategy) ToNextBuildStrategy() TrackedIncrementalBuildStrategy {
	// TODO: stub
	panic("unimplemented")
}

type PatchedProgramIncrementalBuildStrategy struct {
}

func (recv *PatchedProgramIncrementalBuildStrategy) GetIncrementalState(program any) IncrementalState {
	// TODO: stub
	panic("unimplemented")
}

func (recv *PatchedProgramIncrementalBuildStrategy) SetIncrementalState(state IncrementalState, program any) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *PatchedProgramIncrementalBuildStrategy) ToNextBuildStrategy() IncrementalBuildStrategy {
	// TODO: stub
	panic("unimplemented")
}
