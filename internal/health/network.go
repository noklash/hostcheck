package health

import (
	"fmt"

	"github.com/noklash/hostcheck/internal/network"
)

// NetworkInterfacePolicy defines the expected health requirements for one
// network interface.
//
// RequireCarrier should be enabled when the interface is expected to have a
// live link. It should remain disabled for interfaces where carrier state is
// not meaningful or is not required, such as some virtual interfaces.
type NetworkInterfacePolicy struct {
	Name           string
	RequireCarrier bool
}

// Validate validates an expected network interface policy.
func (p NetworkInterfacePolicy) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("network interface name cannot be empty")
	}

	return nil
}

// EvaluateNetworkInterface evaluates one expected network interface against
// a collected network observation.
//
// The interface must exist and report an operational state of "up".
// Carrier state is evaluated only when the policy explicitly requires it.
//
// The rule does not establish end-to-end network reachability.
func EvaluateNetworkInterface(
	networkObservation network.Network,
	policy NetworkInterfacePolicy,
) (Assessment, error) {
	if err := policy.Validate(); err != nil {
		return Assessment{}, fmt.Errorf(
			"network interface policy: %w",
			err,
		)
	}

	var observed *network.Interface

	for i := range networkObservation.Interfaces {
		if networkObservation.Interfaces[i].Name == policy.Name {
			observed = &networkObservation.Interfaces[i]
			break
		}
	}

	if observed == nil {
		return Assessment{
			Subject:      "network_interface",
			Availability: Assessable,
			Status:       Critical,
			Reason:       "an expected network interface is missing",
			Evidence: []string{
				fmt.Sprintf("missing_interface=%s", policy.Name),
			},
		}, nil
	}

	if observed.OperState != "up" {
		return Assessment{
			Subject:      "network_interface",
			Availability: Assessable,
			Status:       Critical,
			Reason:       "an expected network interface is not operational",
			Evidence: []string{
				fmt.Sprintf(
					"interface=%s oper_state=%s carrier=%t",
					observed.Name,
					observed.OperState,
					observed.Carrier,
				),
			},
		}, nil
	}

	if policy.RequireCarrier && !observed.Carrier {
		return Assessment{
			Subject:      "network_interface",
			Availability: Assessable,
			Status:       Degraded,
			Reason:       "an expected network interface is operational without carrier",
			Evidence: []string{
				fmt.Sprintf(
					"interface=%s oper_state=up carrier=false",
					observed.Name,
				),
			},
		}, nil
	}

	return Assessment{
		Subject:      "network_interface",
		Availability: Assessable,
		Status:       OK,
		Reason:       "the expected network interface is operational",
		Evidence: []string{
			fmt.Sprintf(
				"interface=%s oper_state=up carrier=%t",
				observed.Name,
				observed.Carrier,
			),
		},
	}, nil
}
