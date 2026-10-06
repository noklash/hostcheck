package health

import (
	"fmt"

	"github.com/noklash/hostcheck/internal/process"
)

// ProcessStatePolicy defines health thresholds for processes observed in
// uninterruptible sleep.
//
// A process in state D is waiting in uninterruptible sleep, commonly while
// waiting on a kernel resource such as I/O. The presence of a D-state process
// is not by itself proof of a host failure, so the health decision is based
// on an explicit count policy.
type ProcessStatePolicy struct {
	DegradedAtOrAbove int
	CriticalAtOrAbove int
}

// Validate validates the process state health policy.
func (p ProcessStatePolicy) Validate() error {
	if p.DegradedAtOrAbove < 0 {
		return fmt.Errorf(
			"degraded process state threshold cannot be negative: got %d",
			p.DegradedAtOrAbove,
		)
	}

	if p.CriticalAtOrAbove < 0 {
		return fmt.Errorf(
			"critical process state threshold cannot be negative: got %d",
			p.CriticalAtOrAbove,
		)
	}

	if p.CriticalAtOrAbove <= p.DegradedAtOrAbove {
		return fmt.Errorf(
			"critical process state threshold must be greater than degraded threshold: degraded=%d critical=%d",
			p.DegradedAtOrAbove,
			p.CriticalAtOrAbove,
		)
	}

	return nil
}

// EvaluateProcessState evaluates the number of processes observed in
// uninterruptible sleep (D state) against a health policy.
//
// The assessment describes the process states observed during one collection.
// It does not establish how long any process remained in that state.
func EvaluateProcessState(
	processes []process.Process,
	policy ProcessStatePolicy,
) (Assessment, error) {
	if err := policy.Validate(); err != nil {
		return Assessment{}, fmt.Errorf("process state policy: %w", err)
	}

	dStateCount := 0

	for _, proc := range processes {
		if proc.Stats.State == 'D' {
			dStateCount++
		}
	}

	status := OK
	reason := "no processes were observed in uninterruptible sleep"

	switch {
	case dStateCount >= policy.CriticalAtOrAbove:
		status = Critical
		reason = "observed D-state process count is at or above the critical threshold"

	case dStateCount >= policy.DegradedAtOrAbove:
		status = Degraded
		reason = "observed D-state process count is at or above the degraded threshold"
	}

	return Assessment{
		Subject:      "process_state",
		Availability: Assessable,
		Status:       status,
		Reason:       reason,
		Evidence: []string{
			fmt.Sprintf("d_state_processes=%d", dStateCount),
		},
	}, nil
}
