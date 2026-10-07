package main

import (
	"testing"

	"github.com/noklash/hostcheck/internal/health"
)

func TestDefaultPolicy(t *testing.T) {
	policy := defaultPolicy()

	if err := policy.Validate(); err != nil {
		t.Fatalf("default policy should be valid: %v", err)
	}

	if policy.Memory.DegradedBelowPercent != degradedThresholdPercent {
		t.Fatalf(
			"memory degraded threshold = %.2f, want %.2f",
			policy.Memory.DegradedBelowPercent,
			degradedThresholdPercent,
		)
	}

	if policy.Memory.CriticalBelowPercent != criticalThresholdPercent {
		t.Fatalf(
			"memory critical threshold = %.2f, want %.2f",
			policy.Memory.CriticalBelowPercent,
			criticalThresholdPercent,
		)
	}

	if policy.Filesystem.DegradedBelowPercent != degradedThresholdPercent {
		t.Fatalf(
			"filesystem degraded threshold = %.2f, want %.2f",
			policy.Filesystem.DegradedBelowPercent,
			degradedThresholdPercent,
		)
	}

	if policy.Filesystem.CriticalBelowPercent != criticalThresholdPercent {
		t.Fatalf(
			"filesystem critical threshold = %.2f, want %.2f",
			policy.Filesystem.CriticalBelowPercent,
			criticalThresholdPercent,
		)
	}

	if policy.FilesystemInode.DegradedBelowPercent != degradedThresholdPercent {
		t.Fatalf(
			"filesystem inode degraded threshold = %.2f, want %.2f",
			policy.FilesystemInode.DegradedBelowPercent,
			degradedThresholdPercent,
		)
	}

	if policy.FilesystemInode.CriticalBelowPercent != criticalThresholdPercent {
		t.Fatalf(
			"filesystem inode critical threshold = %.2f, want %.2f",
			policy.FilesystemInode.CriticalBelowPercent,
			criticalThresholdPercent,
		)
	}

	if policy.ProcessState != nil {
		t.Fatal("default policy should not configure process-state health")
	}

	if len(policy.NetworkInterfaces) != 0 {
		t.Fatal("default policy should not configure network-interface health")
	}

	if len(policy.NetworkRoutes) != 0 {
		t.Fatal("default policy should not configure network-route health")
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		result   health.Result
		expected int
	}{
		{
			name: "complete healthy",
			result: health.Result{
				Status:   health.OK,
				Coverage: health.Complete,
			},
			expected: 0,
		},
		{
			name: "partial healthy",
			result: health.Result{
				Status:   health.OK,
				Coverage: health.Partial,
			},
			expected: 0,
		},
		{
			name: "complete degraded",
			result: health.Result{
				Status:   health.Degraded,
				Coverage: health.Complete,
			},
			expected: 1,
		},
		{
			name: "partial degraded",
			result: health.Result{
				Status:   health.Degraded,
				Coverage: health.Partial,
			},
			expected: 1,
		},
		{
			name: "complete critical",
			result: health.Result{
				Status:   health.Critical,
				Coverage: health.Complete,
			},
			expected: 2,
		},
		{
			name: "partial critical",
			result: health.Result{
				Status:   health.Critical,
				Coverage: health.Partial,
			},
			expected: 2,
		},
		{
			name: "unavailable",
			result: health.Result{
				Coverage: health.Unavailable,
			},
			expected: 1,
		},
		{
			name:     "invalid result",
			result:   health.Result{},
			expected: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := exitCode(test.result)

			if got != test.expected {
				t.Fatalf("exitCode() = %d, want %d", got, test.expected)
			}
		})
	}
}
