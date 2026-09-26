package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Route struct {
	Interface   string
	Destination net.IP
	Gateway     net.IP
	Mask        net.IP
	Flags       uint16
	Metric      uint32
}

func decodeIPv4(value string) (net.IP, error) {
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", value, err)
	}

	var bytes [4]byte
	binary.LittleEndian.PutUint32(bytes[:], uint32(n))

	return net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3]), nil
}

func readRoutes(path string) ([]Route, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	if !scanner.Scan() {
		return nil, fmt.Errorf("missing route header")
	}

	var routes []Route

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 11 {
			return nil, fmt.Errorf("unexpected route entry: %q", scanner.Text())
		}

		destination, err := decodeIPv4(fields[1])
		if err != nil {
			return nil, fmt.Errorf("decode destination: %w", err)
		}

		gateway, err := decodeIPv4(fields[2])
		if err != nil {
			return nil, fmt.Errorf("decode gateway: %w", err)
		}

		flags, err := strconv.ParseUint(fields[3], 16, 16)
		if err != nil {
			return nil, fmt.Errorf("parse flags: %w", err)
		}

		metric, err := strconv.ParseUint(fields[6], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parse metric: %w", err)
		}

		mask, err := decodeIPv4(fields[7])
		if err != nil {
			return nil, fmt.Errorf("decode mask: %w", err)
		}

		routes = append(routes, Route{
			Interface:   fields[0],
			Destination: destination,
			Gateway:     gateway,
			Mask:        mask,
			Flags:       uint16(flags),
			Metric:      uint32(metric),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return routes, nil
}

func main() {
	routes, err := readRoutes("/proc/net/route")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, route := range routes {
		fmt.Printf(
			"interface=%s destination=%s gateway=%s mask=%s flags=0x%04x metric=%d\n",
			route.Interface,
			route.Destination,
			route.Gateway,
			route.Mask,
			route.Flags,
			route.Metric,
		)
	}
}
