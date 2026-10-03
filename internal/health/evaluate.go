package health

import (
	"fmt"

	"github.com/noklash/hostcheck/internal/host"
	"github.com/noklash/hostcheck/internal/memory"
)

type Policy struct {
	Memory     MemoryPolicy
	Filesystem FilesystemPolicy
}

func (p Policy) Validate() error {
	if err := p.Memory.Validate(); err != nil {
		return fmt.Errorf("memory policy: %w", err)
	}

	if err := p.Filesystem.Validate(); err != nil {
		return fmt.Errorf("filesystem policy: %w", err)
	}

	return nil
}

func EvaluateMemory(snapshot host.Snapshot, policy MemoryPolicy) Assessment {
	if snapshot.Memory == nil {
		return Assessment{
			Subject:      "memory",
			Availability: Unassessable,
			Reason:       "memory observation is unavailable",
		}
	}

	availablePercent, err := memory.AvailablePercent(*snapshot.Memory)
	if err != nil {
		return Assessment{
			Subject:      "memory",
			Availability: Unassessable,
			Reason:       fmt.Sprintf("memory observation cannot be evaluated: %v", err),
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
			Reason:       fmt.Sprintf("memory policy could not be evaluated: %v", err),
		}
	}

	return assessment
}
