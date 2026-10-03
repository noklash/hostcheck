package health

import (
	"fmt"
	"math"
)

type MemoryPolicy struct {
	DegradedBelowPercent float64
	CriticalBelowPercent float64
}

func (p MemoryPolicy) Validate() error {
	if p.DegradedBelowPercent < 0 || p.DegradedBelowPercent > 100 {
		return fmt.Errorf("degraded threshold must be between 0 and 100")
	}

	if p.CriticalBelowPercent < 0 || p.CriticalBelowPercent > 100 {
		return fmt.Errorf("critical threshold must be below degraded threshold")
	}

	if p.CriticalBelowPercent >= p.DegradedBelowPercent {
		return fmt.Errorf("critical threshold must be below degraded threshold")
	}

	return nil
}

func EvaluateMemoryAvailablePercent(
	availablePercent float64,
	policy MemoryPolicy,
) (Assessment, error) {
	if err := policy.Validate(); err != nil {
		return Assessment{}, err
	}

	if math.IsNaN(availablePercent) || availablePercent < 0 || availablePercent > 100 {
		return Assessment{}, fmt.Errorf(
			"available percentage must be between 0 and 100",
		)
	}

	assessment := Assessment{
		Subject:      "memory",
		Availability: Assessable,
		Evidence: []string{
			fmt.Sprintf("available_percent=%.2f", availablePercent),
		},
	}

	switch {
	case availablePercent < policy.CriticalBelowPercent:
		assessment.Status = Critical
		assessment.Reason = "available memory capacity is below the critical policy threshold"

	case availablePercent < policy.DegradedBelowPercent:
		assessment.Status = Degraded
		assessment.Reason = "available memory capacity is below the degraded policy threshold"

	default:
		assessment.Status = OK
		assessment.Reason = "available memory capacity is within the configured policy"
	}

	return assessment, nil
}
