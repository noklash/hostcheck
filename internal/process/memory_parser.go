package process

import (
	"fmt"
	"strconv"
	"strings"
)

const kib = uint64(1024)

// ParseStatus parses the process fields required from /proc/<pid>/status.
//
// Linux reports memory values in kB. Hostcheck converts them to bytes
// at the parsing boundary so the rest of the process subsystem has
// one consistent unit.
//
// Kthread identifies kernel threads. Kernel threads do not have the
// userspace memory fields reported for normal processes.
func ParseStatus(data string) (Status, error) {
	var (
		status       Status
		memory       Memory
		foundKthread bool
		foundMemory  uint8
	)

	for _, line := range strings.Split(data, "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		value = strings.TrimSpace(value)

		switch name {
		case "Kthread":
			kthread, err := strconv.ParseUint(value, 10, 8)
			if err != nil {
				return Status{}, fmt.Errorf("parse Kthread value %q: %w", value, err)
			}

			switch kthread {
			case 0:
				status.Kthread = false
			case 1:
				status.Kthread = true
			default:
				return Status{}, fmt.Errorf(
					"invalid Kthread value %q: expected 0 or 1",
					value,
				)
			}

			foundKthread = true

		case "VmSize":
			n, err := parseMemoryValue("VmSize", value)
			if err != nil {
				return Status{}, err
			}
			memory.VirtualBytes = n
			foundMemory |= 1 << 0

		case "VmRSS":
			n, err := parseMemoryValue("VmRSS", value)
			if err != nil {
				return Status{}, err
			}
			memory.ResidentBytes = n
			foundMemory |= 1 << 1

		case "RssAnon":
			n, err := parseMemoryValue("RssAnon", value)
			if err != nil {
				return Status{}, err
			}
			memory.AnonymousBytes = n
			foundMemory |= 1 << 2

		case "RssFile":
			n, err := parseMemoryValue("RssFile", value)
			if err != nil {
				return Status{}, err
			}
			memory.FileBackedBytes = n
			foundMemory |= 1 << 3

		case "RssShmem":
			n, err := parseMemoryValue("RssShmem", value)
			if err != nil {
				return Status{}, err
			}
			memory.SharedMemoryBytes = n
			foundMemory |= 1 << 4
		}
	}

	if !foundKthread {
		return Status{}, fmt.Errorf("missing required field: Kthread")
	}

	const requiredMemory = (1 << 5) - 1

	if status.Kthread {
		return status, nil
	}

	if foundMemory != requiredMemory {
		return Status{}, fmt.Errorf(
			"missing required memory fields: got mask %#x, want %#x",
			foundMemory,
			requiredMemory,
		)
	}

	status.Memory = &memory

	return status, nil
}

func parseMemoryValue(name, value string) (uint64, error) {
	fields := strings.Fields(value)

	if len(fields) != 2 {
		return 0, fmt.Errorf(
			"invalid %s value %q: expected '<value> kB'",
			name,
			value,
		)
	}

	if fields[1] != "kB" {
		return 0, fmt.Errorf(
			"invalid %s unit %q: expected kB",
			name,
			fields[1],
		)
	}

	kb, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"parse %s value %q: %w",
			name,
			fields[0],
			err,
		)
	}

	return kb * kib, nil
}
