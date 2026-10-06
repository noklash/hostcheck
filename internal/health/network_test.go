package health

import (
	"net"
	"strings"
	"testing"

	"github.com/noklash/hostcheck/internal/network"
	"golang.org/x/sys/unix"
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
				t.Fatalf(
					"EvaluateNetworkInterface() unexpected error: %v",
					err,
				)
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

func TestNetworkRoutePolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  NetworkRoutePolicy
		wantErr string
	}{
		{
			name: "valid IPv4 default route",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
		},
		{
			name: "valid IPv4 network route",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				Destination:   net.ParseIP("10.0.2.0"),
				PrefixLen:     24,
				InterfaceName: "enp0s3",
			},
		},
		{
			name: "valid IPv6 default route",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET6,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
		},
		{
			name: "valid IPv6 network route",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET6,
				Destination:   net.ParseIP("fd17:625c:f037:2::"),
				PrefixLen:     64,
				InterfaceName: "enp0s3",
			},
		},
		{
			name: "empty interface name",
			policy: NetworkRoutePolicy{
				Family: unix.AF_INET,
			},
			wantErr: "network route interface name cannot be empty",
		},
		{
			name: "unsupported family",
			policy: NetworkRoutePolicy{
				Family:        99,
				InterfaceName: "enp0s3",
			},
			wantErr: "unsupported network route family",
		},
		{
			name: "IPv4 prefix too large",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     33,
				InterfaceName: "enp0s3",
			},
			wantErr: "IPv4 network route prefix length 33 exceeds 32",
		},
		{
			name: "IPv6 prefix too large",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET6,
				PrefixLen:     129,
				InterfaceName: "enp0s3",
			},
			wantErr: "IPv6 network route prefix length 129 exceeds 128",
		},
		{
			name: "IPv4 policy with IPv6 destination",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				Destination:   net.ParseIP("2001:db8::"),
				PrefixLen:     64,
				InterfaceName: "enp0s3",
			},
			wantErr: "IPv4 network route prefix length 64 exceeds 32",
		},
		{
			name: "IPv6 policy with IPv4 destination",
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET6,
				Destination:   net.ParseIP("10.0.2.0"),
				PrefixLen:     24,
				InterfaceName: "enp0s3",
			},
			wantErr: "IPv6 network route destination must be IPv6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf(
						"Validate() unexpected error: %v",
						err,
					)
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

