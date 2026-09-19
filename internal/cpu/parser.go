package cpu

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseStat parses the aggregate CPU line from /proc/stat.
//
// Expected format:
//
//	cpu user nice system idle iowait irq softirq steal guest guest_nice
//
// The returned counters are cumulative clock ticks.
func ParseStat(line string) (Stat, error) {
	fields := strings.Fields(line)

	if len(fields) != 11 {
		return Stat{}, fmt.Errorf("invalid CPU stat line: expected 11 fields, got %d", len(fields))
	}

	if fields[0] != "cpu" {
		return Stat{}, fmt.Errorf("invalid CPU stat line: expected aggregate CPU field, got %q", fields[0])
	}

	values := make([]uint64, 10)

	for i := 0; i < len(values); i++ {
		value, err := strconv.ParseUint(fields[i+1], 10, 64)
		if err != nil {
			return Stat{}, fmt.Errorf("invalid CPU counter %q: %w", fields[i+1], err)
		}

		values[i] = value
	}

	return Stat{
		User:      values[0],
		Nice:      values[1],
		System:    values[2],
		Idle:      values[3],
		IOWait:    values[4],
		IRQ:       values[5],
		SoftIRQ:   values[6],
		Steal:     values[7],
		Guest:     values[8],
		GuestNice: values[9],
	}, nil
}
