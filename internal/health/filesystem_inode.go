package health

import (
	"fmt"
	"math"
)

type FilesystemInodePolicy struct {
	DegradedBelowPercent float64
	CriticalBelowPercent float64
}

func (p FilesystemInodePolicy) Validate() error {
	if math.IsNaN(p.DegradedBelowPercent) || p.DegradedBelowPercent < 0 || p.DegradedBelowPercent > 100 {
		return fmt.Errorf("degraded threshold must be between 0 and 100")
	}

	if math.IsNaN(p.CriticalBelowPercent) || p.CriticalBelowPercent < 0 || p.CriticalBelowPercent > 100 {
		return fmt.Errorf("critical threshold must be between 0 and 100")
	}

	if p.CriticalBelowPercent >= p.DegradedBelowPercent {
		return fmt.Errorf("critical threshold must be below degraded threshold")
	}

	return nil
}

func EvaluateFilesystemAvailableInodePercent(
	availablePercent float64,
	policy FilesystemInodePolicy,
) (Assessment, error) {
	if err := policy.Validate(); err != nil {
		return Assessment{}, err
	}

	if math.IsNaN(availablePercent) || availablePercent < 0 || availablePercent > 100 {
		return Assessment{}, fmt.Errorf(
			"available inode percentage must be between 0 and 100",
		)
	}

	assessment := Assessment{
		Subject:      "filesystem_inodes",
		Availability: Assessable,
		Evidence: []string{
			fmt.Sprintf("available_inode_percent=%.2f", availablePercent),
		},
	}

	switch {
	case availablePercent < policy.CriticalBelowPercent:
		assessment.Status = Critical
		assessment.Reason = "available filesystem inodes are below the critical policy threshold"

	case availablePercent < policy.DegradedBelowPercent:
		assessment.Status = Degraded
		assessment.Reason = "available filesystem inodes are below the degraded policy threshold"

	default:
		assessment.Status = OK
		assessment.Reason = "available filesystem inodes are within the configured policy"
	}

	return assessment, nil
}
