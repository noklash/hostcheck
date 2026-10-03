package health

import (
	"math"
	"testing"
)

func TestEvaluateMemoryAvailablePercent(t *testing.T) {
	policy := MemoryPolicy{
		DegradedBelowPercent: 20,
		CriticalBelowPercent: 10,
	}

	tests := []struct {
		name       string
		available  float64
		wantStatus Status
		wantReason string
	}{
		{
			name:       "above degraded threshold",
			available:  50,
			wantStatus: OK,
			wantReason: "available memory capacity is within the configured policy",
		},
		{
			name:       "at degraded threshold",
			available:  20,
			wantStatus: OK,
			wantReason: "available memory capacity is within the configured policy",
		},
		{
			name:       "below degraded threshold",
			available:  19.99,
			wantStatus: Degraded,
			wantReason: "available memory capacity is below the degraded policy threshold",
		},
		{
			name:       "at critical threshold",
			available:  10,
			wantStatus: Degraded,
			wantReason: "available memory capacity is below the degraded policy threshold",
		},
		{
			name:       "below critical threshold",
			available:  9.99,
			wantStatus: Critical,
			wantReason: "available memory capacity is below the critical policy threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := EvaluateMemoryAvailablePercent(
				tt.available,
				policy,
			)
			if err != nil {
				t.Fatalf(
					"EvaluateMemoryAvailablePercent() error = %v",
					err,
				)
			}

			if assessment.Subject != "memory" {
				t.Fatalf(
					"Subject = %q, want memory",
					assessment.Subject,
				)
			}

			if assessment.Availability != Assessable {
				t.Fatalf(
					"Availability = %q, want %q",
					assessment.Availability,
					Assessable,
				)
			}

			if assessment.Status != tt.wantStatus {
				t.Fatalf(
					"Status = %q, want %q",
					assessment.Status,
					tt.wantStatus,
				)
			}

			if assessment.Reason != tt.wantReason {
				t.Fatalf(
					"Reason = %q, want %q",
					assessment.Reason,
					tt.wantReason,
				)
			}

			if err := assessment.Validate(); err != nil {
				t.Fatalf(
					"Assessment.Validate() error = %v",
					err,
				)
			}
		})
	}
}

func TestEvaluateMemoryAvailablePercentRejectsInvalidObservation(t *testing.T) {
	policy := MemoryPolicy{
		DegradedBelowPercent: 20,
		CriticalBelowPercent: 10,
	}

	tests := []struct {
		name      string
		available float64
	}{
		{
			name:      "below zero",
			available: -1,
		},
		{
			name:      "above one hundred",
			available: 101,
		},
		{
			name:      "not a number",
			available: math.NaN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EvaluateMemoryAvailablePercent(
				tt.available,
				policy,
			)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestMemoryPolicyValidate(t *testing.T) {
	tests := []struct {
		name   string
		policy MemoryPolicy
	}{
		{
			name: "degraded threshold below zero",
			policy: MemoryPolicy{
				DegradedBelowPercent: -1,
				CriticalBelowPercent: 10,
			},
		},
		{
			name: "degraded threshold above one hundred",
			policy: MemoryPolicy{
				DegradedBelowPercent: 101,
				CriticalBelowPercent: 10,
			},
		},
		{
			name: "critical threshold below zero",
			policy: MemoryPolicy{
				DegradedBelowPercent: 20,
				CriticalBelowPercent: -1,
			},
		},
		{
			name: "critical threshold above one hundred",
			policy: MemoryPolicy{
				DegradedBelowPercent: 20,
				CriticalBelowPercent: 101,
			},
		},
		{
			name: "critical threshold equal to degraded threshold",
			policy: MemoryPolicy{
				DegradedBelowPercent: 10,
				CriticalBelowPercent: 10,
			},
		},
		{
			name: "critical threshold above degraded threshold",
			policy: MemoryPolicy{
				DegradedBelowPercent: 10,
				CriticalBelowPercent: 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.policy.Validate(); err == nil {
				t.Fatal("expected policy validation error, got nil")
			}
		})
	}
}
