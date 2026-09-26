package main

import (
	"fmt"
	"net"
)

func main() {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, iface := range interfaces {
		fmt.Printf("interface=%s\n", iface.Name)

		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Printf("  error: %v\n", err)
			continue
		}

		for _, addr := range addrs {
			fmt.Printf("  type=%T network=%s string=%s\n",
				addr,
				addr.Network(),
				addr.String(),
			)

			if ipnet, ok := addr.(*net.IPNet); ok {
				ones, bits := ipnet.Mask.Size()

				fmt.Printf("  ip=%s mask=%s prefix=%d bits=%d\n",
					ipnet.IP,
					ipnet.Mask,
					ones,
					bits,
				)
			}
		}
	}
}
