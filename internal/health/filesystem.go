package health

import (
	"fmt"
	"math"
)

type FilesystemPolicy struct {
	DegradedBelowPercent float64
	CriticalBelowPercent float64
}

func (p FilesystemPolicy) Validate() error {
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

func EvaluateFilesystemAvailablePercent(
	availablePercent float64,
	policy FilesystemPolicy,
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
		Subject:      "filesystem",
		Availability: Assessable,
		Evidence: []string{
			fmt.Sprintf("available_percent=%.2f", availablePercent),
		},
	}

	switch {
	case availablePercent < policy.CriticalBelowPercent:
		assessment.Status = Critical
		assessment.Reason = "available filesystem capacity is below the critical policy threshold"

	case availablePercent < policy.DegradedBelowPercent:
		assessment.Status = Degraded
		assessment.Reason = "available filesystem capacity is below the degraded policy threshold"

	default:
		assessment.Status = OK
		assessment.Reason = "available filesystem capacity is within the configured policy"
	}

	return assessment, nil
}
