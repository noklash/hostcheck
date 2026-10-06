package health

import (
	"strings"
	"testing"

	"github.com/noklash/hostcheck/internal/process"
)

func TestProcessStatePolicyValidate(t *testing.T) {
	tests := []struct {
		name   string
		policy ProcessStatePolicy
	}{
		{
			name: "valid",
			policy: ProcessStatePolicy{
				DegradedAtOrAbove: 2,
				CriticalAtOrAbove: 5,
			},
		},
		{
			name: "zero degraded threshold",
			policy: ProcessStatePolicy{
				DegradedAtOrAbove: 0,
				CriticalAtOrAbove: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.policy.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestProcessStatePolicyValidateRejectsInvalidPolicies(t *testing.T) {
	tests := []struct {
		name   string
		policy ProcessStatePolicy
		want   string
	}{
		{
			name: "negative degraded threshold",
			policy: ProcessStatePolicy{
				DegradedAtOrAbove: -1,
				CriticalAtOrAbove: 5,
			},
			want: "degraded process state threshold cannot be negative",
		},
		{
			name: "negative critical threshold",
			policy: ProcessStatePolicy{
				DegradedAtOrAbove: 1,
				CriticalAtOrAbove: -1,
			},
			want: "critical process state threshold cannot be negative",
		},
		{
			name: "equal thresholds",
			policy: ProcessStatePolicy{
				DegradedAtOrAbove: 5,
				CriticalAtOrAbove: 5,
			},
			want: "critical process state threshold must be greater than degraded threshold",
		},
		{
			name: "critical below degraded",
			policy: ProcessStatePolicy{
				DegradedAtOrAbove: 5,
				CriticalAtOrAbove: 2,
			},
			want: "critical process state threshold must be greater than degraded threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf(
					"Validate() error = %q, want substring %q",
					err,
					tt.want,
				)
			}
		})
	}
}

func TestEvaluateProcessState(t *testing.T) {
	policy := ProcessStatePolicy{
		DegradedAtOrAbove: 2,
		CriticalAtOrAbove: 5,
	}

	tests := []struct {
		name         string
		processes    []process.Process
		wantStatus   Status
		wantReason   string
		wantEvidence string
	}{
		{
			name: "no D-state processes",
			processes: []process.Process{
				{
					Stats: process.Stats{
						PID:   100,
						State: 'R',
					},
				},
				{
					Stats: process.Stats{
						PID:   101,
						State: 'S',
					},
				},
			},
			wantStatus:   OK,
			wantReason:   "no processes were observed in uninterruptible sleep",
			wantEvidence: "d_state_processes=0",
		},
		{
			name: "below degraded threshold",
			processes: []process.Process{
				{
					Stats: process.Stats{
						PID:   100,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   101,
						State: 'S',
					},
				},
			},
			wantStatus:   OK,
			wantReason:   "no processes were observed in uninterruptible sleep",
			wantEvidence: "d_state_processes=1",
		},
		{
			name: "at degraded threshold",
			processes: []process.Process{
				{
					Stats: process.Stats{
						PID:   100,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   101,
						State: 'D',
					},
				},
			},
			wantStatus:   Degraded,
			wantReason:   "observed D-state process count is at or above the degraded threshold",
			wantEvidence: "d_state_processes=2",
		},
		{
			name: "below critical threshold",
			processes: []process.Process{
				{
					Stats: process.Stats{
						PID:   100,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   101,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   102,
						State: 'D',
					},
				},
			},
			wantStatus:   Degraded,
			wantReason:   "observed D-state process count is at or above the degraded threshold",
			wantEvidence: "d_state_processes=3",
		},
		{
			name: "at critical threshold",
			processes: []process.Process{
				{
					Stats: process.Stats{
						PID:   100,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   101,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   102,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   103,
						State: 'D',
					},
				},
				{
					Stats: process.Stats{
						PID:   104,
						State: 'D',
					},
				},
			},
			wantStatus:   Critical,
			wantReason:   "observed D-state process count is at or above the critical threshold",
			wantEvidence: "d_state_processes=5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := EvaluateProcessState(tt.processes, policy)
			if err != nil {
				t.Fatalf("EvaluateProcessState() error = %v", err)
			}

			if assessment.Subject != "process_state" {
				t.Fatalf(
					"Subject = %q, want %q",
					assessment.Subject,
					"process_state",
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

			if len(assessment.Evidence) != 1 {
				t.Fatalf(
					"Evidence length = %d, want 1",
					len(assessment.Evidence),
				)
			}

			if assessment.Evidence[0] != tt.wantEvidence {
				t.Fatalf(
					"Evidence[0] = %q, want %q",
					assessment.Evidence[0],
					tt.wantEvidence,
				)
			}
		})
	}
}

func TestEvaluateProcessStateRejectsInvalidPolicy(t *testing.T) {
	_, err := EvaluateProcessState(
		nil,
		ProcessStatePolicy{
			DegradedAtOrAbove: 5,
			CriticalAtOrAbove: 2,
		},
	)

	if err == nil {
		t.Fatal("EvaluateProcessState() error = nil, want error")
	}

	if !strings.Contains(
		err.Error(),
		"process state policy",
	) {
		t.Fatalf(
			"EvaluateProcessState() error = %q, want policy context",
			err,
		)
	}
}
