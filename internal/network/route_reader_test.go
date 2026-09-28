package network

import (
	"net"
	"testing"

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
