package health

import (
	"testing"

	"github.com/noklash/hostcheck/internal/filesystem"
	"github.com/noklash/hostcheck/internal/host"
	"github.com/noklash/hostcheck/internal/memory"
)

func TestEvaluateMemory(t *testing.T) {
	policy := MemoryPolicy{
		DegradedBelowPercent: 20,
		CriticalBelowPercent: 10,
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
	}

	assessment := EvaluateMemory(snapshot, policy)

	if assessment.Subject != "memory" {
		t.Fatalf("Subject = %q, want memory", assessment.Subject)
	}

	if assessment.Availability != Assessable {
		t.Fatalf(
			"Availability = %q, want %q",
			assessment.Availability,
			Assessable,
		)
	}

	if assessment.Status != OK {
		t.Fatalf("Status = %q, want %q", assessment.Status, OK)
	}

	if err := assessment.Validate(); err != nil {
		t.Fatalf("Assessment.Validate() error = %v", err)
	}
}

func TestEvaluateMemoryReturnsUnassessableWhenObservationMissing(t *testing.T) {
	policy := MemoryPolicy{
		DegradedBelowPercent: 20,
		CriticalBelowPercent: 10,
	}

	snapshot := host.Snapshot{}

	assessment := EvaluateMemory(snapshot, policy)

	if assessment.Subject != "memory" {
		t.Fatalf("Subject = %q, want memory", assessment.Subject)
	}

	if assessment.Availability != Unassessable {
		t.Fatalf(
			"Availability = %q, want %q",
			assessment.Availability,
			Unassessable,
		)
	}

	if assessment.Status != "" {
		t.Fatalf("Status = %q, want empty", assessment.Status)
	}

	if assessment.Reason != "memory observation is unavailable" {
		t.Fatalf(
			"Reason = %q, want %q",
			assessment.Reason,
			"memory observation is unavailable",
		)
	}

	if err := assessment.Validate(); err != nil {
		t.Fatalf("Assessment.Validate() error = %v", err)
	}
}

func TestEvaluateMemoryReturnsUnassessableForInvalidObservation(t *testing.T) {
	policy := MemoryPolicy{
		DegradedBelowPercent: 20,
		CriticalBelowPercent: 10,
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 1001,
		},
	}

	assessment := EvaluateMemory(snapshot, policy)

	if assessment.Availability != Unassessable {
		t.Fatalf(
			"Availability = %q, want %q",
			assessment.Availability,
			Unassessable,
		)
	}

	if assessment.Status != "" {
		t.Fatalf("Status = %q, want empty", assessment.Status)
	}

	if err := assessment.Validate(); err != nil {
		t.Fatalf("Assessment.Validate() error = %v", err)
	}
}

func TestEvaluateReturnsCompleteHealthyResult(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
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
			Path:            "/",
			BlockSize:       4096,
			BlocksTotal:     100,
			BlocksFree:      50,
			BlocksAvailable: 50,
			InodesTotal:     100,
			InodesFree:      50,
		},
	}

	result, err := Evaluate(snapshot, policy)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if result.Status != OK {
		t.Fatalf("Status = %q, want %q", result.Status, OK)
	}

	if result.Coverage != Complete {
		t.Fatalf(
			"Coverage = %q, want %q",
			result.Coverage,
			Complete,
		)
	}

	if len(result.Assessments) != 3 {
		t.Fatalf(
			"assessment count = %d, want 3",
			len(result.Assessments),
		)
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("Result.Validate() error = %v", err)
	}
}

func TestEvaluateReturnsCriticalResult(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 50,
		},
		Filesystem: &filesystem.Stats{
			Path:            "/",
			BlockSize:       4096,
			BlocksTotal:     100,
			BlocksFree:      50,
			BlocksAvailable: 50,
			InodesTotal:     100,
			InodesFree:      50,
		},
	}

	result, err := Evaluate(snapshot, policy)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if result.Status != Critical {
		t.Fatalf("Status = %q, want %q", result.Status, Critical)
	}

	if result.Coverage != Complete {
		t.Fatalf(
			"Coverage = %q, want %q",
			result.Coverage,
			Complete,
		)
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("Result.Validate() error = %v", err)
	}
}

func TestEvaluateReturnsPartialResultWhenMemoryIsUnavailable(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	snapshot := host.Snapshot{
		Filesystem: &filesystem.Stats{
			Path:            "/",
			BlockSize:       4096,
			BlocksTotal:     100,
			BlocksFree:      50,
			BlocksAvailable: 50,
			InodesTotal:     100,
			InodesFree:      50,
		},
	}

	result, err := Evaluate(snapshot, policy)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if result.Status != OK {
		t.Fatalf("Status = %q, want %q", result.Status, OK)
	}

	if result.Coverage != Partial {
		t.Fatalf(
			"Coverage = %q, want %q",
			result.Coverage,
			Partial,
		)
	}

	if len(result.Assessments) != 3 {
		t.Fatalf(
			"assessment count = %d, want 3",
			len(result.Assessments),
		)
	}

	if result.Assessments[0].Subject != "memory" {
		t.Fatalf(
			"first assessment subject = %q, want memory",
			result.Assessments[0].Subject,
		)
	}

	if result.Assessments[0].Availability != Unassessable {
		t.Fatalf(
			"memory availability = %q, want %q",
			result.Assessments[0].Availability,
			Unassessable,
		)
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("Result.Validate() error = %v", err)
	}
}

func TestEvaluateRejectsInvalidPolicy(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 10,
			CriticalBelowPercent: 20,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	_, err := Evaluate(host.Snapshot{}, policy)
	if err == nil {
		t.Fatal("Evaluate() error = nil, want invalid policy error")
	}
}
