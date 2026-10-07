package health

import (
	"fmt"

	"github.com/noklash/hostcheck/internal/host"
)

type SnapshotPolicy struct {
	Memory            MemoryPolicy
	Filesystem        FilesystemPolicy
	FilesystemInode   FilesystemInodePolicy
	ProcessState      *ProcessStatePolicy
	NetworkInterfaces []NetworkInterfacePolicy
	NetworkRoutes     []NetworkRoutePolicy
}

func (p SnapshotPolicy) Validate() error {
	if err := p.Memory.Validate(); err != nil {
		return fmt.Errorf("memory policy: %w", err)
	}

	if err := p.Filesystem.Validate(); err != nil {
		return fmt.Errorf("filesystem policy: %w", err)
	}

	if err := p.FilesystemInode.Validate(); err != nil {
		return fmt.Errorf("filesystem inode policy: %w", err)
	}

	if p.ProcessState != nil {
		if err := p.ProcessState.Validate(); err != nil {
			return fmt.Errorf("process state policy: %w", err)
		}
	}

	for i, policy := range p.NetworkInterfaces {
		if err := policy.Validate(); err != nil {
			return fmt.Errorf(
				"network interface policy %d: %w",
				i,
				err,
			)
		}
	}

	for i, policy := range p.NetworkRoutes {
		if err := policy.Validate(); err != nil {
			return fmt.Errorf(
				"network route policy %d: %w",
				i,
				err,
			)
		}
	}

	return nil
}

func EvaluateSnapshot(
	snapshot host.Snapshot,
	policy SnapshotPolicy,
) ([]Assessment, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	assessments := make([]Assessment, 0, 3+
		boolToInt(policy.ProcessState != nil)+
		len(policy.NetworkInterfaces)+
		len(policy.NetworkRoutes))

	assessments = append(
		assessments,
		EvaluateMemory(snapshot, policy.Memory),
		evaluateFilesystem(snapshot, policy.Filesystem),
		evaluateFilesystemInodes(snapshot, policy.FilesystemInode),
	)

	if policy.ProcessState != nil {
		if err := snapshotCollectionError(snapshot, "process"); err != nil {
			assessments = append(assessments, Assessment{
				Subject:      "process_state",
				Availability: Unassessable,
				Reason: fmt.Sprintf(
					"process collection failed: %v",
					err,
				),
			})
		} else {
			assessment, err := EvaluateProcessState(
				snapshot.Processes,
				*policy.ProcessState,
			)
			if err != nil {
				return nil, err
			}

			assessments = append(assessments, assessment)
		}
	}

	if len(policy.NetworkInterfaces) > 0 {
		for _, interfacePolicy := range policy.NetworkInterfaces {
			if snapshot.Network == nil {
				assessments = append(assessments, Assessment{
					Subject:      "network_interface",
					Availability: Unassessable,
					Reason:       networkObservationUnavailableReason(snapshot),
				})
				continue
			}

			assessment, err := EvaluateNetworkInterface(
				*snapshot.Network,
				interfacePolicy,
			)
			if err != nil {
				return nil, err
			}

			assessments = append(assessments, assessment)
		}
	}

	if len(policy.NetworkRoutes) > 0 {
		for _, routePolicy := range policy.NetworkRoutes {
			if snapshot.Network == nil {
				assessments = append(assessments, Assessment{
					Subject:      "network_route",
					Availability: Unassessable,
					Reason:       networkObservationUnavailableReason(snapshot),
				})
				continue
			}

			assessment, err := EvaluateNetworkRoute(
				*snapshot.Network,
				routePolicy,
			)
			if err != nil {
				return nil, err
			}

			assessments = append(assessments, assessment)
		}
	}

	return assessments, nil
}

func evaluateFilesystem(
	snapshot host.Snapshot,
	policy FilesystemPolicy,
) Assessment {
	if snapshot.Filesystem == nil {
		return Assessment{
			Subject:      "filesystem",
			Availability: Unassessable,
			Reason:       observationUnavailableReason(snapshot, "filesystem"),
		}
	}

	availablePercent, err := snapshot.Filesystem.AvailablePercent()
	if err != nil {
		return Assessment{
			Subject:      "filesystem",
			Availability: Unassessable,
			Reason: fmt.Sprintf(
				"filesystem observation cannot be evaluated: %v",
				err,
			),
		}
	}

	assessment, err := EvaluateFilesystemAvailablePercent(
		availablePercent,
		policy,
	)
	if err != nil {
		return Assessment{
			Subject:      "filesystem",
			Availability: Unassessable,
			Reason: fmt.Sprintf(
				"filesystem policy could not be evaluated: %v",
				err,
			),
		}
	}

	return assessment
}

func evaluateFilesystemInodes(
	snapshot host.Snapshot,
	policy FilesystemInodePolicy,
) Assessment {
	if snapshot.Filesystem == nil {
		return Assessment{
			Subject:      "filesystem_inodes",
			Availability: Unassessable,
			Reason: observationUnavailableReason(
				snapshot,
				"filesystem",
			),
		}
	}

	availablePercent, err := snapshot.Filesystem.AvailableInodePercent()
	if err != nil {
		return Assessment{
			Subject:      "filesystem_inodes",
			Availability: Unassessable,
			Reason: fmt.Sprintf(
				"filesystem inode observation cannot be evaluated: %v",
				err,
			),
		}
	}

	assessment, err := EvaluateFilesystemAvailableInodePercent(
		availablePercent,
		policy,
	)
	if err != nil {
		return Assessment{
			Subject:      "filesystem_inodes",
			Availability: Unassessable,
			Reason: fmt.Sprintf(
				"filesystem inode policy could not be evaluated: %v",
				err,
			),
		}
	}

	return assessment
}

func observationUnavailableReason(
	snapshot host.Snapshot,
	subsystem string,
) string {
	if err := snapshotCollectionError(snapshot, subsystem); err != nil {
		return fmt.Sprintf(
			"%s collection failed: %v",
			subsystem,
			err,
		)
	}

	return fmt.Sprintf(
		"%s observation is unavailable",
		subsystem,
	)
}

func networkObservationUnavailableReason(snapshot host.Snapshot) string {
	if err := snapshotCollectionError(snapshot, "network"); err != nil {
		return fmt.Sprintf(
			"network collection failed: %v",
			err,
		)
	}

	return "network observation is unavailable"
}

func snapshotCollectionError(
	snapshot host.Snapshot,
	subsystem string,
) error {
	for _, collectionError := range snapshot.Errors {
		if collectionError.Subsystem == subsystem {
			return collectionError.Err
		}
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
