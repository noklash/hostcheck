package network

import (
	"net"
	"testing"

	"github.com/jsimonetti/rtnetlink"
	"golang.org/x/sys/unix"
)

func TestReadRoutes(t *testing.T) {
	routes, err := ReadRoutes()
	if err != nil {
		t.Fatalf("ReadRoutes(): %v", err)
	}

	if len(routes) == 0 {
		t.Fatal("ReadRoutes() returned no routes")
	}

	for i, route := range routes {
		switch route.Family {
		case unix.AF_INET, unix.AF_INET6:
		default:
			t.Errorf("route %d has unexpected family %d", i, route.Family)
		}

		if route.PrefixLen > 128 {
			t.Errorf(
				"route %d has invalid destination prefix length %d",
				i,
				route.PrefixLen,
			)
		}

		if route.SourcePrefixLen > 128 {
			t.Errorf(
				"route %d has invalid source prefix length %d",
				i,
				route.SourcePrefixLen,
			)
		}

		if route.Destination != nil {
			if route.Family == unix.AF_INET && route.Destination.To4() == nil {
				t.Errorf(
					"route %d IPv4 destination is not IPv4: %s",
					i,
					route.Destination,
				)
			}

			if route.Family == unix.AF_INET6 && route.Destination.To4() != nil {
				t.Errorf(
					"route %d IPv6 destination is IPv4: %s",
					i,
					route.Destination,
				)
			}
		}

		if route.Gateway != nil && route.Family == unix.AF_INET &&
			route.Gateway.To4() == nil {
			t.Errorf(
				"route %d IPv4 gateway is not IPv4: %s",
				i,
				route.Gateway,
			)
		}

		if route.Gateway != nil && route.Family == unix.AF_INET6 &&
			route.Gateway.To4() != nil {
			t.Errorf(
				"route %d IPv6 gateway is IPv4: %s",
				i,
				route.Gateway,
			)
		}

		for j, nextHop := range route.Multipath {
			if nextHop.Gateway == nil {
				continue
			}

			if route.Family == unix.AF_INET && nextHop.Gateway.To4() == nil {
				t.Errorf(
					"route %d next hop %d IPv4 gateway is not IPv4: %s",
					i,
					j,
					nextHop.Gateway,
				)
			}

			if route.Family == unix.AF_INET6 && nextHop.Gateway.To4() != nil {
				t.Errorf(
					"route %d next hop %d IPv6 gateway is IPv4: %s",
					i,
					j,
					nextHop.Gateway,
				)
			}
		}
	}
}

func TestReadRoutesContainsDefaultRoute(t *testing.T) {
	routes, err := ReadRoutes()
	if err != nil {
		t.Fatalf("ReadRoutes(): %v", err)
	}

	foundIPv4 := false
	foundIPv6 := false

	for _, route := range routes {
		if route.PrefixLen != 0 || route.Destination != nil {
			continue
		}

		switch route.Family {
		case unix.AF_INET:
			foundIPv4 = true
		case unix.AF_INET6:
			foundIPv6 = true
		}
	}

	if !foundIPv4 {
		t.Error("did not find IPv4 default route")
	}

	if !foundIPv6 {
		t.Error("did not find IPv6 default route")
	}
}

func TestReadRoutesContainsExpectedIPv4Network(t *testing.T) {
	routes, err := ReadRoutes()
	if err != nil {
		t.Fatalf("ReadRoutes(): %v", err)
	}

	expectedDestination := net.ParseIP("10.0.2.0")

	for _, route := range routes {
		if route.Family != unix.AF_INET {
			continue
		}

		if route.PrefixLen != 24 {
			continue
		}

		if route.Destination.Equal(expectedDestination) {
			return
		}
	}

	t.Fatal("did not find expected 10.0.2.0/24 IPv4 route")
}

func TestRouteFromMessageMultipath(t *testing.T) {
	gateway1 := net.ParseIP("10.0.2.2")
	gateway2 := net.ParseIP("10.0.3.2")

	message := rtnetlink.RouteMessage{
		Family:     unix.AF_INET,
		DstLength:  24,
		SrcLength:  0,
		Table:      254,
		Protocol:   4,
		Scope:      0,
		Type:       1,
		Flags:      0,
		Attributes: rtnetlink.RouteAttributes{},
	}

	message.Attributes.Dst = net.ParseIP("10.0.4.0")
	message.Attributes.Multipath = []rtnetlink.NextHop{
		{
			Hop: rtnetlink.RTNextHop{
				IfIndex: 2,
				Hops:    3,
				Flags:   4,
			},
			Gateway: gateway1,
		},
		{
			Hop: rtnetlink.RTNextHop{
				IfIndex: 3,
				Hops:    7,
				Flags:   8,
			},
			Gateway: gateway2,
		},
	}

	route := routeFromMessage(message)

	if len(route.Multipath) != 2 {
		t.Fatalf(
			"expected 2 multipath next hops, got %d",
			len(route.Multipath),
		)
	}

	first := route.Multipath[0]

	if first.InterfaceIndex != 2 {
		t.Errorf(
			"first next hop interface index = %d, want 2",
			first.InterfaceIndex,
		)
	}

	if !first.Gateway.Equal(gateway1) {
		t.Errorf(
			"first next hop gateway = %v, want %v",
			first.Gateway,
			gateway1,
		)
	}

	if first.Hops != 3 {
		t.Errorf(
			"first next hop hops = %d, want 3",
			first.Hops,
		)
	}

	if first.Flags != 4 {
		t.Errorf(
			"first next hop flags = %d, want 4",
			first.Flags,
		)
	}

	second := route.Multipath[1]

	if second.InterfaceIndex != 3 {
		t.Errorf(
			"second next hop interface index = %d, want 3",
			second.InterfaceIndex,
		)
	}

	if !second.Gateway.Equal(gateway2) {
		t.Errorf(
			"second next hop gateway = %v, want %v",
			second.Gateway,
			gateway2,
		)
	}

	if second.Hops != 7 {
		t.Errorf(
			"second next hop hops = %d, want 7",
			second.Hops,
		)
	}

	if second.Flags != 8 {
		t.Errorf(
			"second next hop flags = %d, want 8",
			second.Flags,
		)
	}
}
