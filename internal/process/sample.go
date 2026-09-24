package process

import "time"

// Sample represents process CPU accounting observed at a point in time.
//
// UTime and STime retain the cumulative CPU-time counters reported by Linux.
// ObservedAt records when the sample was taken.
type Sample struct {
	PID        int64
	StartTime  uint64
	UTime      uint64
	STime      uint64
	ObservedAt time.Time
}

// NewSample converts raw process statistics into a CPU accounting sample.
func NewSample(stats Stats, observedAt time.Time) Sample {
	return Sample{
		PID:        stats.PID,
		StartTime:  stats.StartTime,
		UTime:      stats.UTime,
		STime:      stats.STime,
		ObservedAt: observedAt,
	}
}
