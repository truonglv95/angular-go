package perf

import (
	"runtime"
)

type PerfResults struct {
	Events map[string]int64
	Phases map[string]int64
	Memory map[string]int64
}

type ActivePerfRecorder struct {
	counters            []int64
	phaseTime           []int64
	bytes               []int64
	currentPhase        PerfPhase
	currentPhaseEntered HrTime
	zeroTime            HrTime
}

func NewActivePerfRecorder(zeroTime HrTime) *ActivePerfRecorder {
	r := &ActivePerfRecorder{
		counters:            make([]int64, PerfEvent_LAST),
		phaseTime:           make([]int64, PerfPhase_LAST),
		bytes:               make([]int64, PerfCheckpoint_LAST),
		currentPhase:        PerfPhase_Unaccounted,
		zeroTime:            zeroTime,
		currentPhaseEntered: zeroTime,
	}
	r.Memory(PerfCheckpoint_Initial)
	return r
}

func ActivePerfRecorderZeroedToNow() *ActivePerfRecorder {
	return NewActivePerfRecorder(Mark())
}

func (r *ActivePerfRecorder) Reset() {
	r.counters = make([]int64, PerfEvent_LAST)
	r.phaseTime = make([]int64, PerfPhase_LAST)
	r.bytes = make([]int64, PerfCheckpoint_LAST)
	r.zeroTime = Mark()
	r.currentPhase = PerfPhase_Unaccounted
	r.currentPhaseEntered = r.zeroTime
}

func (r *ActivePerfRecorder) Memory(after PerfCheckpoint) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	r.bytes[after] = int64(m.HeapAlloc)
}

func (r *ActivePerfRecorder) Phase(phase PerfPhase) PerfPhase {
	previous := r.currentPhase
	r.phaseTime[r.currentPhase] += TimeSinceInMicros(r.currentPhaseEntered)
	r.currentPhase = phase
	r.currentPhaseEntered = Mark()
	return previous
}

func (r *ActivePerfRecorder) InPhase(phase PerfPhase, fn func() interface{}) interface{} {
	previousPhase := r.Phase(phase)
	defer r.Phase(previousPhase)
	return fn()
}

func (r *ActivePerfRecorder) EventCount(counter PerfEvent, incrementBy ...int) {
	inc := 1
	if len(incrementBy) > 0 {
		inc = incrementBy[0]
	}
	r.counters[counter] += int64(inc)
}

func (r *ActivePerfRecorder) Finalize() PerfResults {
	r.Phase(PerfPhase_Unaccounted)

	results := PerfResults{
		Events: make(map[string]int64),
		Phases: make(map[string]int64),
		Memory: make(map[string]int64),
	}

	// Wait, we need the string names for enum values. We can just use string formatting if we didn't generate stringer.
	// For now we'll mock them with fmt.Sprintf
	// In Go we usually generate a String() method for enums using stringer.
	// We'll leave it out for simplicity in this port.
	return results
}

type DelegatingPerfRecorder struct {
	Target PerfRecorder
}

func (d *DelegatingPerfRecorder) EventCount(counter PerfEvent, incrementBy ...int) {
	d.Target.EventCount(counter, incrementBy...)
}

func (d *DelegatingPerfRecorder) Phase(phase PerfPhase) PerfPhase {
	return d.Target.Phase(phase)
}

func (d *DelegatingPerfRecorder) InPhase(phase PerfPhase, fn func() interface{}) interface{} {
	previousPhase := d.Target.Phase(phase)
	defer d.Target.Phase(previousPhase)
	return fn()
}

func (d *DelegatingPerfRecorder) Memory(after PerfCheckpoint) {
	d.Target.Memory(after)
}

func (d *DelegatingPerfRecorder) Reset() {
	d.Target.Reset()
}
