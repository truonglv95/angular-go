package perf

import (
	"time"
)

type HrTime time.Time

func Mark() HrTime {
	return HrTime(time.Now())
}

func TimeSinceInMicros(mark HrTime) int64 {
	return time.Since(time.Time(mark)).Microseconds()
}
