package health

import "testing"

func TestAggregate(t *testing.T) {
	tests := []struct {
		name         string
		assessments  []Assessment
		wantStatus   Status
		wantCoverage Coverage
	}{
		{
			name: "all OK",
			assessments: []Assessment{
				{
					Subject:      "memory",
					Availability: Assessable,
					Status:       OK,
				},
				{
					Subject:      "filesystem",
					Availability: Assessable,
					Status:       OK,
				},
			},
			wantStatus:   OK,
			wantCoverage: Complete,
		},
		{
			name: "degraded dominates OK",
			assessments: []Assessment{
				{
					Subject:      "memory",
					Availability: Assessable,
					Status:       OK,
				},
				{
					Subject:      "filesystem",
					Availability: Assessable,
					Status:       Degraded,
				},
			},
			wantStatus:   Degraded,
			wantCoverage: Complete,
		},
		{
			name: "critical dominates degraded",
			assessments: []Assessment{
				{
					Subject:      "memory",
					Availability: Assessable,
					Status:       Degraded,
				},
				{
					Subject:      "filesystem",
					Availability: Assessable,
					Status:       Critical,
				},
			},
			wantStatus:   Critical,
			wantCoverage: Complete,
		},
		{
			name: "unassessable with healthy assessment is partial",
			assessments: []Assessment{
				{
					Subject:      "memory",
					Availability: Assessable,
					Status:       OK,
				},
				{
					Subject:      "filesystem",
					Availability: Unassessable,
					Reason:       "filesystem observation is unavailable",
				},
			},
			wantStatus:   OK,
			wantCoverage: Partial,
		},
		{
			name: "critical remains critical with incomplete coverage",
			assessments: []Assessment{
				{
					Subject:      "memory",
					Availability: Assessable,
					Status:       Critical,
				},
				{
					Subject:      "filesystem",
					Availability: Unassessable,
					Reason:       "filesystem observation is unavailable",
				},
			},
			wantStatus:   Critical,
			wantCoverage: Partial,
		},
		{
			name: "all unassessable",
			assessments: []Assessment{
				{
					Subject:      "memory",
					Availability: Unassessable,
					Reason:       "memory observation is unavailable",
				},
				{
					Subject:      "filesystem",
					Availability: Unassessable,
					Reason:       "filesystem observation is unavailable",
				},
			},
			wantStatus:   "",
			wantCoverage: Unavailable,
		},
		{
			name:         "no assessments",
			assessments:  nil,
			wantStatus:   "",
			wantCoverage: Unavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Aggregate(tt.assessments)
			if err != nil {
				t.Fatalf("Aggregate() error = %v", err)
			}

			if result.Status != tt.wantStatus {
				t.Fatalf(
					"Status = %q, want %q",
					result.Status,
					tt.wantStatus,
				)
			}

			if result.Coverage != tt.wantCoverage {
				t.Fatalf(
					"Coverage = %q, want %q",
					result.Coverage,
					tt.wantCoverage,
				)
			}

			if err := result.Validate(); err != nil {
				t.Fatalf("Result.Validate() error = %v", err)
			}
		})
	}
}

func TestAggregateRejectsInvalidAssessment(t *testing.T) {
	_, err := Aggregate([]Assessment{
		{
			Subject:      "memory",
			Availability: Assessable,
		},
	})

	if err == nil {
		t.Fatal("Aggregate() error = nil, want invalid assessment error")
	}
}

func TestResultValidate(t *testing.T) {
	tests := []struct {
		name    string
		result  Result
		wantErr bool
	}{
		{
			name: "complete result",
			result: Result{
				Status:   OK,
				Coverage: Complete,
				Assessments: []Assessment{
					{
						Subject:      "memory",
						Availability: Assessable,
						Status:       OK,
					},
				},
			},
		},
		{
			name: "partial result",
			result: Result{
				Status:   Critical,
				Coverage: Partial,
				Assessments: []Assessment{
					{
						Subject:      "memory",
						Availability: Assessable,
						Status:       Critical,
					},
					{
						Subject:      "filesystem",
						Availability: Unassessable,
						Reason:       "filesystem observation is unavailable",
					},
				},
			},
		},
		{
			name: "unavailable result",
			result: Result{
				Coverage: Unavailable,
				Assessments: []Assessment{
					{
						Subject:      "memory",
						Availability: Unassessable,
						Reason:       "memory observation is unavailable",
					},
				},
			},
		},
		{
			name: "invalid status for unavailable result",
			result: Result{
				Status:   Critical,
				Coverage: Unavailable,
			},
			wantErr: true,
		},
		{
			name: "missing status for complete result",
			result: Result{
				Coverage: Complete,
				Assessments: []Assessment{
					{
						Subject:      "memory",
						Availability: Assessable,
						Status:       OK,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty complete result",
			result: Result{
				Status:   OK,
				Coverage: Complete,
			},
			wantErr: true,
		},
		{
			name: "invalid coverage",
			result: Result{
				Status:   OK,
				Coverage: "unknown",
			},
			wantErr: true,
		},
		{
			name: "invalid contained assessment",
			result: Result{
				Status:   OK,
				Coverage: Complete,
				Assessments: []Assessment{
					{
						Subject:      "memory",
						Availability: Assessable,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}
		})
	}
}
