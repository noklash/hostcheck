package main

import (
	"fmt"
	"net"
)

func main() {
	values := []string{
		"127.0.0.1",
		"::1",
		"10.0.2.15",
		"172.16.1.10",
		"192.168.1.10",
		"8.8.8.8",
		"fd17:625c:f037:2:a00:27ff:fed0:dc4d",
		"fe80::a00:27ff:fed0:dc4d",
		"2001:4860:4860::8888",
	}

	for _, value := range values {
		ip := net.ParseIP(value)
		if ip == nil {
			panic("invalid IP: " + value)
		}

		fmt.Printf(
			"%-45s loopback=%-5t private=%-5t linklocal=%-5t globalunicast=%-5t\n",
			value,
			ip.IsLoopback(),
			ip.IsPrivate(),
			ip.IsLinkLocalUnicast(),
			ip.IsGlobalUnicast(),
		)
	}
}
