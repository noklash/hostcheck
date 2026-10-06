package health

import (
	"strings"
	"testing"

	"github.com/noklash/hostcheck/internal/network"
)

func TestNetworkInterfacePolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  NetworkInterfacePolicy
		wantErr string
	}{
		{
			name: "valid policy",
			policy: NetworkInterfacePolicy{
				Name:           "enp0s3",
				RequireCarrier: true,
			},
		},
		{
			name: "carrier not required",
			policy: NetworkInterfacePolicy{
				Name: "lo",
			},
		},
		{
			name:    "empty interface name",
			policy:  NetworkInterfacePolicy{},
			wantErr: "network interface name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("Validate() expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf(
					"Validate() error = %q, want substring %q",
					err.Error(),
					tt.wantErr,
				)
			}
		})
	}
}

func TestEvaluateNetworkInterface(t *testing.T) {
	tests := []struct {
		name             string
		observation      network.Network
		policy           NetworkInterfacePolicy
		wantStatus       Status
		wantAvailability Availability
		wantReason       string
		wantEvidence     string
	}{
		{
			name: "operational interface with carrier",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Name:      "enp0s3",
						OperState: "up",
						Carrier:   true,
					},
				},
			},
			policy: NetworkInterfacePolicy{
				Name:           "enp0s3",
				RequireCarrier: true,
			},
			wantStatus:       OK,
			wantAvailability: Assessable,
			wantReason:       "the expected network interface is operational",
			wantEvidence:     "interface=enp0s3 oper_state=up carrier=true",
		},
		{
			name: "operational interface without carrier when carrier is required",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Name:      "enp0s3",
						OperState: "up",
						Carrier:   false,
					},
				},
			},
			policy: NetworkInterfacePolicy{
				Name:           "enp0s3",
				RequireCarrier: true,
			},
			wantStatus:       Degraded,
			wantAvailability: Assessable,
			wantReason:       "an expected network interface is operational without carrier",
			wantEvidence:     "interface=enp0s3 oper_state=up carrier=false",
		},
		{
			name: "operational interface without carrier when carrier is not required",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Name:      "virtual0",
						OperState: "up",
						Carrier:   false,
					},
				},
			},
			policy: NetworkInterfacePolicy{
				Name: "virtual0",
			},
			wantStatus:       OK,
			wantAvailability: Assessable,
			wantReason:       "the expected network interface is operational",
			wantEvidence:     "interface=virtual0 oper_state=up carrier=false",
		},
		{
			name: "interface is down",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Name:      "enp0s3",
						OperState: "down",
						Carrier:   false,
					},
				},
			},
			policy: NetworkInterfacePolicy{
				Name:           "enp0s3",
				RequireCarrier: true,
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "an expected network interface is not operational",
			wantEvidence:     "interface=enp0s3 oper_state=down carrier=false",
		},
		{
			name: "interface has unknown operational state",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Name:      "enp0s3",
						OperState: "unknown",
						Carrier:   true,
					},
				},
			},
			policy: NetworkInterfacePolicy{
				Name: "enp0s3",
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "an expected network interface is not operational",
			wantEvidence:     "interface=enp0s3 oper_state=unknown carrier=true",
		},
		{
			name: "expected interface is missing",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Name:      "lo",
						OperState: "unknown",
						Carrier:   false,
					},
				},
			},
			policy: NetworkInterfacePolicy{
				Name: "enp0s3",
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "an expected network interface is missing",
			wantEvidence:     "missing_interface=enp0s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := EvaluateNetworkInterface(
				tt.observation,
				tt.policy,
			)

			if err != nil {
				t.Fatalf("EvaluateNetworkInterface() unexpected error: %v", err)
			}

			if assessment.Subject != "network_interface" {
				t.Fatalf(
					"Subject = %q, want %q",
					assessment.Subject,
					"network_interface",
				)
			}

			if assessment.Availability != tt.wantAvailability {
				t.Fatalf(
					"Availability = %q, want %q",
					assessment.Availability,
					tt.wantAvailability,
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

func TestEvaluateNetworkInterfaceInvalidPolicy(t *testing.T) {
	_, err := EvaluateNetworkInterface(
		network.Network{},
		NetworkInterfacePolicy{},
	)

	if err == nil {
		t.Fatal("EvaluateNetworkInterface() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "network interface policy") {
		t.Fatalf(
			"error = %q, want network interface policy context",
			err.Error(),
		)
	}
}
