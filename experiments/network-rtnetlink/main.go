package main

import (
	"fmt"
	"log"
	"net"

	"github.com/jsimonetti/rtnetlink"
	"golang.org/x/sys/unix"
)

func main() {
	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		log.Fatalf("dial rtnetlink: %v", err)
	}
	defer conn.Close()

	if err := printAddresses(conn); err != nil {
		log.Fatalf("read addresses: %v", err)
	}

	if err := printRoutes(conn); err != nil {
		log.Fatalf("read routes: %v", err)
	}
}

func printAddresses(conn *rtnetlink.Conn) error {
	fmt.Println("=== ADDRESSES ===")

	addresses, err := conn.Address.List()
	if err != nil {
		return fmt.Errorf("list addresses: %w", err)
	}

	for _, addr := range addresses {
		fmt.Printf(
			"family=%s index=%d prefix=%d scope=%d flags=%d\n",
			familyName(addr.Family),
			addr.Index,
			addr.PrefixLength,
			addr.Scope,
			addr.Flags,
		)

		if addr.Attributes == nil {
			continue
		}

		fmt.Printf("  address=%s\n", ipString(addr.Attributes.Address))
		fmt.Printf("  local=%s\n", ipString(addr.Attributes.Local))
		fmt.Printf("  broadcast=%s\n", ipString(addr.Attributes.Broadcast))
		fmt.Printf("  label=%s\n", addr.Attributes.Label)
	}

	return nil
}

func printRoutes(conn *rtnetlink.Conn) error {
	fmt.Println()
	fmt.Println("=== ROUTES ===")

	routes, err := conn.Route.List()
	if err != nil {
		return fmt.Errorf("list routes: %w", err)
	}

	for _, route := range routes {
		fmt.Printf(
			"family=%s dst=%s/%d src=%s/%d table=%d protocol=%d scope=%d type=%d flags=%d\n",
			familyName(route.Family),
			ipString(route.Attributes.Dst),
			route.DstLength,
			ipString(route.Attributes.Src),
			route.SrcLength,
			route.Table,
			route.Protocol,
			route.Scope,
			route.Type,
			route.Flags,
		)

		fmt.Printf("  gateway=%s\n", ipString(route.Attributes.Gateway))
		fmt.Printf("  oif=%d\n", route.Attributes.OutIface)
		fmt.Printf("  priority=%d\n", route.Attributes.Priority)
		fmt.Printf("  table-attr=%d\n", route.Attributes.Table)
		fmt.Printf("  multipath=%d\n", len(route.Attributes.Multipath))
	}

	return nil
}

func familyName(family uint8) string {
	switch family {
	case unix.AF_INET:
		return "IPv4"
	case unix.AF_INET6:
		return "IPv6"
	default:
		return fmt.Sprintf("family-%d", family)
	}
}

func ipString(ip net.IP) string {
	if ip == nil {
		return "<none>"
	}

	return ip.String()
}
