package process

import (
	"fmt"
	"time"
)

const clkTCK = 100

// Delta represents the change in process CPU accounting between two samples.
type Delta struct {
	PID            int64
	CPUTimeTicks   uint64
	CPUTimeSeconds float64
	Elapsed        time.Duration
}

// DeltaSamples calculates process CPU consumption between two samples.
//
// The samples must refer to the same process instance. PID reuse is detected
// using StartTime, which identifies the process lifetime within the PID space.
func DeltaSamples(previous, current Sample) (Delta, error) {
	if previous.PID != current.PID {
		return Delta{}, fmt.Errorf(
			"pid changed: previous=%d current=%d",
			previous.PID,
			current.PID,
		)
	}

	if previous.StartTime != current.StartTime {
		return Delta{}, fmt.Errorf(
			"process identity changed for pid %d: previous start=%d current start=%d",
			current.PID,
			previous.StartTime,
			current.StartTime,
		)
	}

	if current.ObservedAt.Before(previous.ObservedAt) {
		return Delta{}, fmt.Errorf("sample timestamp moved backwards")
	}

	if current.UTime < previous.UTime {
		return Delta{}, fmt.Errorf("utime counter moved backwards")
	}

	if current.STime < previous.STime {
		return Delta{}, fmt.Errorf("stime counter moved backwards")
	}

	elapsed := current.ObservedAt.Sub(previous.ObservedAt)
	if elapsed <= 0 {
		return Delta{}, fmt.Errorf("elapsed time must be positive")
	}

	userTicks := current.UTime - previous.UTime
	systemTicks := current.STime - previous.STime
	totalTicks := userTicks + systemTicks

	return Delta{
		PID:            current.PID,
		CPUTimeTicks:   totalTicks,
		CPUTimeSeconds: float64(totalTicks) / clkTCK,
		Elapsed:        elapsed,
	}, nil
}
