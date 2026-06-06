package incremental

type IncrementalBuildStrategy interface {
	GetIncrementalState(program any) IncrementalState
	SetIncrementalState(state IncrementalState, program any) any
	ToNextBuildStrategy() IncrementalBuildStrategy
}

type NoopIncrementalBuildStrategy struct{}

func (recv *NoopIncrementalBuildStrategy) GetIncrementalState(program any) IncrementalState {
	return IncrementalState{
		Versions:     make(map[string]string),
		EmittedFiles: make(map[string]bool),
	}
}

func (recv *NoopIncrementalBuildStrategy) SetIncrementalState(state IncrementalState, program any) any {
	return nil
}

func (recv *NoopIncrementalBuildStrategy) ToNextBuildStrategy() IncrementalBuildStrategy {
	return recv
}

type TrackedIncrementalBuildStrategy struct {
	state IncrementalState
}

func NewTrackedIncrementalBuildStrategy() *TrackedIncrementalBuildStrategy {
	return &TrackedIncrementalBuildStrategy{
		state: IncrementalState{
			Versions:     make(map[string]string),
			EmittedFiles: make(map[string]bool),
		},
	}
}

func (recv *TrackedIncrementalBuildStrategy) GetIncrementalState(program any) IncrementalState {
	return recv.state
}

func (recv *TrackedIncrementalBuildStrategy) SetIncrementalState(state IncrementalState, program any) any {
	recv.state = state
	return nil
}

func (recv *TrackedIncrementalBuildStrategy) ToNextBuildStrategy() IncrementalBuildStrategy {
	return recv
}

type PatchedProgramIncrementalBuildStrategy struct {
	fallback *TrackedIncrementalBuildStrategy
}

func NewPatchedProgramIncrementalBuildStrategy() *PatchedProgramIncrementalBuildStrategy {
	return &PatchedProgramIncrementalBuildStrategy{
		fallback: NewTrackedIncrementalBuildStrategy(),
	}
}

func (recv *PatchedProgramIncrementalBuildStrategy) GetIncrementalState(program any) IncrementalState {
	return recv.fallback.GetIncrementalState(program)
}

func (recv *PatchedProgramIncrementalBuildStrategy) SetIncrementalState(state IncrementalState, program any) any {
	return recv.fallback.SetIncrementalState(state, program)
}

func (recv *PatchedProgramIncrementalBuildStrategy) ToNextBuildStrategy() IncrementalBuildStrategy {
	return recv
}
