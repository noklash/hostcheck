package network

import "net"

type Address struct {
	IP        net.IP
	PrefixLen int
}
