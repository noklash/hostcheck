package network

import "net"

type Route struct {
	Family          uint8
	Destination     net.IP
	PrefixLen       uint8
	Source          net.IP
	SourcePrefixLen uint8
	Gateway         net.IP
	InterfaceIndex  uint32
	Priority        uint32
	Table           uint32
	Protocol        uint8
	Scope           uint8
	Type            uint8
	Flags           uint32
}