func TestEvaluateNetworkRoute(t *testing.T) {
	table := uint32(254)
	priority := uint32(100)

	tests := []struct {
		name             string
		observation      network.Network
		policy           NetworkRoutePolicy
		wantStatus       Status
		wantAvailability Availability
		wantReason       string
		wantEvidence     string
	}{
		{
			name: "IPv4 default route on expected interface",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
				},
				Routes: []network.Route{
					{
						Family:         unix.AF_INET,
						PrefixLen:      0,
						InterfaceIndex: 2,
						Table:          254,
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
			wantStatus:       OK,
			wantAvailability: Assessable,
			wantReason:       "the expected network route is present",
			wantEvidence:     "route=default prefix_len=0 interface=enp0s3",
		},
		{
			name: "IPv4 network route on expected interface",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
				},
				Routes: []network.Route{
					{
						Family:         unix.AF_INET,
						Destination:    net.ParseIP("10.0.2.0"),
						PrefixLen:      24,
						InterfaceIndex: 2,
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				Destination:   net.ParseIP("10.0.2.0"),
				PrefixLen:     24,
				InterfaceName: "enp0s3",
			},
			wantStatus:       OK,
			wantAvailability: Assessable,
			wantReason:       "the expected network route is present",
			wantEvidence:     "route=10.0.2.0 prefix_len=24 interface=enp0s3",
		},
		{
			name: "expected route is missing",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "an expected network route is missing",
			wantEvidence:     "route=default prefix_len=0 interface=enp0s3",
		},
		{
			name: "expected route exists on wrong interface",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
					{
						Index: 3,
						Name:  "enp0s8",
					},
				},
				Routes: []network.Route{
					{
						Family:         unix.AF_INET,
						PrefixLen:      0,
						InterfaceIndex: 3,
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "the expected route exists but uses an unexpected interface",
			wantEvidence:     "route=default prefix_len=0 expected_interface=enp0s3",
		},
		{
			name: "multipath route uses expected interface",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
					{
						Index: 3,
						Name:  "enp0s8",
					},
				},
				Routes: []network.Route{
					{
						Family:    unix.AF_INET,
						PrefixLen: 0,
						Multipath: []network.NextHop{
							{
								InterfaceIndex: 3,
							},
							{
								InterfaceIndex: 2,
							},
						},
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
			wantStatus:       OK,
			wantAvailability: Assessable,
			wantReason:       "the expected network route is present",
			wantEvidence:     "route=default prefix_len=0 interface=enp0s3",
		},
		{
			name: "IPv6 default route on expected interface",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
				},
				Routes: []network.Route{
					{
						Family:         unix.AF_INET6,
						PrefixLen:      0,
						InterfaceIndex: 2,
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET6,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
			wantStatus:       OK,
			wantAvailability: Assessable,
			wantReason:       "the expected network route is present",
			wantEvidence:     "route=default prefix_len=0 interface=enp0s3",
		},
		{
			name: "table mismatch",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
				},
				Routes: []network.Route{
					{
						Family:         unix.AF_INET,
						PrefixLen:      0,
						InterfaceIndex: 2,
						Table:          100,
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
				Table:         &table,
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "an expected network route is missing",
			wantEvidence:     "route=default prefix_len=0 interface=enp0s3",
		},
		{
			name: "priority mismatch",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
				},
				Routes: []network.Route{
					{
						Family:         unix.AF_INET,
						PrefixLen:      0,
						InterfaceIndex: 2,
						Table:          254,
						Priority:       200,
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
				Table:         &table,
				Priority:      &priority,
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "an expected network route is missing",
			wantEvidence:     "route=default prefix_len=0 interface=enp0s3",
		},
		{
			name: "destination mismatch",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "enp0s3",
					},
				},
				Routes: []network.Route{
					{
						Family:         unix.AF_INET,
						Destination:    net.ParseIP("10.0.3.0"),
						PrefixLen:      24,
						InterfaceIndex: 2,
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				Destination:   net.ParseIP("10.0.2.0"),
				PrefixLen:     24,
				InterfaceName: "enp0s3",
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "an expected network route is missing",
			wantEvidence:     "route=10.0.2.0 prefix_len=24 interface=enp0s3",
		},
		{
			name: "expected interface is missing",
			observation: network.Network{
				Interfaces: []network.Interface{
					{
						Index: 2,
						Name:  "lo",
					},
				},
			},
			policy: NetworkRoutePolicy{
				Family:        unix.AF_INET,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
			wantStatus:       Critical,
			wantAvailability: Assessable,
			wantReason:       "the interface required by an expected network route is missing",
			wantEvidence:     "missing_interface=enp0s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := EvaluateNetworkRoute(
				tt.observation,
				tt.policy,
			)

			if err != nil {
				t.Fatalf(
					"EvaluateNetworkRoute() unexpected error: %v",
					err,
				)
			}

			if assessment.Subject != "network_route" {
				t.Fatalf(
					"Subject = %q, want %q",
					assessment.Subject,
					"network_route",
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

func TestEvaluateNetworkRouteInvalidPolicy(t *testing.T) {
	_, err := EvaluateNetworkRoute(
		network.Network{},
		NetworkRoutePolicy{
			Family:        99,
			InterfaceName: "enp0s3",
		},
	)

	if err == nil {
		t.Fatal("EvaluateNetworkRoute() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "network route policy") {
		t.Fatalf(
			"error = %q, want network route policy context",
			err.Error(),
		)
	}
}
