package health

import (
	"math"
	"testing"
)

func TestHealthPoliciesRejectNonFiniteThresholds(t *testing.T) {
	nonFiniteValues := []struct {
		name  string
		value float64
	}{
		{name: "NaN", value: math.NaN()},
		{name: "positive infinity", value: math.Inf(1)},
		{name: "negative infinity", value: math.Inf(-1)},
	}

	for _, value := range nonFiniteValues {
		t.Run(value.name, func(t *testing.T) {
			tests := []struct {
				name     string
				validate func() error
			}{
				{
					name: "CPU degraded threshold",
					validate: func() error {
						return (CPUUtilizationPolicy{
							DegradedAbovePercent: value.value,
							CriticalAbovePercent: 95,
						}).Validate()
					},
				},
				{
					name: "CPU critical threshold",
					validate: func() error {
						return (CPUUtilizationPolicy{
							DegradedAbovePercent: 80,
							CriticalAbovePercent: value.value,
						}).Validate()
					},
				},
				{
					name: "memory degraded threshold",
					validate: func() error {
						return (MemoryPolicy{
							DegradedBelowPercent: value.value,
							CriticalBelowPercent: 10,
						}).Validate()
					},
				},
				{
					name: "memory critical threshold",
					validate: func() error {
						return (MemoryPolicy{
							DegradedBelowPercent: 20,
							CriticalBelowPercent: value.value,
						}).Validate()
					},
				},
				{
					name: "filesystem degraded threshold",
					validate: func() error {
						return (FilesystemPolicy{
							DegradedBelowPercent: value.value,
							CriticalBelowPercent: 10,
						}).Validate()
					},
				},
				{
					name: "filesystem critical threshold",
					validate: func() error {
						return (FilesystemPolicy{
							DegradedBelowPercent: 20,
							CriticalBelowPercent: value.value,
						}).Validate()
					},
				},
				{
					name: "inode degraded threshold",
					validate: func() error {
						return (FilesystemInodePolicy{
							DegradedBelowPercent: value.value,
							CriticalBelowPercent: 10,
						}).Validate()
					},
				},
				{
					name: "inode critical threshold",
					validate: func() error {
						return (FilesystemInodePolicy{
							DegradedBelowPercent: 20,
							CriticalBelowPercent: value.value,
						}).Validate()
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					if err := tt.validate(); err == nil {
						t.Fatal("Validate() error = nil, want invalid threshold error")
					}
				})
			}
		})
	}
}

func TestEvaluateCPUUtilizationRejectsNonFiniteValues(t *testing.T) {
	policy := CPUUtilizationPolicy{
		DegradedAbovePercent: 80,
		CriticalAbovePercent: 95,
	}

	values := []struct {
		name  string
		value float64
	}{
		{name: "NaN", value: math.NaN()},
		{name: "positive infinity", value: math.Inf(1)},
		{name: "negative infinity", value: math.Inf(-1)},
	}

	for _, value := range values {
		t.Run(value.name, func(t *testing.T) {
			_, err := EvaluateCPUUtilization(value.value, policy)
			if err == nil {
				t.Fatal("EvaluateCPUUtilization() error = nil, want invalid utilization error")
			}
		})
	}
}
