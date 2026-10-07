package health

import (
	"fmt"

	"github.com/noklash/hostcheck/internal/host"
	"github.com/noklash/hostcheck/internal/memory"
)

// Evaluate evaluates a host snapshot under the supplied policy and combines
// the resulting assessments into one host-level health result.
//
// Snapshot evaluation determines which health propositions can be assessed
// from the supplied observations. Aggregation then combines those individual
// assessments into one status and coverage result.
func Evaluate(
	snapshot host.Snapshot,
	policy SnapshotPolicy,
) (Result, error) {
	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		return Result{}, err
	}

	return Aggregate(assessments)
}

// EvaluateMemory evaluates the memory observation in a host snapshot against
// the supplied memory policy.
func EvaluateMemory(
	snapshot host.Snapshot,
	policy MemoryPolicy,
) Assessment {
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
