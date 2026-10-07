package health

import (
	"fmt"

	"github.com/noklash/hostcheck/internal/host"
	"github.com/noklash/hostcheck/internal/memory"
)

func EvaluateMemory(snapshot host.Snapshot, policy MemoryPolicy) Assessment {
	if snapshot.Memory == nil {
		return Assessment{
			Subject:      "memory",
			Availability: Unassessable,
			Reason:       observationUnavailableReason(snapshot, "memory"),
		}
	}

	availablePercent, err := memory.AvailablePercent(*snapshot.Memory)
	if err != nil {
		return Assessment{
			Subject:      "memory",
			Availability: Unassessable,
			Reason: fmt.Sprintf(
				"memory observation cannot be evaluated: %v",
				err,
			),
		}
	}

	assessment, err := EvaluateMemoryAvailablePercent(
		availablePercent,
		policy,
	)
	if err != nil {
		return Assessment{
			Subject:      "memory",
			Availability: Unassessable,
			Reason: fmt.Sprintf(
				"memory policy could not be evaluated: %v",
				err,
			),
		}
	}

	return assessment
}
