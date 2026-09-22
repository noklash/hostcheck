package process

import (
	"fmt"
	"os"
)

// ReadStats reads and parses /proc/<pid>/stat for pid.
func ReadStats(pid int64) (Stats, error) {
	if pid <= 0 {
		return Stats{}, fmt.Errorf("invalid pid %d", pid)
	}

	path := fmt.Sprintf("/proc/%d/stat", pid)

	data, err := os.ReadFile(path)
	if err != nil {
		return Stats{}, fmt.Errorf("read %s: %w", path, err)
	}

	stats, err := ParseStat(string(data))
	if err != nil {
		return Stats{}, fmt.Errorf("parse %s: %w", path, err)
	}

	if stats.PID != pid {
		return Stats{}, fmt.Errorf(
			"pid mismatch: requested %d, record contains %d",
			pid,
			stats.PID,
		)
	}

	return stats, nil
}
