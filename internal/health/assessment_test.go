package health

import "testing"

func TestAssessmentValidate(t *testing.T) {
	tests := []struct {
		name       string
		assessment Assessment
		wantErr    bool
	}{
		{
			name: "assessable requires status",
			assessment: Assessment{
				Subject:      "memory",
				Availability: Assessable,
			},
			wantErr: true,
		},
		{
			name: "assessable with ok status",
			assessment: Assessment{
				Subject:      "memory",
				Availability: Assessable,
				Status:       OK,
			},
		},
		{
			name: "assessable with degraded status",
			assessment: Assessment{
				Subject:      "memory",
				Availability: Assessable,
				Status:       Degraded,
			},
		},
		{
			name: "assessable with critical status",
			assessment: Assessment{
				Subject:      "memory",
				Availability: Assessable,
				Status:       Critical,
			},
		},
		{
			name: "unassessable cannot have status",
			assessment: Assessment{
				Subject:      "network",
				Availability: Unassessable,
				Status:       Critical,
			},
			wantErr: true,
		},
		{
			name: "unassessable without status",
			assessment: Assessment{
				Subject:      "network",
				Availability: Unassessable,
			},
		},
		{
			name: "missing subject",
			assessment: Assessment{
				Availability: Assessable,
				Status:       OK,
			},
			wantErr: true,
		},
		{
			name: "invalid availability",
			assessment: Assessment{
				Subject:      "memory",
				Availability: "unknown",
				Status:       OK,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			assessment: Assessment{
				Subject:      "memory",
				Availability: Assessable,
				Status:       "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.assessment.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("Validate() expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}
		})
	}
}
