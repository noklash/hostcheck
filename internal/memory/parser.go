package memory

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// ParseMemInfo parses Linux /proc/meminfo output into MemInfo.
func ParseMemInfo(input string) (MemInfo, error) {
	var mem MemInfo

	var seenTotal bool
	var seenAvailable bool
	var seenFree bool
	var seenBuffers bool
	var seenCached bool
	var seenSwapTotal bool
	var seenSwapFree bool

	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "MemTotal:", "MemAvailable:", "MemFree:", "Buffers:",
			"Cached:", "SwapTotal:", "SwapFree:":
		default:
			continue
		}

		if len(fields) != 3 {
			return MemInfo{}, fmt.Errorf("invalid meminfo line %q: expected 3 fields", scanner.Text())
		}

		if fields[2] != "kB" {
			return MemInfo{}, fmt.Errorf("invalid meminfo line %q: expected kB unit", scanner.Text())
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return MemInfo{}, fmt.Errorf("invalid value in meminfo line %q: %w", scanner.Text(), err)
		}

		switch fields[0] {
		case "MemTotal:":
			if seenTotal {
				return MemInfo{}, fmt.Errorf("duplicate MemTotal field")
			}
			mem.Total = value
			seenTotal = true

		case "MemAvailable:":
			if seenAvailable {
				return MemInfo{}, fmt.Errorf("duplicate MemAvailable field")
			}
			mem.Available = value
			seenAvailable = true

		case "MemFree:":
			if seenFree {
				return MemInfo{}, fmt.Errorf("duplicate MemFree field")
			}
			mem.Free = value
			seenFree = true

		case "Buffers:":
			if seenBuffers {
				return MemInfo{}, fmt.Errorf("duplicate Buffers field")
			}
			mem.Buffers = value
			seenBuffers = true

		case "Cached:":
			if seenCached {
				return MemInfo{}, fmt.Errorf("duplicate Cached field")
			}
			mem.Cached = value
			seenCached = true

		case "SwapTotal:":
			if seenSwapTotal {
				return MemInfo{}, fmt.Errorf("duplicate SwapTotal field")
			}
			mem.SwapTotal = value
			seenSwapTotal = true

		case "SwapFree:":
			if seenSwapFree {
				return MemInfo{}, fmt.Errorf("duplicate SwapFree field")
			}
			mem.SwapFree = value
			seenSwapFree = true
		}
	}

	if err := scanner.Err(); err != nil {
		return MemInfo{}, fmt.Errorf("scan /proc/meminfo: %w", err)
	}

	if !seenTotal ||
		!seenAvailable ||
		!seenFree ||
		!seenBuffers ||
		!seenCached ||
		!seenSwapTotal ||
		!seenSwapFree {
		return MemInfo{}, fmt.Errorf("missing required meminfo field")
	}

	return mem, nil
}
