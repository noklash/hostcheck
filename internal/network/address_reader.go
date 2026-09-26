package network

import (
	"fmt"
	"net"
)

func ReadAddresses(name string) ([]Address, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("find interface %s: %w", name, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("read addresses for %s: %w", name, err)
	}

	addresses := make([]Address, 0, len(addrs))

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			return nil, fmt.Errorf(
				"unsupported address type for %s: %T",
				name,
				addr,
			)
		}

		prefixLen, bits := ipnet.Mask.Size()
		if prefixLen < 0 {
			return nil, fmt.Errorf(
				"invalid address mask for %s: %s",
				name,
				addr,
			)
		}

		if bits != 32 && bits != 128 {
			return nil, fmt.Errorf(
				"invalid address width for %s: %d",
				name,
				bits,
			)
		}

		addresses = append(addresses, Address{
			IP:        append(net.IP(nil), ipnet.IP...),
			PrefixLen: prefixLen,
		})
	}

	return addresses, nil
}
