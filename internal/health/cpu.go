package health

import "fmt"

// CPUUtilizationPolicy defines health thresholds for aggregate CPU utilization.
//
// Utilization below DegradedAbovePercent is healthy.
// Utilization at or above DegradedAbovePercent but below
// CriticalAbovePercent is degraded.
// Utilization at or above CriticalAbovePercent is critical.
type CPUUtilizationPolicy struct {
	DegradedAbovePercent float64
	CriticalAbovePercent float64
}

// Validate validates the CPU utilization health policy.
func (p CPUUtilizationPolicy) Validate() error {
	if p.DegradedAbovePercent < 0 || p.DegradedAbovePercent > 100 {
		return fmt.Errorf(
			"degraded CPU utilization threshold must be between 0 and 100: got %v",
			p.DegradedAbovePercent,
		)
	}

	if p.CriticalAbovePercent < 0 || p.CriticalAbovePercent > 100 {
		return fmt.Errorf(
			"critical CPU utilization threshold must be between 0 and 100: got %v",
			p.CriticalAbovePercent,
		)
	}

	if p.CriticalAbovePercent <= p.DegradedAbovePercent {
		return fmt.Errorf(
			"critical CPU utilization threshold must be greater than degraded threshold: degraded=%v critical=%v",
			p.DegradedAbovePercent,
			p.CriticalAbovePercent,
		)
	}

	return nil
}

// EvaluateCPUUtilization evaluates aggregate CPU utilization against a health policy.
//
// Utilization is expected to be a percentage in the range 0 to 100.
func EvaluateCPUUtilization(
	utilization float64,
	policy CPUUtilizationPolicy,
) (Assessment, error) {
	if err := policy.Validate(); err != nil {
		return Assessment{}, fmt.Errorf("CPU utilization policy: %w", err)
	}

	if utilization < 0 || utilization > 100 {
		return Assessment{}, fmt.Errorf(
			"CPU utilization must be between 0 and 100: got %v",
			utilization,
		)
	}

	status := OK
	reason := "CPU utilization is within the healthy range"

	switch {
	case utilization >= policy.CriticalAbovePercent:
		status = Critical
		reason = "CPU utilization is at or above the critical threshold"
	case utilization >= policy.DegradedAbovePercent:
		status = Degraded
		reason = "CPU utilization is at or above the degraded threshold"
	}

	return Assessment{
		Subject:      "cpu",
		Availability: Assessable,
		Status:       status,
		Reason:       reason,
		Evidence: []string{
			fmt.Sprintf("utilization_percent=%.2f", utilization),
		},
	}, nil
}
