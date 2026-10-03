package health

import (
	"testing"

	"github.com/noklash/hostcheck/internal/filesystem"
	"github.com/noklash/hostcheck/internal/host"
	"github.com/noklash/hostcheck/internal/memory"
)

func TestEvaluateSnapshot(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf("EvaluateSnapshot() error = %v", err)
	}

	if len(assessments) != 2 {
		t.Fatalf("len(assessments) = %d, want 2", len(assessments))
	}

	if assessments[0].Subject != "memory" {
		t.Fatalf(
			"assessments[0].Subject = %q, want memory",
			assessments[0].Subject,
		)
	}

	if assessments[0].Status != OK {
		t.Fatalf(
			"memory status = %q, want %q",
			assessments[0].Status,
			OK,
		)
	}

	if assessments[1].Subject != "filesystem" {
		t.Fatalf(
			"assessments[1].Subject = %q, want filesystem",
			assessments[1].Subject,
		)
	}

	if assessments[1].Status != OK {
		t.Fatalf(
			"filesystem status = %q, want %q",
			assessments[1].Status,
			OK,
		)
	}

	for _, assessment := range assessments {
		if err := assessment.Validate(); err != nil {
			t.Fatalf(
				"%s assessment.Validate() error = %v",
				assessment.Subject,
				err,
			)
		}
	}
}

func TestEvaluateSnapshotMarksMissingObservationsUnassessable(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	assessments, err := EvaluateSnapshot(host.Snapshot{}, policy)
	if err != nil {
		t.Fatalf("EvaluateSnapshot() error = %v", err)
	}

	if len(assessments) != 2 {
		t.Fatalf("len(assessments) = %d, want 2", len(assessments))
	}

	for _, assessment := range assessments {
		if assessment.Availability != Unassessable {
			t.Fatalf(
				"%s availability = %q, want %q",
				assessment.Subject,
				assessment.Availability,
				Unassessable,
			)
		}

		if assessment.Status != "" {
			t.Fatalf(
				"%s status = %q, want empty",
				assessment.Subject,
				assessment.Status,
			)
		}

		if err := assessment.Validate(); err != nil {
			t.Fatalf(
				"%s assessment.Validate() error = %v",
				assessment.Subject,
				err,
			)
		}
	}
}

func TestEvaluateSnapshotRejectsInvalidPolicy(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 10,
			CriticalBelowPercent: 20,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	if _, err := EvaluateSnapshot(host.Snapshot{}, policy); err == nil {
		t.Fatal("EvaluateSnapshot() error = nil, want error")
	}
}
