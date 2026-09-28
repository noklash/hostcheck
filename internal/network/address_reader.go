package network

import (
	"fmt"
	"net"

	"github.com/jsimonetti/rtnetlink"
)

func ReadAddresses(name string) ([]Address, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("find interface %s: %w", name, err)
	}

	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("dial rtnetlink: %w", err)
	}
	defer conn.Close()

	messages, err := conn.Address.List()
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}

	addresses := make([]Address, 0)

	for _, message := range messages {
		if message.Index != uint32(iface.Index) {
			continue
		}

		if message.Attributes == nil {
			continue
		}

		ip := message.Attributes.Address
		if ip == nil {
			ip = message.Attributes.Local
		}

		if ip == nil {
			continue
		}

		addresses = append(addresses, Address{
			IP:        cloneIP(ip),
			PrefixLen: int(message.PrefixLength),
			Scope:     message.Scope,
		})
	}

	return addresses, nil
}
