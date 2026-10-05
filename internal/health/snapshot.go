package health

import (
	"fmt"

	"github.com/noklash/hostcheck/internal/host"
)

type SnapshotPolicy struct {
	Memory          MemoryPolicy
	Filesystem      FilesystemPolicy
	FilesystemInode FilesystemInodePolicy
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

	return nil
}

func EvaluateSnapshot(
	snapshot host.Snapshot,
	policy SnapshotPolicy,
) ([]Assessment, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	assessments := []Assessment{
		EvaluateMemory(snapshot, policy.Memory),
		evaluateFilesystem(snapshot, policy.Filesystem),
		evaluateFilesystemInodes(snapshot, policy.FilesystemInode),
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
			Reason:       "filesystem observation is unavailable",
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
			Reason:       "filesystem inode observation is unavailable",
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
