package health

import (
	"testing"

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
