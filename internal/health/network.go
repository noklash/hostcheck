package health

import (
	"fmt"
	"net"

	"github.com/noklash/hostcheck/internal/network"
	"golang.org/x/sys/unix"
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

// NetworkRoutePolicy defines one expected route.
//
// Destination nil with PrefixLen 0 represents a default route.
//
// InterfaceName identifies the interface expected to provide the route.
// Table and Priority are optional. When supplied, they further constrain the
// expected route.
//
// The policy evaluates route configuration only. It does not establish
// gateway reachability, packet delivery, DNS resolution, Internet access, or
// remote service availability.
type NetworkRoutePolicy struct {
	Family        uint8
	Destination   net.IP
	PrefixLen     uint8
	InterfaceName string
	Table         *uint32
	Priority      *uint32
}

// Validate validates an expected network route policy.
func (p NetworkRoutePolicy) Validate() error {
	switch p.Family {
	case unix.AF_INET:
		if p.PrefixLen > 32 {
			return fmt.Errorf(
				"IPv4 network route prefix length %d exceeds 32",
				p.PrefixLen,
			)
		}

		if p.Destination != nil && p.Destination.To4() == nil {
			return fmt.Errorf(
				"IPv4 network route destination must be IPv4",
			)
		}
	case unix.AF_INET6:
		if p.PrefixLen > 128 {
			return fmt.Errorf(
				"IPv6 network route prefix length %d exceeds 128",
				p.PrefixLen,
			)
		}

		if p.Destination != nil && p.Destination.To4() != nil {
			return fmt.Errorf(
				"IPv6 network route destination must be IPv6",
			)
		}
	default:
		return fmt.Errorf(
			"unsupported network route family %d",
			p.Family,
		)
	}

	if p.InterfaceName == "" {
		return fmt.Errorf("network route interface name cannot be empty")
	}

	return nil
}

// EvaluateNetworkRoute evaluates one expected route against a collected
// network observation.
//
// A route is considered matching when its family, destination, prefix length,
// and optional table and priority match the policy.
//
// The expected interface must also be associated with the route. For
// multipath routes, any next hop using the expected interface satisfies the
// interface requirement.
//
// The evaluator does not determine which route the kernel would select for
// arbitrary traffic and does not establish end-to-end reachability.
func EvaluateNetworkRoute(
	networkObservation network.Network,
	policy NetworkRoutePolicy,
) (Assessment, error) {
	if err := policy.Validate(); err != nil {
		return Assessment{}, fmt.Errorf(
			"network route policy: %w",
			err,
		)
	}

	var expectedInterface *network.Interface

	for i := range networkObservation.Interfaces {
		if networkObservation.Interfaces[i].Name == policy.InterfaceName {
			expectedInterface = &networkObservation.Interfaces[i]
			break
		}
	}

	if expectedInterface == nil {
		return Assessment{
			Subject:      "network_route",
			Availability: Assessable,
			Status:       Critical,
			Reason:       "the interface required by an expected network route is missing",
			Evidence: []string{
				fmt.Sprintf(
					"missing_interface=%s",
					policy.InterfaceName,
				),
			},
		}, nil
	}

	var matchingRoute *network.Route
	var sameRouteDifferentInterface *network.Route

	for i := range networkObservation.Routes {
		route := &networkObservation.Routes[i]

		if !networkRouteMatches(route, policy) {
			continue
		}

		if routeUsesInterface(*route, expectedInterface.Index) {
			matchingRoute = route
			break
		}

		sameRouteDifferentInterface = route
	}

	if matchingRoute == nil {
		if sameRouteDifferentInterface != nil {
			return Assessment{
				Subject:      "network_route",
				Availability: Assessable,
				Status:       Critical,
				Reason:       "the expected route exists but uses an unexpected interface",
				Evidence: []string{
					fmt.Sprintf(
						"route=%s prefix_len=%d expected_interface=%s",
						routeDestinationEvidence(*sameRouteDifferentInterface),
						sameRouteDifferentInterface.PrefixLen,
						policy.InterfaceName,
					),
				},
			}, nil
		}

		return Assessment{
			Subject:      "network_route",
			Availability: Assessable,
			Status:       Critical,
			Reason:       "an expected network route is missing",
			Evidence: []string{
				fmt.Sprintf(
					"route=%s prefix_len=%d interface=%s",
					policyDestinationEvidence(policy),
					policy.PrefixLen,
					policy.InterfaceName,
				),
			},
		}, nil
	}

	return Assessment{
		Subject:      "network_route",
		Availability: Assessable,
		Status:       OK,
		Reason:       "the expected network route is present",
		Evidence: []string{
			fmt.Sprintf(
				"route=%s prefix_len=%d interface=%s",
				routeDestinationEvidence(*matchingRoute),
				matchingRoute.PrefixLen,
				policy.InterfaceName,
			),
		},
	}, nil
}

func networkRouteMatches(
	route *network.Route,
	policy NetworkRoutePolicy,
) bool {
	if route.Family != policy.Family {
		return false
	}

	if route.PrefixLen != policy.PrefixLen {
		return false
	}

	if !ipsEqual(route.Destination, policy.Destination) {
		return false
	}

	if policy.Table != nil && route.Table != *policy.Table {
		return false
	}

	if policy.Priority != nil && route.Priority != *policy.Priority {
		return false
	}

	return true
}

func routeUsesInterface(
	route network.Route,
	interfaceIndex uint32,
) bool {
	if route.InterfaceIndex == interfaceIndex {
		return true
	}

	for _, nextHop := range route.Multipath {
		if nextHop.InterfaceIndex == interfaceIndex {
			return true
		}
	}

	return false
}

func ipsEqual(left, right net.IP) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return left.Equal(right)
}

func policyDestinationEvidence(policy NetworkRoutePolicy) string {
	if policy.Destination == nil {
		return "default"
	}

	return policy.Destination.String()
}

func routeDestinationEvidence(route network.Route) string {
	if route.Destination == nil {
		return "default"
	}

	return route.Destination.String()
}
