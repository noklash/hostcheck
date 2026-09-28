package network

import (
	"fmt"
	"net"

	"github.com/jsimonetti/rtnetlink"
)

func ReadRoutes() ([]Route, error) {
	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("dial rtnetlink: %w", err)
	}
	defer conn.Close()

	messages, err := conn.Route.List()
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}

	routes := make([]Route, 0, len(messages))

	for _, message := range messages {
		attributes := message.Attributes

		routes = append(routes, Route{
			Family:          message.Family,
			Destination:     cloneIP(attributes.Dst),
			PrefixLen:       message.DstLength,
			Source:          cloneIP(attributes.Src),
			SourcePrefixLen: message.SrcLength,
			Gateway:         cloneIP(attributes.Gateway),
			InterfaceIndex:  attributes.OutIface,
			Priority:        attributes.Priority,
			Table:           uint32(message.Table),
			Protocol:        message.Protocol,
			Scope:           message.Scope,
			Type:            message.Type,
			Flags:           message.Flags,
		})
	}

	return routes, nil
}

func cloneIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}

	return append(net.IP(nil), ip...)
}
