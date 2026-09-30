package network

import (
	"fmt"
	"time"
)

type Network struct {
	ObservedAt time.Time
	Interfaces []Interface
	Routes     []Route
}

func ReadNetwork() (Network, error) {
	observedAt := time.Now()

	names, err := EnumerateInterfaces()
	if err != nil {
		return Network{}, fmt.Errorf("enumerate interfaces: %w", err)
	}

	interfaces := make([]Interface, 0, len(names))

	for _, name := range names {
		iface, err := ReadInterface(name)
		if err != nil {
			return Network{}, fmt.Errorf("read interface %q: %w", name, err)
		}

		addresses, err := ReadAddresses(name)
		if err != nil {
			return Network{}, fmt.Errorf("read addresses for %q: %w", name, err)
		}

		iface.Addresses = addresses
		interfaces = append(interfaces, iface)
	}

	routes, err := ReadRoutes()
	if err != nil {
		return Network{}, fmt.Errorf("read routes: %w", err)
	}

	return Network{
		ObservedAt: observedAt,
		Interfaces: interfaces,
		Routes:     routes,
	}, nil
}
