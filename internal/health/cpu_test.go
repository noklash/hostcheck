package health

import "testing"

func TestEvaluateCPUUtilization(t *testing.T) {
	policy := CPUUtilizationPolicy{
		DegradedAbovePercent: 80,
		CriticalAbovePercent: 95,
	}

	tests := []struct {
		name        string
		utilization float64
		wantStatus  Status
	}{
		{
			name:        "healthy below degraded threshold",
			utilization: 50,
			wantStatus:  OK,
		},
		{
			name:        "healthy just below degraded threshold",
			utilization: 79.99,
			wantStatus:  OK,
		},
		{
			name:        "degraded at degraded threshold",
			utilization: 80,
			wantStatus:  Degraded,
		},
		{
			name:        "degraded below critical threshold",
			utilization: 90,
			wantStatus:  Degraded,
		},
		{
			name:        "critical at critical threshold",
			utilization: 95,
			wantStatus:  Critical,
		},
		{
			name:        "critical above critical threshold",
			utilization: 99.5,
			wantStatus:  Critical,
		},
		{
			name:        "zero utilization",
			utilization: 0,
			wantStatus:  OK,
		},
		{
			name:        "full utilization",
			utilization: 100,
			wantStatus:  Critical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := EvaluateCPUUtilization(
				tt.utilization,
				policy,
			)
			if err != nil {
				t.Fatalf(
					"EvaluateCPUUtilization() error = %v",
					err,
				)
			}

			if assessment.Subject != "cpu" {
				t.Fatalf(
					"Subject = %q, want %q",
					assessment.Subject,
					"cpu",
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

			if err := assessment.Validate(); err != nil {
				t.Fatalf(
					"Assessment.Validate() error = %v",
					err,
				)
			}

			if len(assessment.Evidence) != 1 {
				t.Fatalf(
					"len(Evidence) = %d, want 1",
					len(assessment.Evidence),
				)
			}
		})
	}
}

func TestEvaluateCPUUtilizationRejectsNegativeUtilization(t *testing.T) {
	policy := CPUUtilizationPolicy{
		DegradedAbovePercent: 80,
		CriticalAbovePercent: 95,
	}

	_, err := EvaluateCPUUtilization(-1, policy)
	if err == nil {
		t.Fatal(
			"EvaluateCPUUtilization() error = nil, want invalid utilization error",
		)
	}
}

func TestEvaluateCPUUtilizationRejectsUtilizationAbove100(t *testing.T) {
	policy := CPUUtilizationPolicy{
		DegradedAbovePercent: 80,
		CriticalAbovePercent: 95,
	}

	_, err := EvaluateCPUUtilization(100.01, policy)
	if err == nil {
		t.Fatal(
			"EvaluateCPUUtilization() error = nil, want invalid utilization error",
		)
	}
}

func TestEvaluateCPUUtilizationRejectsInvalidPolicy(t *testing.T) {
	policy := CPUUtilizationPolicy{
		DegradedAbovePercent: 95,
		CriticalAbovePercent: 80,
	}

	_, err := EvaluateCPUUtilization(50, policy)
	if err == nil {
		t.Fatal(
			"EvaluateCPUUtilization() error = nil, want invalid policy error",
		)
	}
}

func TestEvaluateCPUUtilizationRejectsNegativeDegradedThreshold(t *testing.T) {
	policy := CPUUtilizationPolicy{
		DegradedAbovePercent: -1,
		CriticalAbovePercent: 95,
	}

	_, err := EvaluateCPUUtilization(50, policy)
	if err == nil {
		t.Fatal(
			"EvaluateCPUUtilization() error = nil, want invalid policy error",
		)
	}
}

func TestEvaluateCPUUtilizationRejectsCriticalThresholdAbove100(t *testing.T) {
	policy := CPUUtilizationPolicy{
		DegradedAbovePercent: 80,
		CriticalAbovePercent: 101,
	}

	_, err := EvaluateCPUUtilization(50, policy)
	if err == nil {
		t.Fatal(
			"EvaluateCPUUtilization() error = nil, want invalid policy error",
		)
	}
}
