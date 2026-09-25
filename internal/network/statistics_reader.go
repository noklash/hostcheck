package network

import (
	"fmt"
	"strconv"
)

func ReadStatistics(name string) (Statistics, error) {
	var stats Statistics

	fields := []struct {
		name string
		dest *uint64
	}{
		{"rx_bytes", &stats.RXBytes},
		{"rx_packets", &stats.RXPackets},
		{"rx_errors", &stats.RXErrors},
		{"rx_dropped", &stats.RXDropped},
		{"tx_bytes", &stats.TXBytes},
		{"tx_packets", &stats.TXPackets},
		{"tx_errors", &stats.TXErrors},
		{"tx_dropped", &stats.TXDropped},
	}

	for _, field := range fields {
		value, err := readRequiredString(
			name,
			"statistics/"+field.name,
		)
		if err != nil {
			return Statistics{}, err
		}

		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return Statistics{}, fmt.Errorf(
				"parse %s/statistics/%s value %q: %w",
				name,
				field.name,
				value,
				err,
			)
		}

		*field.dest = n
	}

	return stats, nil
}
