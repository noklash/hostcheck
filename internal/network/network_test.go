package network

import (
	"testing"
)

func TestReadNetwork(t *testing.T) {
	network, err := ReadNetwork()
	if err != nil {
		t.Fatalf("ReadNetwork() error = %v", err)
	}

	if len(network.Interfaces) == 0 {
		t.Fatal("ReadNetwork() returned no interfaces")
	}

	if len(network.Routes) == 0 {
		t.Fatal("ReadNetwork() returned no routes")
	}

	for _, iface := range network.Interfaces {
		if iface.Index == 0 {
			t.Errorf("interface %q has invalid index 0", iface.Name)
		}

		for _, address := range iface.Addresses {
			if address.IP == nil {
				t.Errorf("interface %q has nil address", iface.Name)
			}
		}
	}
}

func TestReadNetworkRouteInterfaceIdentity(t *testing.T) {
	network, err := ReadNetwork()
	if err != nil {
		t.Fatalf("ReadNetwork() error = %v", err)
	}

	interfaces := make(map[uint32]string, len(network.Interfaces))

	for _, iface := range network.Interfaces {
		interfaces[iface.Index] = iface.Name
	}

	for _, route := range network.Routes {
		if route.InterfaceIndex == 0 {
			continue
		}

		ifaceName, ok := interfaces[route.InterfaceIndex]
		if !ok {
			t.Errorf(
				"route references unknown interface index %d",
				route.InterfaceIndex,
			)
			continue
		}

		if ifaceName == "" {
			t.Errorf(
				"route interface index %d resolved to empty interface name",
				route.InterfaceIndex,
			)
		}
	}
}
