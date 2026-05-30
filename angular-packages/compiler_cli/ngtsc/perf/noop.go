package perf

type NoopPerfRecorder struct{}

func (n *NoopPerfRecorder) EventCount(event PerfEvent, incrementBy ...int) {}

func (n *NoopPerfRecorder) Memory(after PerfCheckpoint) {}

func (n *NoopPerfRecorder) Phase(phase PerfPhase) PerfPhase {
	return PerfPhase_Unaccounted
}

func (n *NoopPerfRecorder) InPhase(phase PerfPhase, fn func() interface{}) interface{} {
	return fn()
}

func (n *NoopPerfRecorder) Reset() {}

var NOOP_PERF_RECORDER PerfRecorder = &NoopPerfRecorder{}
