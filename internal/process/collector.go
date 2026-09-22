package process

import (
	"errors"
	"fmt"
	"os"
)

// Collect returns process statistics for processes visible during collection.
//
// The /proc filesystem is live and processes may exit between enumeration
// and stat collection. A process that disappears during collection is skipped.
func Collect() ([]Stats, error) {
	pids, err := ListPIDs()
	if err != nil {
		return nil, fmt.Errorf("enumerate processes: %w", err)
	}

	stats := make([]Stats, 0, len(pids))

	for _, pid := range pids {
		processStats, err := ReadStats(pid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("collect process %d: %w", pid, err)
		}

		stats = append(stats, processStats)
	}

	return stats, nil
}
